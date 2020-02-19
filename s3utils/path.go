package s3utils

import (
	"fmt"
	"net/url"
	"path"

	"github.com/aws/aws-sdk-go/service/s3"
)

// Path is an interface of functions to apply transparently to s3 or local paths
type Path interface {
	IsDir() (bool, error)
	Exists() (bool, error)
	IsS3() bool
	IsLocal() bool
	ListPathsWithPrefix() ([]Path, error)
	Join(...string) Path
	Base() string
	WithoutBucket() string
	Bucket() (string, error)
	String() string
}

// s3PathToBucketAndKey converts an s3 path into its bucket and key
func s3PathToBucketAndKey(s3path string) (string, string, error) {
	url, err := url.Parse(s3path)
	if err != nil {
		return "", "", err
	}
	key := ""
	if url.Path != "/" && url.Path != "" {
		key = url.Path[1:]
	}
	return url.Host, key, nil
}

// bucketAndKeyToS3Path converts a bucket and key to an s3 path
func bucketAndKeyToS3Path(bucket string, key string) string {
	return fmt.Sprintf("s3://%s", path.Join(bucket, key))
}

// isS3Path checks whether a string is an s3 path
func isS3Path(path string) bool {
	url, err := url.Parse(path)
	if err != nil {
		return false
	}

	return url.Scheme == "s3"
}

// addTrailingSlash adds a / to the end of a string if there isn't one there
func addTrailingSlash(path string) string {
	if path[len(path)-1] != '/' {
		return path + "/"
	}
	return path
}

// NewPath creates a Path from a raw string
func NewPath(client *s3.S3, raw string) (Path, error) {
	if isS3Path(raw) {
		bucket, key, err := s3PathToBucketAndKey(raw)
		if err != nil {
			return nil, fmt.Errorf("parsing s3 path %s: %v", raw, err)
		}
		return s3Path{
			bucket: bucket,
			prefix: key,
			raw:    raw,
			client: client,
		}, nil
	}
	return localPath{
		raw:    raw,
		client: client,
	}, nil
}
