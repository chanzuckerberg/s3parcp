package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/chanzuckerberg/s3parcp/options"
	"github.com/chanzuckerberg/s3parcp/s3utils"
)

// Update this with new versions
const version = "0.1.4-alpha"

// FileCacheProvider bleep
type FileCacheProvider struct {
	Creds *credentials.Credentials
}

// CachedCredentials caches credentials
type CachedCredentials struct {
	AccessKeyID     string
	ExpiresAt       time.Time
	ProviderName    string
	SecretAccessKey string
	SessionToken    string
}

func fileExists(name string) (bool, error) {
	_, err := os.Stat(name)
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

// Retrieve retrieves
func (f *FileCacheProvider) Retrieve() (credentials.Value, error) {
	cacheDir := path.Join(os.Getenv("HOME"), ".s3parcp")
	err := os.MkdirAll(cacheDir, os.ModePerm)
	if err != nil {
		return credentials.Value{}, err
	}

	cacheFile := path.Join(cacheDir, "credentials-cache.json")

	useCache, err := fileExists(cacheFile)
	if err != nil {
		return credentials.Value{}, err
	}

	cachedCredentials := CachedCredentials{}

	if useCache {
		bytes, err := ioutil.ReadFile(cacheFile)
		useCache = err == nil
		if err != nil {
			err = json.Unmarshal(bytes, &cachedCredentials)
			useCache = err == nil
		}
	}

	if useCache {
		useCache = cachedCredentials.ExpiresAt.After(time.Now())
	}

	if !useCache {
		credentials, err := f.Creds.Get()
		if err != nil {
			fmt.Println(err)
			panic(err)
		}
		expiresAt, _ := f.Creds.ExpiresAt()
		cachedCredentials = CachedCredentials{
			AccessKeyID:     credentials.AccessKeyID,
			ExpiresAt:       expiresAt,
			ProviderName:    credentials.ProviderName,
			SecretAccessKey: credentials.SecretAccessKey,
			SessionToken:    credentials.SessionToken,
		}
		data, _ := json.Marshal(cachedCredentials)
		fd, err := syscall.Open(cacheFile, syscall.O_CREAT|syscall.O_RDWR, 0600)
		if err != nil {
			panic(err)
		}
		err = syscall.Flock(fd, syscall.LOCK_EX)
		if err != nil {
			panic(err)
		}
		_, err = syscall.Write(fd, data)
		if err != nil {
			panic(err)
		}
		syscall.Flock(fd, syscall.LOCK_UN)
		syscall.Close(fd)
	}

	return credentials.Value{
		AccessKeyID:     cachedCredentials.AccessKeyID,
		ProviderName:    cachedCredentials.ProviderName,
		SecretAccessKey: cachedCredentials.SecretAccessKey,
		SessionToken:    cachedCredentials.SessionToken,
	}, nil
}

// IsExpired checks is expired
func (f *FileCacheProvider) IsExpired() bool {
	return f.Creds.IsExpired()
}

func main() {
	before := time.Now()
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
	sess.Config.Credentials = credentials.NewCredentials(&FileCacheProvider{
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
		PartSize:    opts.PartSize,
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

	duration := time.Since(before)
	if opts.Duration {
		fmt.Println(duration.Seconds())
	}
}
