package mains

import (
	"io/ioutil"
	"github.com/chanzuckerberg/s3parcp/options"
)

// LocalToLocal is the main method for copying local files to local files
func LocalToLocal(opts options.Options) {
	bytes, err := ioutil.ReadFile(opts.Positional.Source)
	if err != nil {
		panic(err)
	}
	ioutil.WriteFile(opts.Positional.Destination, bytes, 0644)
}
