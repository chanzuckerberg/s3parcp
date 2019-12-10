package main

import (
	"s3parcp/mains"
	"s3parcp/options"
	"s3parcp/s3utils"
)

func main() {
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
}
