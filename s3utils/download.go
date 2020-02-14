package s3utils

import (
	"os"
	"path"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/chanzuckerberg/s3parcp/checksum"
	"github.com/chanzuckerberg/s3parcp/mmap"
)

// DownloaderOptions are configuration options for Downloader
type DownloaderOptions struct {
	BufferSize  int
	Checksum    bool
	Concurrency int
	Mmap        bool
	PartSize    int64
}

// Downloader is a wrapper for s3manager.Downloader with additional options and methods
type Downloader struct {
	Client     *s3.S3
	Downloader *s3manager.Downloader
	Options    DownloaderOptions
}

// NewDownloader creates a new Downloader
func NewDownloader(opts DownloaderOptions) Downloader {
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

	return Downloader{
		Client:     client,
		Downloader: downloader,
		Options:    opts,
	}
}

// Download executes a single download job
func (d *Downloader) Download(downloadJob DownloadJob) error {
	var err error = nil

	// Only get object info if mmap or checksum is enabled
	var headObjectResponse *s3.HeadObjectOutput
	if d.Options.Mmap || d.Options.Checksum {
		headObjectInput := s3.HeadObjectInput{
			Bucket: aws.String(downloadJob.Bucket),
			Key:    aws.String(downloadJob.Key),
		}

		headObjectResponse, err = d.Client.HeadObject(&headObjectInput)
	}

	getObjectInput := s3.GetObjectInput{
		Bucket: aws.String(downloadJob.Bucket),
		Key:    aws.String(downloadJob.Key),
	}

	parentDirExists, err := IsLocalDirectory(path.Dir(downloadJob.Filepath))
	if err != nil {
		panic(err)
	}
	if !parentDirExists {
		err = os.MkdirAll(path.Dir(downloadJob.Filepath), os.ModePerm)
		if err != nil {
			panic(err)
		}
	}

	// TODO make an mmap API that is compatible with os.File to avoid this branching
	if d.Options.Mmap {
		file, err := mmap.CreateFile(downloadJob.Filepath, *headObjectResponse.ContentLength)
		if err != nil {
			return err
		}
		_, downloadErr := d.Downloader.Download(file, &getObjectInput)
		err = file.Close()
		// Return the download error if encountered but we still want to close the file
		if downloadErr != nil {
			return downloadErr
		}
	} else {
		file, err := os.Create(downloadJob.Filepath)
		if err != nil {
			return err
		}
		_, downloadErr := d.Downloader.Download(file, &getObjectInput)
		err = file.Close()
		// Return the download error if encountered but we still want to close the file
		if downloadErr != nil {
			return downloadErr
		}
	}
	if err != nil {
		return err
	}

	if d.Options.Checksum {
		checksumOptions := checksum.ParallelChecksumOptions{
			Concurrency: d.Options.Concurrency,
			PartSize:    d.Options.PartSize,
			UseMmap:     d.Options.Mmap,
		}
		CompareChecksum(headObjectResponse, downloadJob.Filepath, checksumOptions)
	}

	return err
}

func downloadWorker(downloader *Downloader, downloadJobs <-chan DownloadJob, errors chan<- error) {
	for downloadJob := range downloadJobs {
		errors <- downloader.Download(downloadJob)
	}
}

// DownloadAll completes all of the download jobs in a slice of download jobs
func (d *Downloader) DownloadAll(downloadJobs []DownloadJob) error {
	// TODO replace this with a function that takes an iterator
	numJobs := len(downloadJobs)
	downloadJobsChannel := make(chan DownloadJob, numJobs)
	errorChannel := make(chan error, numJobs)

	for w := 0; w < d.Downloader.Concurrency; w++ {
		go downloadWorker(d, downloadJobsChannel, errorChannel)
	}

	for _, downloadJob := range downloadJobs {
		downloadJobsChannel <- downloadJob
	}
	close(downloadJobsChannel)

	var err error = nil
	for i := 0; i < numJobs; i++ {
		currentError := <-errorChannel
		if currentError != nil {
			err = currentError
		}
	}
	return err
}

// DownloadJob defines a single download from S3
type DownloadJob struct {
	Bucket   string
	Key      string
	Filepath string
}

// NewDownloadJob creates a new download job
func NewDownloadJob(bucket string, key string, filepath string) DownloadJob {
	return DownloadJob{
		Bucket:   bucket,
		Key:      key,
		Filepath: filepath,
	}
}
