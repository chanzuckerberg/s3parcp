package s3utils

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/aws/aws-sdk-go/service/s3"
)

type localPath struct {
	raw    string
	client *s3.S3
}

// IsDir Checks if a localPath is a directory
func (p localPath) IsDir() (bool, error) {
	stat, err := os.Stat(p.raw)
	if err != nil {
		return false, err
	}

	return stat.IsDir(), nil
}

// Exists Checks if a localPath exists as a file or a directory
func (p localPath) Exists() (bool, error) {
	_, err := os.Stat(p.raw)
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

// IsS3 checks if a localPath is an S3Path (it will never be)
func (p localPath) IsS3() bool {
	return false
}

// IsLocal checks if a localPath is a localPath (it will always be)
func (p localPath) IsLocal() bool {
	return true
}

// ListPathsWithPrefix lists all paths with the localPath as a prefix
func (p localPath) ListPathsWithPrefix() ([]Path, error) {
	filepaths := []Path{}
	err := filepath.Walk(
		p.raw,
		func(filepath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				currentPath, err := NewPath(p.client, filepath)
				if err != nil {
					return err
				}
				filepaths = append(filepaths, currentPath)
			}
			return nil
		},
	)
	return filepaths, err
}

// Join joins suffixes to this path
func (p localPath) Join(suffixes ...string) Path {
	joinArgs := append([]string{p.raw}, suffixes...)
	p.raw = path.Join(joinArgs...)
	return p
}

// Base gets the base name of this path
func (p localPath) Base() string {
	return path.Base(p.raw)
}

// WithoutBucket returns a raw string path without the s3 bucket
func (p localPath) WithoutBucket() string {
	return p.raw
}

// Bucket returns an error since localPath has no bucket
func (p localPath) Bucket() (string, error) {
	return "", fmt.Errorf("requested bucket of non-s3 path: %s", p)
}

func (p localPath) String() string {
	return p.raw
}
