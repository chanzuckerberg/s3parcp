package mains

import (
	"os"

	"github.com/chanzuckerberg/s3parcp/options"
)

// S3ToS3 is the main method for copying s3 objects to s3 objects
func S3ToS3(opts options.Options) {
	os.Stderr.WriteString("Copying between s3 not yet supported")
	os.Exit(2)
}
