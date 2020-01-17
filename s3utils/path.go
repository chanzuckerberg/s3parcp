package s3utils

import (
	"net/url"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

// S3PathToBucketAndKey converts an s3 path into it's bucket and key
func S3PathToBucketAndKey(s3path string) (string, string, error) {
	url, err := url.Parse(s3path)
	if err != nil {
		return "", "", err
	}
	return url.Host, url.Path[1:], nil
}

// IsS3Path checks whether a string is an s3 path
func IsS3Path(path string) bool {
	url, err := url.Parse(path)
	if err != nil {
		return false
	}

	return url.Scheme == "s3"
}

// IsS3Directory checks if an s3 path is a directory
func IsS3Directory(client *s3.S3, bucket string, key string) (bool, error) {
	maxKeys := int64(1)
	res, err := client.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket:  aws.String(bucket),
		Prefix:  aws.String(key),
		MaxKeys: &maxKeys,
	})
	if err != nil {
		return false, err
	}

	contents := res.Contents
	if len(contents) == 0 {
		return false, nil
	}

	if *contents[0].Key == key {
		return false, nil
	}

	return true, nil
}

// IsLocalDirectory checks if a local path is to a directory
func IsLocalDirectory(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return fileInfo.IsDir(), nil
}
