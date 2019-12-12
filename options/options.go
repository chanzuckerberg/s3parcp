package options

import (
	"os"
	"path"
	"runtime"

	"github.com/jessevdk/go-flags"
)

// Options - the options passed to the executable
type Options struct {
	PartSize    int64 `short:"p" long:"part-size" description:"Part size of parts to be downloaded"`
	Concurrency int   `short:"c" long:"concurrency" description:"Download concurrency"`
	BufferSize  int   `short:"b" long:"buffer-size" description:"Size of download buffer"`
	Checksum    bool  `long:"checksum" description:"Should compare checksum when downloading or place checksum in metadata while uploading"`
	Duration    bool  `short:"d" long:"duration" description:"Prints the duration of the download"`
	MMap        bool  `short:"m" long:"mmap" description:"Use mmap for downloads"`
	Positional  struct {
		Source      string `description:"Source to copy from" required:"yes"`
		Destination string `description:"Destination to copy to (Optional, defaults to source's base name)"`
	} `positional-args:"yes"`
}

// ParseArgs wraps flags.ParseArgs and adds system-dependent defaults
func ParseArgs() Options {
	var opts Options
	_, err := flags.ParseArgs(&opts, os.Args[1:])
	if err != nil {
		os.Exit(2)
	}

	if opts.Positional.Destination == "" {
		opts.Positional.Destination = path.Base(opts.Positional.Source)
	}

	if opts.PartSize == 0 {
		opts.PartSize = int64(os.Getpagesize()) * 1024 * 10
	}

	if opts.Concurrency == 0 {
		opts.Concurrency = runtime.NumCPU()
	}

	return opts
}
