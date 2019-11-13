package main

import (
	"net/url"
	"os"
	"runtime"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/jessevdk/go-flags"
)

var opts struct {
	PartSize    int64 `short:"p" long:"part-size" description:"Part size of parts to be downloaded"`
	Concurrency int   `short:"c" long:"concurrency" description:"Download concurrency"`
	BufferSize  int   `short:"b" long:"buffer-size" description:"Size of download buffer"`
	Positional  struct {
		S3Path string `required:"yes"`
		Dest   string `required:"yes"`
	} `positional-args:"yes"`
}

func main() {
	_, err := flags.ParseArgs(&opts, os.Args[1:])
	if err != nil {
		os.Exit(2)
	}

	url, err := url.Parse(opts.Positional.S3Path)
	Bucket := url.Host
	Path := url.Path

	PartSize := opts.PartSize
	if PartSize == 0 {
		PartSize = int64(os.Getpagesize()) * 1024
	}

	Concurrency := opts.Concurrency
	if Concurrency == 0 {
		Concurrency = runtime.NumCPU()
	}

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	disableSSL := true
	client := s3.New(sess, &aws.Config{
		DisableSSL: &disableSSL,
	})

	downloader := s3manager.NewDownloader(sess, func(d *s3manager.Downloader) {
		d.PartSize = PartSize
		d.Concurrency = Concurrency
		d.S3 = client
		if opts.BufferSize > 0 {
			d.BufferProvider = s3manager.NewPooledBufferedWriterReadFromProvider(opts.BufferSize)
		}
	})

	// Create a file to write the S3 Object contents to.
	f, err := os.Create(opts.Positional.Dest)
	if err != nil {
		panic(err)
	}

	// Write the contents of S3 Object to the file
	_, err = downloader.Download(f, &s3.GetObjectInput{
		Bucket: aws.String(Bucket),
		Key:    aws.String(Path),
	})
	if err != nil {
		panic(err)
	}
}
