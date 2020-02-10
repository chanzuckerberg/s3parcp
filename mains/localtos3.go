package mains

import (
	"errors"
	"path"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/chanzuckerberg/s3parcp/options"
	"github.com/chanzuckerberg/s3parcp/s3utils"
	"github.com/chanzuckerberg/s3parcp/transparents3"
)

func getUploadJobs(client *s3.S3, source transparents3.Path, destination transparents3.Path) ([]s3utils.UploadJob, error) {
	isSourceDir, err := source.IsDir()
	isDestDir, err := destination.IsDir()
	destExists, err := destination.Exists()

	// We don't want to create a direcory with the same name as an object but if there is
	//   already a directory with that name it's fine to put stuff in it
	if !isDestDir && isSourceDir && destExists {
		return []s3utils.UploadJob{}, errors.New("Cannot copy directory to existing object")
	}

	filepaths, err := source.ListPathsWithPrefix()
	uploadJobs := make([]s3utils.UploadJob, len(filepaths))

	for i, filepath := range filepaths {
		// TODO create joiner
		destinationKey := destination.ToString()
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
func LocalToS3(opts options.Options, source Path, destination Path) {
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
		source,
		destination,
	)
	if err != nil {
		panic(err)
	}

	err = uploader.UploadAll(uploadJobs)
	if err != nil {
		panic(err)
	}
}
