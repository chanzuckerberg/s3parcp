package s3utils

import (
	"context"
	"errors"
	"path"

	"github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type s3Path struct {
	bucket string
	prefix string
	raw    string
	client *s3.Client
}

// IsDir Checks if a s3Path is a directory
func (p s3Path) IsDir() (bool, error) {
	// Consider the bucket alone as a directory
	if p.prefix == "" {
		return true, nil
	}

	// Paths with a trailing slash must be directories because creating
	//   an object with a trailing slash doesn't work
	if p.raw[len(p.raw)-1] == '/' {
		return true, nil
	}

	// Add trailing / to the prefix to avoid partial matches
	prefix := addTrailingSlash(p.prefix)

	// Only one key is required for the check
	var maxKeys int32 = 1
	request := s3.ListObjectsV2Input{
		Bucket:  &p.bucket,
		Prefix:  &prefix,
		MaxKeys: maxKeys,
	}
	res, err := p.client.ListObjectsV2(context.Background(), &request)
	if err != nil {
		return false, err
	}

	// If no files match the prefix it isn't a directory
	if len(res.Contents) == 0 {
		return false, nil
	}

	return true, nil
}

// Exists Checks if a s3Path exists as an object or a folder
func (p s3Path) Exists() (bool, error) {
	// The bucket alone exists
	if p.prefix == "" {
		return true, nil
	}

	_, err := p.client.HeadObject(context.Background(), &s3.HeadObjectInput{
		Bucket: &p.bucket,
		Key:    &p.prefix,
	})

	var notFound *http.ResponseError
	if err != nil && errors.As(err, &notFound) && notFound.HTTPStatusCode() == 404 {
		return false, nil
	}

	return true, err
}

// IsS3 checks if a s3Path is an s3Path (it always will be)
func (p s3Path) IsS3() bool {
	return true
}

// IsLocal checks if a localPath is a s3Path (it never will be)
func (p s3Path) IsLocal() bool {
	return false
}

// DirOrFolder returns "directory" for localPath and "folder" for s3Path
func (p s3Path) DirOrFolder() string {
	return "folder"
}

// FileOrObject returns "file" for localPath and "object" for s3path
func (p s3Path) FileOrObject() string {
	return "object"
}

// ListPathsWithPrefix lists all paths with the s3Path as a prefix
func (p s3Path) ListPathsWithPrefix() ([]Path, error) {
	res, err := p.client.ListObjectsV2(context.Background(), &s3.ListObjectsV2Input{
		Bucket: &p.bucket,
		Prefix: &p.prefix,
	})

	if err != nil {
		return []Path{}, err
	}

	// Add trailing / to the prefix to avoid partial matches
	prefixDir := addTrailingSlash(p.prefix)

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

// Join joins suffixes to this path
func (p s3Path) Join(suffixes ...string) Path {
	rawJoinArgs := append([]string{p.raw}, suffixes...)
	prefixJoinArgs := append([]string{p.prefix}, suffixes...)
	p.raw = path.Join(rawJoinArgs...)
	p.prefix = path.Join(prefixJoinArgs...)
	return p
}

// WithoutBucket returns a raw string path without the s3 bucket
func (p s3Path) WithoutBucket() string {
	return p.prefix
}

// Base gets the base name of this path
func (p s3Path) Base() string {
	return path.Base(p.raw)
}

// Bucket returns the s3 bucket of this path
func (p s3Path) Bucket() (string, error) {
	return p.bucket, nil
}

func (p s3Path) String() string {
	return p.raw
}
