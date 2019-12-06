package main

import (
	"hash/crc32"
	"io/ioutil"
)

// CRC32CChecksum computes the crc32c checksum of a file
func CRC32CChecksum(filename string) (uint32, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return 0, err
	}

	// This ensures that we use the crc32c system command if available
	//   I stepped though the code to verify
	crc32q := crc32.MakeTable(crc32.Castagnoli)
	return crc32.Checksum(data, crc32q), nil
}
