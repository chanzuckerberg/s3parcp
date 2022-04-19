package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
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

	configFuncs := make([]func(*config.LoadOptions) error, 0)

	if opts.MaxRetries != 0 {
		configFuncs = append(configFuncs, config.WithRetryMaxAttempts(opts.MaxRetries))
	}

	if opts.Verbose {
		configFuncs = append(configFuncs, config.WithClientLogMode(aws.LogRetries|aws.LogRequest))
	}

	if opts.S3Url != "" {
		customDomainResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			if service == s3.ServiceID {
				return aws.Endpoint{
					URL: opts.S3Url,
				}, nil
			}

			return aws.Endpoint{}, &aws.EndpointNotFoundError{}
		})

		configFuncs = append(configFuncs, config.WithEndpointResolverWithOptions(customDomainResolver))
		if err != nil {
			log.Fatal(err)
		}
	}

	cfg, err := config.LoadDefaultConfig(context.Background(), configFuncs...)
	if err != nil {
		log.Fatal(err)
	}

	if opts.FileCachedCredentials {
		fileCacheProvider, err := filecachedcredentials.NewFileCacheProvider(cfg.Credentials)
		if err != nil {
			log.Fatal("error setting up cached credentials\n")
		}

		cfg.Credentials = &fileCacheProvider
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if opts.S3Url != "" {
			o.UsePathStyle = true
		}
	})

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
	copier := s3utils.NewCopier(copierOpts, client)
	jobs, err := s3utils.GetCopyJobs(sourcePath, destPath, opts.Recursive)
	if err != nil {
		if strings.HasPrefix(err.Error(), "AccessDenied") {
			log.Println("error received from the s3 api - access denied")
		} else if strings.HasPrefix(err.Error(), "NoSuchBucket") {
			log.Println("no such bucket")
		} else if strings.HasPrefix(err.Error(), "MissingRegion") {
			log.Println("missing region configuration")
		} else {
			log.Printf("%s\n", err)
		}
		os.Exit(1)
	}
	if len(jobs) == 0 && !opts.Recursive {
		log.Fatalf("no %s found at path %s\n", sourcePath.FileOrObject(), sourcePath)
	}

	err = copier.CopyAll(jobs)
	if err != nil {
		log.Fatalf("%s\n", err)
	}
}
