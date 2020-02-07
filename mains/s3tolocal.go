package mains

import (
	"fmt"
	"os"
	"path"

	"github.com/chanzuckerberg/s3parcp/options"
	"github.com/chanzuckerberg/s3parcp/s3utils"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

func getDownloadJobs(client *s3.S3, source string, destination string) ([]s3utils.DownloadJob, error) {
	sourceBucket, sourcePrefix, err := s3utils.S3PathToBucketAndKey(source)
	if err != nil {
		message := fmt.Sprintf("Encountered error while parsing s3 path: %s\n", source)
		os.Stderr.WriteString(message)
		return []s3utils.DownloadJob{}, err
	}

	isSourceDir, err := s3utils.IsS3Directory(client, sourceBucket, sourcePrefix)
	if err != nil {
		return []s3utils.DownloadJob{}, err
	}

	destStat, err := os.Stat(destination)
	isDestDir := false
	if err != nil {
		if !(os.IsNotExist(err) && !isSourceDir) {
			return []s3utils.DownloadJob{}, err
		}
	} else {
		isDestDir = destStat.IsDir()
	}

	res, err := client.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: aws.String(sourceBucket),
		Prefix: aws.String(sourcePrefix),
	})
	if err != nil {
		return []s3utils.DownloadJob{}, err
	}
	objects := res.Contents
	keys := []string{}
	for _, object := range objects {
		key := *object.Key
		if key != sourcePrefix {
			keys = append(keys, key)
		}
	}
	downloadJobs := make([]s3utils.DownloadJob, len(keys))

	for i, key := range keys {
		destinationFilepath := destination
		if !isSourceDir && isDestDir {
			destinationFilepath = path.Join(destinationFilepath, path.Base(source))
		}
		if isSourceDir && isDestDir {
			destinationFilepath = path.Join(destinationFilepath, key[len(sourcePrefix):])
		}
		fmt.Println(key, destinationFilepath)
		downloadJobs[i] = s3utils.NewDownloadJob(
			sourceBucket,
			key,
			destinationFilepath,
		)
	}

	return downloadJobs, err
}

// S3ToLocal is the main method for copying s3 objects to local files
func S3ToLocal(opts options.Options) {
	downloaderOptions := s3utils.DownloaderOptions{
		BufferSize:  opts.BufferSize,
		Checksum:    opts.Checksum,
		Concurrency: opts.Concurrency,
		Mmap:        opts.Mmap,
		PartSize:    opts.PartSize,
	}
	downloader := s3utils.NewDownloader(downloaderOptions)

	downloadJobs, err := getDownloadJobs(
		downloader.Client,
		opts.Positional.Source,
		opts.Positional.Destination,
	)
	if err != nil {
		panic(err)
	}

	err = downloader.DownloadAll(downloadJobs)
	if err != nil {
		panic(err)
	}
}
