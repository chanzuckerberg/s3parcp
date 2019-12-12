package checksum

import (
	"hash/crc32"
)

// CRC32CChecksum computes the crc32c checksum of a file
func CRC32CChecksum(data []byte) (uint32, error) {
	// This ensures that we use the crc32c system command if available
	//   I stepped though the code to verify
	crc32q := crc32.MakeTable(crc32.Castagnoli)
	return crc32.Checksum(data, crc32q), nil
}
