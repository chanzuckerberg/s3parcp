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
	"github.com/chanzuckerberg/s3parcp/s3checksum"
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
	destExists, err := dest.Exists()
	if err != nil {
		return []CopyJob{}, err
	}

	isSrcDir, err := src.IsDir()
	if err != nil {
		return []CopyJob{}, err
	}

	isDestDir, err := dest.IsDir()
	if destExists && err != nil {
		return []CopyJob{}, err
	}

	if !isDestDir && isSrcDir {
		if !destExists {
			// If the destination doesn't exist and the source is a directory
			//   create a local directory. This brings local behavior in line
			//   with s3, where it is possible to upload to a non-existant folder
			//   and the folder will be created automatically.
			if dest.IsLocal() {
				err = os.MkdirAll(dest.String(), os.ModePerm)
				if err != nil {
					return []CopyJob{}, err
				}
			}

			// Since a local directory was created if necessary it can be assumed
			//   that the destination is a directory if the source was a directory
			isDestDir = true
		} else {
			dirOrFolder := "directory"
			if src.IsS3() {
				dirOrFolder = "folder"
			}
			fileOrObject := "file"
			if dest.IsS3() {
				fileOrObject = "object"
			}

			return []CopyJob{}, fmt.Errorf("cannot copy %s: %s to existing %s: %s", dirOrFolder, fileOrObject, src, dest)
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

	// TODO make configurable
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
	// Only get object info if mmap or checksum is enabled
	var headObjectResponse *s3.HeadObjectOutput
	if c.Options.Mmap || c.Options.Checksum {
		headObjectInput := s3.HeadObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		}

		var err error
		headObjectResponse, err = c.Client.HeadObject(&headObjectInput)
		if err != nil {
			return err
		}
	}

	getObjectInput := s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	err := os.MkdirAll(path.Dir(dest), os.ModePerm)
	if err != nil {
		return fmt.Errorf("while creating directory: %s encountered error: %s", path.Dir(dest), err)
	}

	// TODO make an mmap API that is compatible with os.File to avoid this branching
	if c.Options.Mmap {
		file, err := mmap.CreateFile(dest, *headObjectResponse.ContentLength)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = c.Downloader.Download(file, &getObjectInput)
		if err != nil {
			return err
		}
	} else {
		file, err := os.Create(dest)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = c.Downloader.Download(file, &getObjectInput)
		if err != nil {
			return err
		}
	}

	if c.Options.Checksum {
		checksumOptions := checksum.ParallelChecksumOptions{
			Concurrency: c.Options.Concurrency,
			PartSize:    c.Options.PartSize,
			UseMmap:     c.Options.Mmap,
		}
		expectedChecksum, err := s3checksum.GetCRC32CChecksum(headObjectResponse)
		if err != nil {
			return fmt.Errorf("while getting checksum from object: %s metadata encountered error: %s", key, err)
		}

		checksum, err := checksum.ParallelCRC32CChecksum(dest, checksumOptions)
		if err != nil {
			return fmt.Errorf("while computing checksum for file: %s encountered error: %s", dest, err)
		}

		if expectedChecksum != checksum {
			return fmt.Errorf("checksum did not match for file: %s", dest)
		}
	}

	return nil
}

func (c *Copier) upload(src string, bucket string, key string) error {
	uploadInput := s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	// Only compute checksum if it is necessary
	if c.Options.Checksum {
		checksumOptions := checksum.ParallelChecksumOptions{
			Concurrency: c.Options.Concurrency,
			PartSize:    c.Options.PartSize,
			UseMmap:     c.Options.Mmap,
		}
		crc32cChecksum, err := checksum.ParallelCRC32CChecksum(src, checksumOptions)
		if err != nil {
			return errors.New("Error computing crc32c checksum of source file")
		}
		uploadInput = s3checksum.SetCRC32CChecksum(uploadInput, crc32cChecksum)
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
	if err != nil {
		return err
	}

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
	if copyJob.source.IsS3() && copyJob.destination.IsS3() {
		return errors.New("Copying between s3 is not yet supported")
	} else if !copyJob.source.IsS3() && copyJob.destination.IsS3() {
		bucket, err := copyJob.destination.Bucket()
		if err != nil {
			return fmt.Errorf("path: %s was determined to be an s3 path but getting it's bucket encountered error: %s", copyJob.destination, err)
		}

		return c.upload(
			copyJob.source.String(),
			bucket,
			copyJob.destination.ToStringWithoutBucket(),
		)
	} else if copyJob.source.IsS3() && !copyJob.destination.IsS3() {
		bucket, err := copyJob.source.Bucket()
		if err != nil {
			return fmt.Errorf("path: %s was determined to be an s3 path but getting it's bucket encountered error: %s", copyJob.source, err)
		}

		return c.download(
			bucket,
			copyJob.source.ToStringWithoutBucket(),
			copyJob.destination.String(),
		)
	} else {
		return c.localCopy(copyJob.source.String(), copyJob.destination.String())
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
