package transparents3

import (
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go/service/s3"
)

type localPath struct {
	raw    string
	client *s3.S3
}

// IsDir Checks if a localPath is a directory
func (path localPath) IsDir() (bool, error) {
	stat, err := os.Stat(path.raw)
	if err != nil {
		return false, err
	}

	return stat.IsDir(), nil
}

// Exists Checks if a localPath is a directory
func (path localPath) Exists() (bool, error) {
	_, err := os.Stat(path.raw)
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

// IsS3 checks if a localPath is an S3Path (it will never be)
func (path localPath) IsS3() bool {
	return false
}

// IsLocal checks if a localPath is a localPath (it will always be)
func (path localPath) IsLocal() bool {
	return true
}

// ListPathsWithPrefix lists all paths with the localPath as a prefix
func (path localPath) ListPathsWithPrefix() ([]Path, error) {
	filepaths := []Path{}
	err := filepath.Walk(
		path.raw,
		func(filepath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				p, err := NewPath(path.client, filepath)
				if err != nil {
					return err
				}
				filepaths = append(filepaths, p)
			}
			return nil
		},
	)
	return filepaths, err
}

// ToString converts a LocalPath to a raw string path
func (path localPath) ToString() string {
	return path.raw
}
