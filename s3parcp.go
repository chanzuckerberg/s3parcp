package main

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"runtime"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/jessevdk/go-flags"
)

func illegalArgsCrash(message string) {
	os.Stderr.WriteString(fmt.Sprintf("%s\n", message))
	os.Exit(2)
}

var opts struct {
	PartSize    int64 `short:"p" long:"part-size" description:"Part size of parts to be downloaded"`
	Concurrency int   `short:"c" long:"concurrency" description:"Download concurrency"`
	BufferSize  int   `short:"b" long:"buffer-size" description:"Size of download buffer"`
	Checksum    bool  `long:"checksum" description:"Should compare checksum when downloading or place checksum in metadata while uploading"`
	Positional  struct {
		Source      string `required:"yes"`
		Destination string `description:"Destination to download to (Optional, defaults to source file name)"`
	} `positional-args:"yes"`
}

func localToLocal() {
	bytes, err := ioutil.ReadFile(opts.Positional.Source)
	if err != nil {
		panic(err)
	}
	ioutil.WriteFile(opts.Positional.Destination, bytes, 0644)
}

func compareChecksum(bucket string, key string, file string) {
	expectedCRC32CChecksum, err := GetCRC32CChecksum(bucket, key)
	if err != nil {
		os.Stderr.WriteString("Encountered error while fetching crc32c checksum\n")
		panic(err)
	}

	if expectedCRC32CChecksum == 0 {
		os.Stderr.WriteString("crc32c checksum not found in s3 object's metadata, try writing one with --checksum-only\n")
		os.Exit(1)
	}

	crc32cChecksum, err := CRC32CChecksum(file)
	if err != nil {
		os.Stderr.WriteString("Encountered error while computing crc32c checksum\n")
		panic(err)
	}

	if crc32cChecksum != expectedCRC32CChecksum {
		os.Stderr.WriteString("Checksums do not match\n")
		os.Exit(1)
	}
}

func download(sourceURL *url.URL) {
	sourceBucket := sourceURL.Host
	sourceKey := sourceURL.Path[1:]

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

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

	// Create a file to write the S3 Object contents to.
	f, err := os.Create(opts.Positional.Destination)
	if err != nil {
		panic(err)
	}

	// Write the contents of S3 Object to the file
	_, err = downloader.Download(f, &s3.GetObjectInput{
		Bucket: aws.String(sourceBucket),
		Key:    aws.String(sourceKey),
	})
	if err != nil {
		panic(err)
	}

	if opts.Checksum {
		compareChecksum(sourceBucket, sourceKey, opts.Positional.Destination)
	}
}

func upload(destinationURL *url.URL) {
	destinationBucket := destinationURL.Host
	destinationKey := destinationURL.Path[1:]

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	disableSSL := true
	client := s3.New(sess, &aws.Config{
		DisableSSL: &disableSSL,
	})

	uploader := s3manager.NewUploader(sess, func(u *s3manager.Uploader) {
		u.PartSize = opts.PartSize
		u.Concurrency = opts.Concurrency
		u.S3 = client
		if opts.BufferSize > 0 {
			u.BufferProvider = s3manager.NewBufferedReadSeekerWriteToPool(opts.BufferSize)
		}
	})

	metadata := make(map[string]*string)
	if opts.Checksum {
		crc32cChecksum, err := CRC32CChecksum(opts.Positional.Source)
		if err != nil {
			os.Stderr.WriteString("Error computing crc32c checksum of source file\n")
			panic(err)
		}
		crc32cChecksumString := fmt.Sprintf("%X", crc32cChecksum)
		metadata[crc32cChecksumMetadataName] = &crc32cChecksumString
	}

	// Open a file to upload
	f, err := os.Open(opts.Positional.Source)
	if err != nil {
		panic(err)
	}

	// Write the contents of S3 Object to the file
	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket:   aws.String(destinationBucket),
		Key:      aws.String(destinationKey),
		Body:     f,
		Metadata: metadata,
	})
	if err != nil {
		panic(err)
	}
}

func main() {
	_, err := flags.ParseArgs(&opts, os.Args[1:])
	if err != nil {
		os.Exit(2)
	}

	if opts.PartSize == 0 {
		opts.PartSize = int64(os.Getpagesize()) * 1024 * 10
	}

	if opts.Concurrency == 0 {
		opts.Concurrency = runtime.NumCPU()
	}

	// This is down here because checksum is only supported locally at the moment and other sources can only be s3
	sourceURL, err := url.Parse(opts.Positional.Source)

	if opts.Positional.Destination == "" {
		opts.Positional.Destination = path.Base(sourceURL.Path)
	}
	destinationURL, err := url.Parse(opts.Positional.Destination)

	if sourceURL.Scheme != "s3" && destinationURL.Scheme == "s3" {
		upload(destinationURL)
	} else if sourceURL.Scheme == "s3" && destinationURL.Scheme == "s3" {
		illegalArgsCrash("S3 to S3 copying not yet supported")
	} else if sourceURL.Scheme != "s3" && destinationURL.Scheme != "s3" {
		localToLocal()
	} else {
		download(sourceURL)
	}
}
