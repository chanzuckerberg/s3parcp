package mains

import (
	"os"

	"github.com/chanzuckerberg/s3parcp/options"
	"github.com/chanzuckerberg/s3parcp/transparents3"
)

// LocalToLocal is the main method for copying local files to local files
func LocalToLocal(opts options.Options, localPath transparents3.Path, destPath transparents3.Path) {
	os.Stderr.WriteString("Copying between local destinations is not supported\n")
	os.Exit(2)
}
