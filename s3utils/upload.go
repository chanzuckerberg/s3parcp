package s3utils

import (
	"fmt"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/chanzuckerberg/s3parcp/checksum"
	"github.com/chanzuckerberg/s3parcp/mmap"
)

// UploaderOptions are configuration options for Uploader
type UploaderOptions struct {
	BufferSize  int
	Checksum    bool
	Concurrency int
	Mmap        bool
	PartSize    int64
}

// Uploader is a wrapper for s3manager.Uploader with additional options and methods
type Uploader struct {
	Client   *s3.S3
	Uploader *s3manager.Uploader
	Options  UploaderOptions
}

// NewUploader creates a new Uploader
func NewUploader(opts UploaderOptions) Uploader {
	sess := session.Must(
		session.NewSessionWithOptions(
			session.Options{
				SharedConfigState: session.SharedConfigEnable,
			},
		),
	)

	// TODO make these configurable
	httpClient := &http.Client{
		Timeout: 15e9,
	}
	disableSSL := true
	maxRetries := 3
	client := s3.New(sess, &aws.Config{
		DisableSSL: &disableSSL,
		HTTPClient: httpClient,
		MaxRetries: &maxRetries,
	})

	uploader := s3manager.NewUploader(sess, func(d *s3manager.Uploader) {
		d.PartSize = opts.PartSize
		d.Concurrency = opts.Concurrency
		d.S3 = client
		if opts.BufferSize > 0 {
			d.BufferProvider = s3manager.NewBufferedReadSeekerWriteToPool(opts.BufferSize)
		}
	})

	return Uploader{
		Client:   client,
		Uploader: uploader,
		Options:  opts,
	}
}

// Upload executes a single upload job
func (d *Uploader) Upload(uploadJob UploadJob) error {
	uploadInput := s3manager.UploadInput{
		Bucket: aws.String(uploadJob.Bucket),
		Key:    aws.String(uploadJob.Key),
	}

	// Only compute checksum if we need to
	if d.Options.Checksum {
		checksumOptions := checksum.ParallelChecksumOptions{
			Concurrency: d.Options.Concurrency,
			PartSize:    d.Options.PartSize,
			UseMmap:     d.Options.Mmap,
		}
		crc32cChecksum, err := checksum.ParallelCRC32CChecksum(uploadJob.Filepath, checksumOptions)
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
	if d.Options.Mmap {
		file, err := mmap.OpenFile(uploadJob.Filepath)
		if err != nil {
			return err
		}
		uploadInput.Body = file
		_, uploadErr = d.Uploader.Upload(&uploadInput)
		fileCloseErr = file.Close()
	} else {
		file, err := os.Open(uploadJob.Filepath)
		if err != nil {
			return err
		}
		uploadInput.Body = file
		_, uploadErr = d.Uploader.Upload(&uploadInput)
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

func uploadWorker(uploader *Uploader, uploadJobs <-chan UploadJob, errors chan<- error) {
	for uploadJob := range uploadJobs {
		errors <- uploader.Upload(uploadJob)
	}
}

// UploadAll completes all of the upload jobs in a slice of upload jobs
func (d *Uploader) UploadAll(uploadJobs []UploadJob) error {
	// TODO replace this with a function that takes an iterator
	numJobs := len(uploadJobs)
	uploadJobsChannel := make(chan UploadJob, numJobs)
	errorChannel := make(chan error, numJobs)

	for w := 0; w < d.Uploader.Concurrency; w++ {
		go uploadWorker(d, uploadJobsChannel, errorChannel)
	}

	for _, uploadJob := range uploadJobs {
		uploadJobsChannel <- uploadJob
	}
	close(uploadJobsChannel)

	var err error = nil
	for i := 0; i < numJobs; i++ {
		currentError := <-errorChannel
		if currentError != nil {
			err = currentError
		}
	}
	return err
}

// UploadJob defines a single upload from S3
type UploadJob struct {
	Bucket   string
	Key      string
	Filepath string
}

// NewUploadJob creates a new upload job
func NewUploadJob(bucket string, key string, filepath string) UploadJob {
	return UploadJob{
		Bucket:   bucket,
		Key:      key,
		Filepath: filepath,
	}
}
