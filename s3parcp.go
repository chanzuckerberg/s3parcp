package main

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/chanzuckerberg/s3parcp/mains"
	"github.com/chanzuckerberg/s3parcp/options"
	"github.com/chanzuckerberg/s3parcp/transparents3"
)

func main() {
	before := time.Now()
	opts := options.ParseArgs()

	sess := session.Must(session.NewSessionWithOptions(
		session.Options{
			SharedConfigState: session.SharedConfigEnable,
		},
	))
	client := s3.New(sess)

	sourcePath, err := transparents3.NewPath(client, opts.Positional.Source)
	if err != nil {
		panic(err)
	}

	destPath, err := transparents3.NewPath(client, opts.Positional.Destination)
	if err != nil {
		panic(err)
	}

	if sourcePath.IsS3() && destPath.IsS3() {
		mains.S3ToS3(opts, sourcePath, destPath)
	} else if sourcePath.IsS3() && !destPath.IsS3() {
		mains.S3ToLocal(opts, sourcePath, destPath)
	} else if !sourcePath.IsS3() && destPath.IsS3() {
		mains.LocalToS3(opts, sourcePath, destPath)
	} else {
		mains.LocalToLocal(opts, sourcePath, destPath)
	}
	duration := time.Since(before)
	if opts.Duration {
		fmt.Println(duration.Seconds())
	}
}
