package utils

import (
	"os"
	"path/filepath"
)

// ListFilesRec recursively lists all files in a directory
func ListFilesRec(dirpath string) ([]string, error) {
	filepaths := []string{}
	err := filepath.Walk(
		dirpath,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				filepaths = append(filepaths, path)
			}
			return nil
		},
	)
	return filepaths, err
}
