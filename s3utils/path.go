package s3utils

import "net/url"

// S3PathToBucketAndKey converts an s3 path into it's bucket and key
func S3PathToBucketAndKey(s3path string) (string, string, error) {
	url, err := url.Parse(s3path)
	if err != nil {
		return "", "", err
	}
	return url.Host, url.Path[1:], nil
}

// IsS3Path checks whether a string is an s3 path
func IsS3Path(path string) bool {
	url, err := url.Parse(path)
	if err != nil {
		return false
	}

	return url.Scheme == "s3"
}
