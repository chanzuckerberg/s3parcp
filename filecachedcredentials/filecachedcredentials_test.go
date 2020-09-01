package filecachedcredentials

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws/credentials"
)

type credentialsMock struct {
	credentialsValue credentials.Value
	expiresAt        time.Time

	expiresAtErr error
	getCallsErr  error

	expiresAtCalls int
	getCalls       int
	isExpiredCalls int
}

func (c *credentialsMock) ExpiresAt() (time.Time, error) {
	c.expiresAtCalls++
	return c.expiresAt, c.expiresAtErr
}

func (c *credentialsMock) Get() (credentials.Value, error) {
	c.getCalls++
	return c.credentialsValue, c.getCallsErr
}

func (c *credentialsMock) IsExpired() bool {
	fmt.Println(c.isExpiredCalls)
	c.isExpiredCalls++
	fmt.Println(c.expiresAt)
	return c.expiresAt.Before(time.Now())
}

func TestIsExpiredExpired(t *testing.T) {
	c := credentialsMock{
		expiresAt: time.Now(),
	}

	fileCacheProvider := FileCacheProvider{
		credentials: &c,
	}
	isExpired := fileCacheProvider.IsExpired()

	if isExpired != true {
		t.Error("expected fileCacheProvider.IsExpired to return true if the credential expiration timestamp is before now but it returned false")
	}

	if c.isExpiredCalls != 1 {
		t.Errorf("expected fileCacheProvider.ExpiresAt to call Creds.ExpiresAt once but it was called %d times", c.expiresAtCalls)
	}
}

func TestIsExpiredFresh(t *testing.T) {
	c := credentialsMock{
		expiresAt: time.Now().Add(1 * time.Minute),
	}

	fileCacheProvider := FileCacheProvider{
		credentials: &c,
	}
	isExpired := fileCacheProvider.IsExpired()

	if isExpired != false {
		t.Error("expected fileCacheProvider.IsExpired to return false if the credential expiration timestamp is after now but it returned true")
	}

	if c.isExpiredCalls != 1 {
		t.Errorf("expected fileCacheProvider.ExpiresAt to call Creds.ExpiresAt once but it was called %d times", c.expiresAtCalls)
	}
}

func TestNewFileCachedCredentials(t *testing.T) {
	creds := credentialsMock{}
	fileCacheProvider, fileCacheProviderErr := NewFileCacheProvider(&creds)
	cacheHome, cacheHomeErr := os.UserCacheDir()

	if fileCacheProviderErr != nil && cacheHomeErr != nil {
		return
	}

	if fileCacheProvider.cacheHome != cacheHome {
		t.Errorf("expected fileCacheProvider.cacheHome to equal os.UserCacheDir but it was %s and os.UserCacheDir was %s", fileCacheProvider.cacheHome, cacheHome)
	}

	if fileCacheProviderErr != nil {
		t.Errorf("NewFileCacheProvider should not error if os.UserCacheDir does not error but it errored with %s", fileCacheProviderErr)
	}

	if cacheHomeErr != nil {
		t.Errorf("NewFileCacheProvider should error if os.UserCacheDir does but it did not error and os.UserCacheDir errored with with %s", cacheHomeErr)
	}
}

func TestRetreiveEmptyCache(t *testing.T) {
	creds := credentialsMock{
		credentialsValue: credentials.Value{
			AccessKeyID:     "dummy",
			SecretAccessKey: "dummy",
			SessionToken:    "dummy",
			ProviderName:    "dummy",
		},
		expiresAt: time.Now().Add(1 * time.Minute),
	}

	cacheHome, err := ioutil.TempDir("/tmp", "file-cache-provider-test-")
	defer os.RemoveAll(cacheHome)
	if err != nil {
		t.Errorf("encountered error while making temporary directory: %s", err)
		t.FailNow()
	}

	fileCacheProvider := FileCacheProvider{
		credentials: &creds,
		cacheHome:   cacheHome,
	}

	credResults, _ := fileCacheProvider.Retrieve()

	if credResults.AccessKeyID != "dummy" {
		t.Error("asdasd")
	}

	cacheHomeItems, _ := ioutil.ReadDir(cacheHome)
	cacheDir := cacheHomeItems[0]

	if cacheDir.Name() != "s3parcp" || cacheDir.IsDir() == false {
		t.Errorf("%s", cacheDir.Name())
	}

	cacheDirItems, _ := ioutil.ReadDir(path.Join(cacheHome, cacheDir.Name()))
	cacheFile := cacheDirItems[0]

	if cacheFile.Name() != "credentials-cache.json" {
		t.Error(cacheFile.Name())
	}

	fileCacheProvider.Retrieve()

	ioutil.WriteFile(path.Join(cacheHome, cacheDir.Name(), cacheFile.Name()), []byte("junk"), os.ModePerm)
	fileCacheProvider.Retrieve()
}

func TestRetreiveEmptyCacheExpiresAtError(t *testing.T) {
	creds := credentialsMock{
		credentialsValue: credentials.Value{
			AccessKeyID:     "dummy",
			SecretAccessKey: "dummy",
			SessionToken:    "dummy",
			ProviderName:    "dummy",
		},
		expiresAt:    time.Now(),
		expiresAtErr: errors.New("dummy"),
	}

	cacheHome, err := ioutil.TempDir("/tmp", "file-cache-provider-test-")
	defer os.RemoveAll(cacheHome)
	if err != nil {
		t.Errorf("encountered error while making temporary directory: %s", err)
		t.FailNow()
	}

	fileCacheProvider := FileCacheProvider{
		credentials: &creds,
		cacheHome:   cacheHome,
	}

	credResults, _ := fileCacheProvider.Retrieve()

	if credResults.AccessKeyID != "dummy" {
		t.Error("asdasd")
	}

	cacheHomeItems, _ := ioutil.ReadDir(cacheHome)
	cacheDir := cacheHomeItems[0]

	if cacheDir.Name() != "s3parcp" || cacheDir.IsDir() == false {
		t.Errorf("%s", cacheDir.Name())
	}

	cacheDirItems, _ := ioutil.ReadDir(path.Join(cacheHome, cacheDir.Name()))

	if len(cacheDirItems) > 0 {
		t.Error("asdasdasd")
	}
}
