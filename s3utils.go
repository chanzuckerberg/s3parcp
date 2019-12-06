package main

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// This capitalization is critical to the implementation please do not change it
//	if you write metadata with different capitalization s3 will fuse it with
//  the existing value of the same name instead of overwriting.
const crc32cChecksumMetadataName = "Crc32c-Checksum"

// AddS3Metadata adds metadata to existing s3 object
//  SSE not currently supported
func AddS3Metadata(Bucket string, Key string, metadata map[string]*string) error {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	svc := s3.New(sess)

	headObjectResponse, err := svc.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(Bucket),
		Key:    aws.String(Key),
	})
	if err != nil {
		return err
	}

	objectACLResponse, err := svc.GetObjectAcl(&s3.GetObjectAclInput{
		Bucket: aws.String(Bucket),
		Key:    aws.String(Key),
	})
	if err != nil {
		return err
	}

	bucketRequestPaymentResponse, err := svc.GetBucketRequestPayment(&s3.GetBucketRequestPaymentInput{
		Bucket: aws.String(Bucket),
	})
	if err != nil {
		return err
	}
	payer := *bucketRequestPaymentResponse.Payer

	metadataDirective := "REPLACE"
	taggingDirective := "COPY"
	copyObjectInput := s3.CopyObjectInput{
		Bucket:            aws.String(Bucket),
		Key:               aws.String(Key),
		CopySource:        aws.String(path.Join(Bucket, Key)),
		MetadataDirective: &metadataDirective,
		TaggingDirective:  &taggingDirective,
	}

	mergedMetadata := make(map[string]*string)

	if headObjectResponse.Metadata != nil {
		for key, value := range headObjectResponse.Metadata {
			mergedMetadata[key] = value
		}
	}

	for key, value := range metadata {
		mergedMetadata[key] = value
	}

	copyObjectInput.SetMetadata(mergedMetadata)

	if headObjectResponse.CacheControl != nil {
		copyObjectInput.SetCacheControl(*headObjectResponse.CacheControl)
	}

	if headObjectResponse.ContentDisposition != nil {
		copyObjectInput.SetContentDisposition(*headObjectResponse.ContentDisposition)
	}

	if headObjectResponse.ContentEncoding != nil {
		copyObjectInput.SetContentEncoding(*headObjectResponse.ContentEncoding)
	}

	if headObjectResponse.ContentLanguage != nil {
		copyObjectInput.SetContentLanguage(*headObjectResponse.ContentLanguage)
	}

	if headObjectResponse.ContentType != nil {
		copyObjectInput.SetContentType(*headObjectResponse.ContentType)
	}

	if headObjectResponse.Expires != nil {
		dateString := *headObjectResponse.Expires
		timestamp, err := time.Parse(time.RFC1123, dateString)
		if err != nil {
			fmt.Printf("Error parsing date: %s", dateString)
			panic(err)
		}
		copyObjectInput.SetExpires(timestamp)
	}

	if headObjectResponse.ObjectLockLegalHoldStatus != nil {
		copyObjectInput.SetObjectLockLegalHoldStatus(*headObjectResponse.ObjectLockLegalHoldStatus)
	}

	if headObjectResponse.ObjectLockMode != nil {
		copyObjectInput.SetObjectLockMode(*headObjectResponse.ObjectLockMode)
	}

	if headObjectResponse.ObjectLockRetainUntilDate != nil {
		copyObjectInput.SetObjectLockRetainUntilDate(*headObjectResponse.ObjectLockRetainUntilDate)
	}

	if payer == "requester" {
		copyObjectInput.SetRequestPayer("requester")
	}

	if headObjectResponse.StorageClass != nil {
		copyObjectInput.SetStorageClass(*headObjectResponse.StorageClass)
	}

	if headObjectResponse.WebsiteRedirectLocation != nil {
		copyObjectInput.SetWebsiteRedirectLocation(*headObjectResponse.WebsiteRedirectLocation)
	}

	_, err = svc.CopyObject(&copyObjectInput)
	if err != nil {
		panic(err)
	}

	_, err = svc.PutObjectAcl(&s3.PutObjectAclInput{
		Bucket: aws.String(Bucket),
		Key:    aws.String(Key),
		AccessControlPolicy: &s3.AccessControlPolicy{
			Grants: objectACLResponse.Grants,
			Owner:  objectACLResponse.Owner,
		},
	})
	if err != nil {
		fmt.Println("Warning ACL was not transferred")
		panic(err)
	}

	return err
}

// WriteCRC32CChecksumMetadata writes crc32c checksum to s3 object metadata
func WriteCRC32CChecksumMetadata(bucket string, key string, crc32cChecksum uint32) error {
	crc32cChecksumString := fmt.Sprintf("%X", crc32cChecksum)
	metadata := map[string]*string{crc32cChecksumMetadataName: &crc32cChecksumString}
	return AddS3Metadata(bucket, key, metadata)
}

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
