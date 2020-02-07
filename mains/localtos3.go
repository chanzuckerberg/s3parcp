package mains

import (
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/chanzuckerberg/s3parcp/options"
	"github.com/chanzuckerberg/s3parcp/s3utils"
	"github.com/chanzuckerberg/s3parcp/utils"
)

func getUploadJobs(client *s3.S3, source string, destination string) ([]s3utils.UploadJob, error) {
	destinationBucket, destinationPrefix, err := s3utils.S3PathToBucketAndKey(destination)
	if err != nil {
		message := fmt.Sprintf("Encountered error while parsing s3 path: %s\n", destination)
		os.Stderr.WriteString(message)
		return []s3utils.UploadJob{}, err
	}

	sourceStat, err := os.Stat(source)
	if err != nil {
		return []s3utils.UploadJob{}, err
	}
	isSourceDir := sourceStat.IsDir()

	isDestDir, err := s3utils.IsS3Directory(client, destinationBucket, destinationPrefix)
	if err != nil {
		return []s3utils.UploadJob{}, err
	}

	destExists, err := s3utils.S3Exists(client, destinationBucket, destinationPrefix)
	// We don't want to create a direcory with the same name as an object but if there is
	//   already a directory with that name it's fine to put stuff in it
	if !isDestDir && isSourceDir && destExists {
		return []s3utils.UploadJob{}, errors.New("Cannot copy directory to existing object")
	}

	filepaths, err := utils.ListFilesRec(source)
	uploadJobs := make([]s3utils.UploadJob, len(filepaths))

	for i, filepath := range filepaths {
		destinationKey := destinationPrefix
		if !isSourceDir && isDestDir {
			destinationKey = path.Join(destinationKey, path.Base(source))
		}
		if isSourceDir && isDestDir {
			destinationKey = path.Join(destinationPrefix, filepath[len(source):])
		}
		uploadJobs[i] = s3utils.NewUploadJob(
			destinationBucket,
			destinationKey,
			filepath,
		)
	}

	return uploadJobs, err
}

// LocalToS3 is the main method for copying local files to s3 objects
func LocalToS3(opts options.Options) {
	uploaderOptions := s3utils.UploaderOptions{
		BufferSize:  opts.BufferSize,
		Checksum:    opts.Checksum,
		Concurrency: opts.Concurrency,
		Mmap:        opts.Mmap,
		PartSize:    opts.PartSize,
	}
	uploader := s3utils.NewUploader(uploaderOptions)

	uploadJobs, err := getUploadJobs(
		uploader.Client,
		opts.Positional.Source,
		opts.Positional.Destination,
	)
	if err != nil {
		panic(err)
	}

	err = uploader.UploadAll(uploadJobs)
	if err != nil {
		panic(err)
	}
}
