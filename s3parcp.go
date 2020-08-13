package main

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/chanzuckerberg/s3parcp/cachedcredentials"
	"github.com/chanzuckerberg/s3parcp/options"
	"github.com/chanzuckerberg/s3parcp/s3utils"
)

// to be set with `-ldflags "-X main.version="`
var version string = "unset"

func main() {
	opts, err := options.ParseArgs()

	// go-flags will handle any logging to the user, just exit on error
	if err != nil {
		os.Exit(2)
	}

	if opts.Version {
		fmt.Println(version)
		os.Exit(0)
	}

	sess := session.Must(session.NewSessionWithOptions(
		session.Options{
			SharedConfigState: session.SharedConfigEnable,
		},
	))

	if opts.S3Url != "" {
		customDomainResolver := func(service, region string, optFns ...func(*endpoints.Options)) (endpoints.ResolvedEndpoint, error) {
			if service == endpoints.S3ServiceID {
				return endpoints.ResolvedEndpoint{
					URL: opts.S3Url,
				}, nil
			}

			return endpoints.DefaultResolver().EndpointFor(service, region, optFns...)
		}

		sess = session.Must(session.NewSessionWithOptions(
			session.Options{
				Config: aws.Config{
					EndpointResolver: endpoints.ResolverFunc(customDomainResolver),
				},
				SharedConfigState: session.SharedConfigEnable,
			},
		))
	}
	sess.Config.Credentials = credentials.NewCredentials(&cachedcredentials.FileCacheProvider{
		Creds: sess.Config.Credentials,
	})

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
		DisableSSL:  opts.DisableSSL,
		MaxRetries:  opts.MaxRetries,
		PartSize:    opts.PartSize,
		Verbose:     opts.Verbose,
	}
	copier := s3utils.NewCopier(copierOpts, sess)
	jobs, err := s3utils.GetCopyJobs(sourcePath, destPath, opts.Recursive)
	if err != nil {
		os.Stderr.WriteString(fmt.Sprintf("%s\n", err))
		os.Exit(1)
	}
	if len(jobs) == 0 && !opts.Recursive {
		message := fmt.Sprintf("no %s found at path %s\n", sourcePath.FileOrObject(), sourcePath)
		os.Stderr.WriteString(message)
		os.Exit(1)
	}

	err = copier.CopyAll(jobs)
	if err != nil {
		os.Stderr.WriteString(fmt.Sprintf("%s\n", err))
		os.Exit(1)
	}
}
