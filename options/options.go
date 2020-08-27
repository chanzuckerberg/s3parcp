package options

import (
	"errors"
	"fmt"
	"os"
	"path"
	"runtime"

	"github.com/jessevdk/go-flags"
)

// Options - the options passed to the executable
type Options struct {
	PartSize                 int64  `short:"p" long:"part-size" description:"Part size in bytes of parts to be downloaded"`
	Concurrency              int    `short:"c" long:"concurrency" description:"Download concurrency"`
	BufferSize               int    `short:"b" long:"buffer-size" description:"Size of download buffer in bytes"`
	Checksum                 bool   `long:"checksum" description:"Compare checksum if downloading or place checksum in metadata if uploading"`
	Mmap                     bool   `short:"m" long:"mmap" description:"Use mmap for downloads"`
	Recursive                bool   `short:"r" long:"recursive" description:"Copy directories or folders recursively"`
	Version                  bool   `long:"version" description:"Print the current version"`
	S3Url                    string `long:"s3_url" description:"A custom s3 API url (also available as an environment variable 'S3PARCP_S3_URL', the flag takes precedence)"`
	MaxRetries               int    `long:"max-retries" description:"Max per chunk retries" default:"3"`
	DisableSSL               bool   `long:"disable-ssl" description:"Disable SSL"`
	DisableCachedCredentials bool   `long:"disable-cached-credentials" description:"Disable caching AWS credentials"`
	Verbose                  bool   `short:"v" long:"verbose" description:"verbose logging"`
	Positional               struct {
		Source      flags.Filename `description:"Source to copy from"`
		Destination flags.Filename `description:"Destination to copy to (Optional, defaults to source's base name)"`
	} `positional-args:"yes"`
}

// ParseArgs wraps flags.ParseArgs and adds system-dependent defaults
func ParseArgs(args []string) (Options, error) {
	var opts Options
	_, err := flags.ParseArgs(&opts, args)
	if err != nil {
		return opts, err
	}

	if !opts.Version && opts.Positional.Source == "" {
		message := "the required argument `Source` was not provided"
		os.Stderr.WriteString(fmt.Sprintf("%s\n", message))
		return opts, errors.New(message)
	}

	if opts.Positional.Destination == "" {
		opts.Positional.Destination = flags.Filename(path.Base(string(opts.Positional.Source)))
	}

	if opts.PartSize == 0 {
		opts.PartSize = int64(os.Getpagesize()) * 1024 * 10
	}

	if opts.Concurrency == 0 {
		opts.Concurrency = runtime.NumCPU()
	}

	if opts.S3Url == "" {
		envS3Url := os.Getenv("S3PARCP_S3_URL")
		if envS3Url != "" {
			opts.S3Url = envS3Url
		}
	}

	return opts, nil
}
