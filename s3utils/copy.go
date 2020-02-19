package s3utils

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/chanzuckerberg/s3parcp/checksum"
	"github.com/chanzuckerberg/s3parcp/mmap"
)

// CopyJob defines a file/object copy
type CopyJob struct {
	source      Path
	destination Path
}

// NewCopyJob creates a new CopyJob
func NewCopyJob(source Path, destination Path) CopyJob {
	return CopyJob{
		source:      source,
		destination: destination,
	}
}

// GetCopyJobs gets the jobs required to copy between two paths
func GetCopyJobs(src Path, dest Path) ([]CopyJob, error) {
	// TODO deal with errors
	isSrcDir, err := src.IsDir()
	isDestDir, err := dest.IsDir()
	destExists, err := dest.Exists()

	// We don't want to create a direcory with the same name as an object but if there is
	//   already a directory with that name it's fine to put stuff in it
	if !isDestDir && isSrcDir {
		if !destExists {
			// TODO error
			if dest.IsLocal() {
				os.MkdirAll(dest.ToString(), os.ModePerm)
			}
			isDestDir = true
		} else {
			// TODO update comment
			return []CopyJob{}, errors.New("Cannot copy directory to existing object or file")
		}
	}

	srcFilepaths, err := src.ListPathsWithPrefix()
	copyJobs := make([]CopyJob, len(srcFilepaths))

	for i, srcFilepath := range srcFilepaths {
		destFilepath := dest
		if !isSrcDir && isDestDir {
			destFilepath = destFilepath.Join(src.Base())
		}
		if isSrcDir && isDestDir {
			destFilepath = destFilepath.Join(srcFilepath.WithoutPrefix(src))
		}
		copyJobs[i] = NewCopyJob(srcFilepath, destFilepath)
	}

	return copyJobs, err
}

// CopierOptions are options for a copier object
type CopierOptions struct {
	BufferSize  int
	Checksum    bool
	Concurrency int
	Mmap        bool
	PartSize    int64
}

// Copier holds state for copying
type Copier struct {
	Options    CopierOptions
	Client     *s3.S3
	Downloader *s3manager.Downloader
	Uploader   *s3manager.Uploader
}

// NewCopier creates a new Copier
func NewCopier(opts CopierOptions) Copier {
	sess := session.Must(
		session.NewSessionWithOptions(
			session.Options{
				SharedConfigState: session.SharedConfigEnable,
			},
		),
	)

	disableSSL := true
	maxRetries := 3
	client := s3.New(sess, &aws.Config{
		DisableSSL: &disableSSL,
		MaxRetries: &maxRetries,
	})

	downloader := s3manager.NewDownloader(sess, func(d *s3manager.Downloader) {
		d.PartSize = opts.PartSize
		d.Concurrency = opts.Concurrency
		d.S3 = client
		if opts.BufferSize > 0 {
			d.BufferProvider = s3manager.NewPooledBufferedWriterReadFromProvider(opts.BufferSize)
		}
	})

	uploader := s3manager.NewUploader(sess, func(d *s3manager.Uploader) {
		d.PartSize = opts.PartSize
		d.Concurrency = opts.Concurrency
		d.S3 = client
		if opts.BufferSize > 0 {
			d.BufferProvider = s3manager.NewBufferedReadSeekerWriteToPool(opts.BufferSize)
		}
	})

	return Copier{
		Client:     client,
		Downloader: downloader,
		Uploader:   uploader,
		Options:    opts,
	}
}

func (c *Copier) download(bucket string, key string, dest string) error {
	var err error = nil

	// Only get object info if mmap or checksum is enabled
	var headObjectResponse *s3.HeadObjectOutput
	if c.Options.Mmap || c.Options.Checksum {
		headObjectInput := s3.HeadObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		}

		headObjectResponse, err = c.Client.HeadObject(&headObjectInput)
	}

	getObjectInput := s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	err = os.MkdirAll(path.Dir(dest), os.ModePerm)

	// TODO make an mmap API that is compatible with os.File to avoid this branching
	if c.Options.Mmap {
		file, err := mmap.CreateFile(dest, *headObjectResponse.ContentLength)
		if err != nil {
			return err
		}
		_, downloadErr := c.Downloader.Download(file, &getObjectInput)
		err = file.Close()
		// Return the download error if encountered but we still want to close the file
		if downloadErr != nil {
			return downloadErr
		}
	} else {
		file, err := os.Create(dest)
		if err != nil {
			return err
		}
		_, downloadErr := c.Downloader.Download(file, &getObjectInput)
		err = file.Close()
		// Return the download error if encountered but we still want to close the file
		if downloadErr != nil {
			return downloadErr
		}
	}
	if err != nil {
		return err
	}

	if c.Options.Checksum {
		checksumOptions := checksum.ParallelChecksumOptions{
			Concurrency: c.Options.Concurrency,
			PartSize:    c.Options.PartSize,
			UseMmap:     c.Options.Mmap,
		}
		CompareChecksum(headObjectResponse, dest, checksumOptions)
	}

	return err
}

func (c *Copier) upload(src string, bucket string, key string) error {
	uploadInput := s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	// Only compute checksum if we need to
	if c.Options.Checksum {
		checksumOptions := checksum.ParallelChecksumOptions{
			Concurrency: c.Options.Concurrency,
			PartSize:    c.Options.PartSize,
			UseMmap:     c.Options.Mmap,
		}
		crc32cChecksum, err := checksum.ParallelCRC32CChecksum(src, checksumOptions)
		if err != nil {
			os.Stderr.WriteString("Error computing crc32c checksum of source file\n")
			panic(err)
		}
		crc32cChecksumString := fmt.Sprintf("%X", crc32cChecksum)
		metadata := make(map[string]*string)
		metadata[Crc32cChecksumMetadataName] = &crc32cChecksumString
	}

	var uploadErr error
	var fileCloseErr error
	// TODO make an mmap API that is compatible with os.File to avoid this branching
	if c.Options.Mmap {
		file, err := mmap.OpenFile(src)
		if err != nil {
			return err
		}
		uploadInput.Body = file
		_, uploadErr = c.Uploader.Upload(&uploadInput)
		fileCloseErr = file.Close()
	} else {
		file, err := os.Open(src)
		if err != nil {
			return err
		}
		uploadInput.Body = file
		_, uploadErr = c.Uploader.Upload(&uploadInput)
		fileCloseErr = file.Close()
	}

	// Return the upload error if encountered but we still want to close the file
	if uploadErr != nil {
		return uploadErr
	}

	if fileCloseErr != nil {
		return fileCloseErr
	}

	return nil
}

func (c *Copier) localCopy(src string, dest string) error {
	err := os.MkdirAll(path.Dir(dest), os.ModePerm)

	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destination.Close()
	_, err = io.Copy(destination, source)
	return err
}

// Copy executes a copy job
func (c *Copier) Copy(copyJob CopyJob) error {
	// TODO lots
	if copyJob.source.IsS3() && copyJob.destination.IsS3() {
		return errors.New("Both S3")
	} else if !copyJob.source.IsS3() && copyJob.destination.IsS3() {
		bucket, _ := copyJob.destination.Bucket()
		return c.upload(
			copyJob.source.ToString(),
			bucket,
			copyJob.destination.ToStringWithoutBucket(),
		)
	} else if copyJob.source.IsS3() && !copyJob.destination.IsS3() {
		bucket, _ := copyJob.source.Bucket()
		return c.download(
			bucket,
			copyJob.source.ToStringWithoutBucket(),
			copyJob.destination.ToString(),
		)
	} else {
		return c.localCopy(copyJob.source.ToString(), copyJob.destination.ToString())
	}
}

func copyWorker(copier *Copier, downloadJobs <-chan CopyJob, errors chan<- error) {
	for copyJob := range downloadJobs {
		errors <- copier.Copy(copyJob)
	}
}

// CopyAll executes a slice of copy Jobs
func (c *Copier) CopyAll(copyJobs []CopyJob) error {
	numJobs := len(copyJobs)
	copyJobsChannel := make(chan CopyJob, numJobs)
	errorChannel := make(chan error, numJobs)

	for w := 0; w < c.Options.Concurrency; w++ {
		go copyWorker(c, copyJobsChannel, errorChannel)
	}

	for _, copyJob := range copyJobs {
		copyJobsChannel <- copyJob
	}
	close(copyJobsChannel)

	var err error = nil
	for i := 0; i < numJobs; i++ {
		currentError := <-errorChannel
		if currentError != nil {
			err = currentError
		}
	}
	return err
}
