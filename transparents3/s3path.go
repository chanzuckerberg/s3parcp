package transparents3

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/chanzuckerberg/s3parcp/utils"
)

type s3Path struct {
	bucket string
	prefix string
	raw    string
	client *s3.S3
}

// IsDir Checks if a s3Path is a directory
func (path s3Path) IsDir() (bool, error) {
	// Add trailing / to the prefix to avoid partial matches
	prefix := utils.AddTrailingSlash(path.prefix)

	// Only one key is required for the check
	var maxKeys int64 = 1
	request := s3.ListObjectsV2Input{
		Bucket:  aws.String(path.bucket),
		Prefix:  aws.String(prefix),
		MaxKeys: &maxKeys,
	}
	res, err := path.client.ListObjectsV2(&request)
	if err != nil {
		return false, err
	}

	// If no files match the prefix it isn't a directory
	if len(res.Contents) == 0 {
		return false, nil
	}

	return true, nil
}

// Exists Checks if a s3Path is a directory
func (path s3Path) Exists() (bool, error) {
	_, err := path.client.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(path.bucket),
		Key:    aws.String(path.prefix),
	})

	if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NotFound" {
		return false, nil
	}

	return true, err
}

// IsS3 checks if a s3Path is an s3Path (it will never be)
func (path s3Path) IsS3() bool {
	return false
}

// IsLocal checks if a localPath is a s3Path (it will always be)
func (path s3Path) IsLocal() bool {
	return true
}

// ListPathsWithPrefix lists all paths with the s3Path as a prefix
func (path s3Path) ListPathsWithPrefix() ([]Path, error) {
	// Add trailing / to the prefix to avoid partial matches
	prefix := utils.AddTrailingSlash(path.prefix)

	res, err := path.client.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: aws.String(path.bucket),
		Prefix: aws.String(prefix),
	})
	if err != nil {
		return []Path{}, err
	}

	objects := res.Contents
	paths := make([]Path, len(objects))
	for i, object := range objects {
		var p Path
		p, err = NewPath(path.client, *object.Key)
		paths[i] = p
	}

	return paths, err
}

// ToString converts a s3Path to a raw string path
func (path s3Path) ToString() string {
	return path.raw
}
