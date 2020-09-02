package filecachedcredentials

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws/credentials"
)

type credentialsMock struct {
	credentialsValue credentials.Value
	expiresAt        time.Time

	expiresAtErr error
	getCallsErr  error

	expiresAtCalls int32
	getCalls       int32
	isExpiredCalls int32
}

func (c *credentialsMock) ExpiresAt() (time.Time, error) {
	atomic.AddInt32(&(c.expiresAtCalls), 1)
	return c.expiresAt, c.expiresAtErr
}

func (c *credentialsMock) Get() (credentials.Value, error) {
	atomic.AddInt32(&(c.getCalls), 1)
	return c.credentialsValue, c.getCallsErr
}

func (c *credentialsMock) IsExpired() bool {
	atomic.AddInt32(&(c.isExpiredCalls), 1)
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

func TestRetreiveCachingFresh(t *testing.T) {
	creds := credentialsMock{
		credentialsValue: credentials.Value{
			AccessKeyID:     "DummyAccessKeyID",
			SecretAccessKey: "DummySecretAccessKey",
			SessionToken:    "DummySessionToken",
			ProviderName:    "DummyProviderName",
		},
		expiresAt: time.Now().Add(1 * time.Minute),
	}

	cacheHome, err := ioutil.TempDir("/tmp", "file-cache-provider-test-")
	if err != nil {
		t.Fatalf("error making temporary directory - %s", err)
	}
	defer (func() {
		if os.RemoveAll(cacheHome) != nil {
			t.Logf("failed to remove %s\n", cacheHome)
		}
	})()

	fileCacheProvider := FileCacheProvider{
		credentials: &creds,
		cacheHome:   cacheHome,
	}

	credResults, err := fileCacheProvider.Retrieve()
	if err != nil {
		t.Fatalf("fileCacheProvider.Retrieve returned non nil error - %s\n", err)
	}

	if credResults.AccessKeyID != creds.credentialsValue.AccessKeyID {
		t.Errorf("fileCacheProvider.Retrieve should return credentials from the credentials API - expected AccessKeyID to be %s but it was %s",
			credResults.AccessKeyID, creds.credentialsValue.AccessKeyID)
	}

	if credResults.SecretAccessKey != creds.credentialsValue.SecretAccessKey {
		t.Errorf("fileCacheProvider.Retrieve should return credentials from the credentials API - expected SecretAccessKey to be %s but it was %s",
			credResults.SecretAccessKey, creds.credentialsValue.SecretAccessKey)
	}

	if credResults.SessionToken != creds.credentialsValue.SessionToken {
		t.Errorf("fileCacheProvider.Retrieve should return credentials from the credentials API - expected SessionToken to be %s but it was %s",
			credResults.SessionToken, creds.credentialsValue.SessionToken)
	}

	if credResults.ProviderName != creds.credentialsValue.ProviderName {
		t.Errorf("fileCacheProvider.Retrieve should return credentials from the credentials API - expected ProviderName to be %s but it was %s",
			credResults.ProviderName, creds.credentialsValue.ProviderName)
	}

	if creds.getCalls != 1 {
		t.Errorf("fileCacheProvider.Retrieve should call Get on the credentials API exactly once with no cache but it was called %d times", creds.getCalls)
	}

	cacheHomeItems, err := ioutil.ReadDir(cacheHome)
	if err != nil {
		t.Fatalf("error reading cache home directory %s", cacheHome)
	}
	if len(cacheHomeItems) != 1 {
		t.Fatalf("expected one item in cache home directory but found %d", len(cacheHomeItems))
	}
	cacheDir := cacheHomeItems[0]

	if cacheDir.Name() != "s3parcp" || cacheDir.IsDir() == false {
		t.Error("calling fileCacheProvider.Retrieve should create a cache directory but it did not")
	}

	cacheDirItems, err := ioutil.ReadDir(path.Join(cacheHome, cacheDir.Name()))
	if err != nil {
		t.Fatalf("error reading cache directory %s", path.Join(cacheHome, cacheDir.Name()))
	}
	if len(cacheDirItems) != 1 {
		t.Fatalf("expected one item in cache directory but found %d", len(cacheDirItems))
	}
	cacheFile := cacheDirItems[0]

	if cacheFile.Name() != "credentials-cache.json" {
		t.Errorf("expected cache file credentials-cache.json but found %s", cacheFile.Name())
	}

	err = os.RemoveAll(cacheHome)
	if err != nil {
		t.Logf("failed to remove %s\n", cacheHome)
	}
}

func TestRetreiveCachingCached(t *testing.T) {
	creds := credentialsMock{
		credentialsValue: credentials.Value{
			AccessKeyID:     "DummyAccessKeyID",
			SecretAccessKey: "DummySecretAccessKey",
			SessionToken:    "DummySessionToken",
			ProviderName:    "DummyProviderName",
		},
		expiresAt: time.Now().Add(1 * time.Minute),
	}

	cacheHome, err := ioutil.TempDir("/tmp", "file-cache-provider-test-")
	if err != nil {
		t.Fatalf("error making temporary directory - %s", err)
	}
	defer (func() {
		if os.RemoveAll(cacheHome) != nil {
			t.Logf("failed to remove %s\n", cacheHome)
		}
	})()

	fileCacheProvider := FileCacheProvider{
		credentials: &creds,
		cacheHome:   cacheHome,
	}

	_, err = fileCacheProvider.Retrieve()
	if err != nil {
		t.Fatalf("fileCacheProvider.Retrieve returned non nil error - %s\n", err)
	}
	if creds.getCalls != 1 {
		t.Errorf("fileCacheProvider.Retrieve should call Get on the credentials API exactly once with no cache but it was called %d times", creds.getCalls)
	}

	_, err = fileCacheProvider.Retrieve()
	if creds.getCalls != 1 {
		t.Errorf("fileCacheProvider.Retrieve should call Get on the credentials API exactly once with no cache but it was called %d times", creds.getCalls)
	}
}

func TestRetreiveCachingFileError(t *testing.T) {
	creds := credentialsMock{
		credentialsValue: credentials.Value{
			AccessKeyID:     "DummyAccessKeyID",
			SecretAccessKey: "DummySecretAccessKey",
			SessionToken:    "DummySessionToken",
			ProviderName:    "DummyProviderName",
		},
		expiresAt: time.Now().Add(1 * time.Minute),
	}

	cacheHome, err := ioutil.TempDir("/tmp", "file-cache-provider-test-")
	if err != nil {
		t.Fatalf("error making temporary directory - %s", err)
	}
	defer (func() {
		if os.RemoveAll(cacheHome) != nil {
			t.Logf("failed to remove %s\n", cacheHome)
		}
	})()

	fileCacheProvider := FileCacheProvider{
		credentials: &creds,
		cacheHome:   cacheHome,
	}

	_, err = fileCacheProvider.Retrieve()

	ioutil.WriteFile(path.Join(cacheHome, "s3parcp", "credentials-cache.json"), []byte("junk"), os.ModePerm)

	_, err = fileCacheProvider.Retrieve()
	if creds.getCalls != 2 {
		t.Error("expected fileCacheProvider.Retrieve to refresh credentials if credentials file is invalid")
	}
}

func TestRetreiveCachingThreadSafety(t *testing.T) {
	creds := credentialsMock{
		credentialsValue: credentials.Value{
			AccessKeyID:     "DummyAccessKeyID",
			SecretAccessKey: "DummySecretAccessKey",
			SessionToken:    "DummySessionToken",
			ProviderName:    "DummyProviderName",
		},
		// set the credentials to always be expired, this will trigger a write to the cache with every call to Retrieve
		expiresAt: time.Now(),
	}

	cacheHome, err := ioutil.TempDir("/tmp", "file-cache-provider-test-")
	if err != nil {
		t.Fatalf("error making temporary directory - %s", err)
	}
	defer (func() {
		if os.RemoveAll(cacheHome) != nil {
			t.Logf("failed to remove %s\n", cacheHome)
		}
	})()

	fileCacheProvider := FileCacheProvider{
		credentials: &creds,
		cacheHome:   cacheHome,
	}

	retrievalErrors := int32(0)
	parsingErrors := int32(0)

	// call retrieve with many threads
	var wg sync.WaitGroup
	for i := 1; i <= 100; i++ {
		wg.Add(1)
		go (func() {
			defer wg.Done()
			_, err := fileCacheProvider.Retrieve()
			if err != nil {
				atomic.AddInt32(&retrievalErrors, 1)
			}

			cachedCreds := cachedCredentials{}
			bytes, err := ioutil.ReadFile(fileCacheProvider.cacheFilename())
			err = json.Unmarshal(bytes, &cachedCreds)
			if err != nil {
				atomic.AddInt32(&parsingErrors, 1)
			}
		})()
	}
	wg.Wait()

	// ensure that we fetched new credentials in each thread
	// the count is updated atomically so this should always be true
	if creds.getCalls != 100 {
		t.Errorf("expected Get to be called 100 times but it was called %d times", creds.getCalls)
	}

	if retrievalErrors > 0 {
		t.Errorf("concurrent Retrieve calls errored %d times", retrievalErrors)
	}

	if parsingErrors > 0 {
		t.Errorf("concurrent Retrieve calls resulted in a corrupted cache file %d times", parsingErrors)
	}
}
