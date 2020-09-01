package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/chanzuckerberg/s3parcp/filecachedcredentials"
	"github.com/chanzuckerberg/s3parcp/options"
	"github.com/chanzuckerberg/s3parcp/s3utils"
)

// to be set with `-ldflags "-X main.version="`
var version string = "unset"

func main() {
	log.SetPrefix("s3parcp: ")
	log.SetFlags(0)
	opts, err := options.ParseArgs(os.Args[1:])

	// go-flags will handle any logging to the user, just exit on error
	if err != nil {
		os.Exit(2)
	}

	if opts.Version {
		fmt.Println(version)
		return
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
	if !opts.DisableCachedCredentials {
		fileCacheProvider, err := filecachedcredentials.NewFileCacheProvider(sess.Config.Credentials)
		if err != nil {
			log.Fatal("error while setting up cached credentials, try running with --disable-cached-credentials\n")
		}

		sess.Config.Credentials = credentials.NewCredentials(&fileCacheProvider)
	}

	client := s3.New(sess)

	sourcePath, err := s3utils.NewPath(client, string(opts.Positional.Source))
	if err != nil {
		os.Stderr.WriteString(fmt.Sprintf("%s\n", err))
		os.Exit(1)
	}

	destPath, err := s3utils.NewPath(client, string(opts.Positional.Destination))
	if err != nil {
		os.Stderr.WriteString(fmt.Sprintf("%s\n", err))
		os.Exit(1)
	}

	copierOpts := s3utils.CopierOptions{
		BufferSize:  opts.BufferSize,
		Checksum:    opts.Checksum,
		Concurrency: opts.Concurrency,
		DisableSSL:  opts.DisableSSL,
		MaxRetries:  opts.MaxRetries,
		PartSize:    opts.PartSize,
		Verbose:     opts.Verbose,
	}
	copier := s3utils.NewCopier(copierOpts, sess)
	jobs, err := s3utils.GetCopyJobs(sourcePath, destPath, opts.Recursive)
	if err != nil {
		if strings.HasPrefix(err.Error(), "AccessDenied") {
			os.Stderr.WriteString("s3parcp encountered an error from the s3 api: access denied\n")
		} else if strings.HasPrefix(err.Error(), "NoSuchBucket") {
			os.Stderr.WriteString("s3parcp encountered an error from the s3 api: no such bucket\n")
		} else if strings.HasPrefix(err.Error(), "MissingRegion") {
			os.Stderr.WriteString("s3parcp encountered an error from the s3 api: missing region configuration\n")
		} else {
			os.Stderr.WriteString(fmt.Sprintf("%s\n", err))
		}
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
