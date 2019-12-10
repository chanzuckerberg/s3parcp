package mains

import (
	"fmt"
	"os"
	"s3parcp/checksum"
	"s3parcp/options"
	"s3parcp/s3utils"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// LocalToS3 is the main method for copying local files to s3 objects
func LocalToS3(opts options.Options) {
	destinationBucket, destinationKey, err := s3utils.S3PathToBucketAndKey(opts.Positional.Destination)
	if err != nil {
		message := fmt.Sprintf("Error parsing s3 path: %s\n", opts.Positional.Destination)
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
		crc32cChecksum, err := checksum.CRC32CChecksum(opts.Positional.Source)
		if err != nil {
			os.Stderr.WriteString("Error computing crc32c checksum of source file\n")
			panic(err)
		}
		crc32cChecksumString := fmt.Sprintf("%X", crc32cChecksum)
		metadata[s3utils.Crc32cChecksumMetadataName] = &crc32cChecksumString
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
