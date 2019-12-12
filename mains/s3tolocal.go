package mains

import (
	"fmt"
	"os"
	"s3parcp/mmap"
	"s3parcp/options"
	"s3parcp/s3utils"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// S3ToLocal is the main method for copying s3 objects to local files
func S3ToLocal(opts options.Options) {
	sourceBucket, sourceKey, err := s3utils.S3PathToBucketAndKey(opts.Positional.Source)
	if err != nil {
		message := fmt.Sprintf("Error parsing s3 path: %s\n", opts.Positional.Source)
		os.Stderr.WriteString(message)
		os.Exit(1)
	}

	sess := session.Must(
		session.NewSessionWithOptions(
			session.Options{
				SharedConfigState: session.SharedConfigEnable,
			},
		),
	)

	disableSSL := true
	client := s3.New(sess, &aws.Config{
		DisableSSL: &disableSSL,
	})

	downloader := s3manager.NewDownloader(sess, func(d *s3manager.Downloader) {
		d.PartSize = opts.PartSize
		d.Concurrency = opts.Concurrency
		d.S3 = client
		if opts.BufferSize > 0 {
			d.BufferProvider = s3manager.NewPooledBufferedWriterReadFromProvider(opts.BufferSize)
		}
	})

	headObjectOutput, _ := client.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(sourceBucket),
		Key:    aws.String(sourceKey),
	})

	type file interface {
		WriteAt(p []byte, off int64) (n int, err error)
		Close() error
	}
	var f file
	if opts.MMap {
		contentLength := *headObjectOutput.ContentLength
		f, err = mmap.CreateFile(opts.Positional.Destination, contentLength)
		if err != nil {
			panic(err)
		}
	} else {
		// Create a file to write the S3 Object contents to.
		f, err := os.Create(opts.Positional.Destination)
		if err != nil {
			panic(err)
		}
	}

	// Write the contents of S3 Object to the file
	_, err = downloader.Download(f, &s3.GetObjectInput{
		Bucket: aws.String(sourceBucket),
		Key:    aws.String(sourceKey),
	})
	if err != nil {
		panic(err)
	}

	f.Close()

	if opts.Checksum {
		s3utils.CompareChecksum(headObjectOutput, opts.Positional.Destination)
	}
}
