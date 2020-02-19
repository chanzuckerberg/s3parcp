// Package s3checksum contains utilities for encoding/decoding and setting/getting
//   checksums in s3 metadata
package s3checksum

import (
	"errors"
	"fmt"

	"strconv"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// Crc32cChecksumMetadataName is the name of the metadata field to store the crc32c checksum
// 	This capitalization is critical to the implementation please do not change it
//	if you write metadata with different capitalization s3 will fuse it with
//  the existing value of the same name instead of overwriting.
const Crc32cChecksumMetadataName = "Crc32c-Checksum"

// GetCRC32CChecksum gets the crc32c checksum from the metadata of an s3 object
func GetCRC32CChecksum(headObjectOutput *s3.HeadObjectOutput) (uint32, error) {
	if headObjectOutput.Metadata == nil {
		return 0, nil
	}

	crc32cChecksumString := *headObjectOutput.Metadata[Crc32cChecksumMetadataName]

	if crc32cChecksumString == "" {
		return 0, errors.New("object has no crc32c checksum set")
	}

	crc32cChecksum, err := strconv.ParseUint(crc32cChecksumString, 16, 32)
	if err != nil {
		return 0, fmt.Errorf("crc32c checksum: %s is not a valid hexadecimal 32 bit unsigned int", crc32cChecksumString)
	}

	return uint32(crc32cChecksum), nil
}

// SetCRC32CChecksum sets the crc32c checksum on an s3manager.UploadInput
func SetCRC32CChecksum(
	uploadInput s3manager.UploadInput,
	crc32cChecksum uint32,
) s3manager.UploadInput {
	crc32cChecksumString := fmt.Sprintf("%X", crc32cChecksum)
	metadata := uploadInput.Metadata
	if metadata == nil {
		metadata = make(map[string]*string)
	}
	metadata[Crc32cChecksumMetadataName] = &crc32cChecksumString
	uploadInput.Metadata = metadata
	return uploadInput
}
