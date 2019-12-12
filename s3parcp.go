package main

import (
	"fmt"
	"s3parcp/mains"
	"s3parcp/options"
	"s3parcp/s3utils"
	"time"
)

func main() {
	before := time.Now()
	opts := options.ParseArgs()
	sourceIsS3 := s3utils.IsS3Path(opts.Positional.Source)
	destinationIsS3 := s3utils.IsS3Path(opts.Positional.Destination)

	if sourceIsS3 && destinationIsS3 {
		mains.S3ToS3(opts)
	} else if sourceIsS3 && !destinationIsS3 {
		mains.S3ToLocal(opts)
	} else if !sourceIsS3 && destinationIsS3 {
		mains.LocalToS3(opts)
	} else {
		mains.LocalToLocal(opts)
	}
	duration := time.Since(before)
	if opts.Duration {
		fmt.Println(duration.Seconds())
	}
}
