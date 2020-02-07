package mains

import (
	"os"

	"github.com/chanzuckerberg/s3parcp/options"
)

// LocalToLocal is the main method for copying local files to local files
func LocalToLocal(opts options.Options) {
	os.Stderr.WriteString("Copying between local destinations is not supported\n")
	os.Exit(2)
}
