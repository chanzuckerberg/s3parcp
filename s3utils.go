package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// This capitalization is critical to the implementation please do not change it
//	if you write metadata with different capitalization s3 will fuse it with
//  the existing value of the same name instead of overwriting.
const crc32cChecksumMetadataName = "Crc32c-Checksum"

// GetCRC32CChecksum gets the crc32c checksum from the metadata of an s3 object
func GetCRC32CChecksum(bucket string, key string) (uint32, error) {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	svc := s3.New(sess)

	headObjectResponse, err := svc.HeadObject(&s3.HeadObjectInput{
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

	crc32cChecksumString := *headObjectResponse.Metadata[crc32cChecksumMetadataName]

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
