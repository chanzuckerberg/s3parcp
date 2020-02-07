package s3utils

import (
	"net/url"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
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

// S3Exists checks if an s3 path is an object
func S3Exists(client *s3.S3, bucket string, prefix string) (bool, error) {
	_, err := client.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(prefix),
	})

	if aerr, ok := err.(awserr.Error); ok && aerr.Code() == "NotFound" {
		return false, nil
	}

	return true, err
}

// IsS3Directory checks if an s3 path is a directory
func IsS3Directory(client *s3.S3, bucket string, prefix string) (bool, error) {
	// Add trailing / to the prefix to avoid partial matches
	if prefix[len(prefix)-1] != '/' {
		prefix = prefix + "/"
	}

	// Only one key is required for the check
	var maxKeys int64 = 1
	request := s3.ListObjectsV2Input{
		Bucket:  aws.String(bucket),
		Prefix:  aws.String(prefix),
		MaxKeys: &maxKeys,
	}
	res, err := client.ListObjectsV2(&request)
	if err != nil {
		return false, err
	}

	// If no files match the prefix it isn't a directory
	if len(res.Contents) == 0 {
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
