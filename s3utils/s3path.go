package s3utils

import (
	"path"

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
func (p s3Path) IsDir() (bool, error) {
	// Add trailing / to the prefix to avoid partial matches
	prefix := utils.AddTrailingSlash(p.prefix)

	// Only one key is required for the check
	var maxKeys int64 = 1
	request := s3.ListObjectsV2Input{
		Bucket:  aws.String(p.bucket),
		Prefix:  aws.String(prefix),
		MaxKeys: &maxKeys,
	}
	res, err := p.client.ListObjectsV2(&request)
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
func (p s3Path) Exists() (bool, error) {
	_, err := p.client.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(p.prefix),
	})

	if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NotFound" {
		return false, nil
	}

	return true, err
}

// IsS3 checks if a s3Path is an s3Path (it will never be)
func (p s3Path) IsS3() bool {
	return true
}

// IsLocal checks if a localPath is a s3Path (it will always be)
func (p s3Path) IsLocal() bool {
	return false
}

// ListPathsWithPrefix lists all paths with the s3Path as a prefix
func (p s3Path) ListPathsWithPrefix() ([]Path, error) {
	res, err := p.client.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: aws.String(p.bucket),
		Prefix: aws.String(p.prefix),
	})

	if err != nil {
		return []Path{}, err
	}

	// Add trailing / to the prefix to avoid partial matches
	prefixDir := utils.AddTrailingSlash(p.prefix)

	objects := res.Contents
	paths := []Path{}
	for _, object := range objects {
		key := *object.Key
		if key[len(key)-1] != '/' {
			currentPath := s3Path{
				bucket: p.bucket,
				prefix: key,
				raw:    bucketAndKeyToS3Path(p.bucket, key),
				client: p.client,
			}
			if key == p.prefix {
				return []Path{currentPath}, nil
			}
			if key[:len(prefixDir)] == prefixDir {
				paths = append(paths, currentPath)
			}
		}
	}

	return paths, err
}

// ToString converts a s3Path to a raw string path
func (p s3Path) ToString() string {
	return p.raw
}

// Join joins suffixes to this path
func (p s3Path) Join(suffixes ...string) Path {
	rawJoinArgs := append([]string{p.raw}, suffixes...)
	prefixJoinArgs := append([]string{p.prefix}, suffixes...)
	p.raw = path.Join(rawJoinArgs...)
	p.prefix = path.Join(prefixJoinArgs...)
	return p
}

// ToStringWithoutBucket returns a raw string path without the s3 bucket
func (p s3Path) ToStringWithoutBucket() string {
	return p.prefix
}

// WithoutPrefix TODO
func (p s3Path) WithoutPrefix(prefixPath Path) string {
	prefixLength := len(prefixPath.ToStringWithoutBucket())
	return p.prefix[prefixLength:]
}

// Base gets the base name of this path
func (p s3Path) Base() string {
	return path.Base(p.raw)
}

// Bucket TODO
func (p s3Path) Bucket() (string, error) {
	return p.bucket, nil
}
