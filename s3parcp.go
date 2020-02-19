package main

import (
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/chanzuckerberg/s3parcp/options"
	"github.com/chanzuckerberg/s3parcp/s3utils"
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

	sourcePath, err := s3utils.NewPath(client, opts.Positional.Source)
	if err != nil {
		os.Stderr.WriteString(fmt.Sprintf("%s\n", err))
		os.Exit(1)
	}

	destPath, err := s3utils.NewPath(client, opts.Positional.Destination)
	if err != nil {
		os.Stderr.WriteString(fmt.Sprintf("%s\n", err))
		os.Exit(1)
	}

	copierOpts := s3utils.CopierOptions{
		BufferSize:  opts.BufferSize,
		Checksum:    opts.Checksum,
		Concurrency: opts.Concurrency,
		Mmap:        opts.Mmap,
		PartSize:    opts.PartSize,
	}
	copier := s3utils.NewCopier(copierOpts)
	jobs, err := s3utils.GetCopyJobs(sourcePath, destPath)
	if err != nil {
		os.Stderr.WriteString(fmt.Sprintf("%s\n", err))
		os.Exit(1)
	}
	if len(jobs) == 0 {
		filesOrObjects := "files"
		if sourcePath.IsS3() {
			filesOrObjects = "objects"
		}
		message := fmt.Sprintf("no %s found at path %s\n", filesOrObjects, sourcePath)
		os.Stderr.WriteString(message)
		os.Exit(1)
	}

	err = copier.CopyAll(jobs)
	if err != nil {
		os.Stderr.WriteString(fmt.Sprintf("%s\n", err))
		os.Exit(1)
	}

	duration := time.Since(before)
	if opts.Duration {
		fmt.Println(duration.Seconds())
	}
}
