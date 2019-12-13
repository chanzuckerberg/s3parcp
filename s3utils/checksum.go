package s3utils

import (
	"fmt"
	"os"
	"github.com/chanzuckerberg/s3parcp/checksum"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

// Crc32cChecksumMetadataName is the name of the metadata field to store the crc32c checksum
// 	This capitalization is critical to the implementation please do not change it
//	if you write metadata with different capitalization s3 will fuse it with
//  the existing value of the same name instead of overwriting.
const Crc32cChecksumMetadataName = "Crc32c-Checksum"

// GetCRC32CChecksum gets the crc32c checksum from the metadata of an s3 object
func GetCRC32CChecksum(client *s3.S3, bucket string, key string) (uint32, error) {
	headObjectResponse, err := client.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		os.Stderr.WriteString("Error fetching head object for fetching crc32c checksum\n")
		return 0, err
	}

	if headObjectResponse.Metadata == nil {
		return 0, nil
	}

	crc32cChecksumString := *headObjectResponse.Metadata[Crc32cChecksumMetadataName]

	if crc32cChecksumString == "" {
		return 0, nil
	}

	crc32cChecksum, err := strconv.ParseUint(crc32cChecksumString, 16, 32)
	if err != nil {
		os.Stderr.WriteString(fmt.Sprintf("crc32c checksum: %s is not a valid hexidecimal 32 bit unsigned int\n", crc32cChecksumString))
		return 0, err
	}

	return uint32(crc32cChecksum), nil
}

// CompareChecksum compares an s3 object's checksum from metadata with a file's checksum
func CompareChecksum(client *s3.S3, bucket string, key string, file string) {
	expectedCRC32CChecksum, err := GetCRC32CChecksum(client, bucket, key)
	if err != nil {
		os.Stderr.WriteString("Encountered error while fetching crc32c checksum\n")
		panic(err)
	}

	if expectedCRC32CChecksum == 0 {
		os.Stderr.WriteString("crc32c checksum not found in s3 object's metadata, try writing one with --checksum-only\n")
		os.Exit(1)
	}

	crc32cChecksum, err := checksum.CRC32CChecksum(file)
	if err != nil {
		os.Stderr.WriteString("Encountered error while computing crc32c checksum\n")
		panic(err)
	}

	if crc32cChecksum != expectedCRC32CChecksum {
		os.Stderr.WriteString("Checksums do not match\n")
		os.Exit(1)
	}
}
