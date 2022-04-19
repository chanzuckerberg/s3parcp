package filecachedcredentials

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
)

type credentialsMock struct {
	credentialsValue aws.Credentials
	retrieveCallsErr error
	retrieveCalls    int32
}

func (c *credentialsMock) Retrieve(ctx context.Context) (aws.Credentials, error) {
	atomic.AddInt32(&(c.retrieveCalls), 1)
	return c.credentialsValue, c.retrieveCallsErr
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
		credentialsValue: aws.Credentials{
			AccessKeyID:     "DummyAccessKeyID",
			Expires:         time.Now().Add(1 * time.Minute),
			SecretAccessKey: "DummySecretAccessKey",
			SessionToken:    "DummySessionToken",
			Source:          "DummySource",
		},
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

	credResults, err := fileCacheProvider.Retrieve(context.Background())
	if err != nil {
		t.Fatalf("fileCacheProvider.Retrieve returned non nil error - %s", err)
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

	if credResults.Source != creds.credentialsValue.Source {
		t.Errorf("fileCacheProvider.Retrieve should return credentials from the credentials API - expected Source to be %s but it was %s",
			credResults.Source, creds.credentialsValue.Source)
	}

	if creds.retrieveCalls != 1 {
		t.Errorf("fileCacheProvider.Retrieve should call Retrieve on the credentials API exactly once with no cache but it was called %d times", creds.retrieveCalls)
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
		credentialsValue: aws.Credentials{
			AccessKeyID:     "DummyAccessKeyID",
			Expires:         time.Now().Add(1 * time.Minute),
			SecretAccessKey: "DummySecretAccessKey",
			SessionToken:    "DummySessionToken",
			Source:          "DummySource",
		},
	}

	cacheHome, err := ioutil.TempDir("/tmp", "file-cache-provider-test-")
	if err != nil {
		t.Fatalf("error making temporary directory - %s", err)
	}
	defer (func() {
		if os.RemoveAll(cacheHome) != nil {
			t.Logf("failed to remove %s", cacheHome)
		}
	})()

	fileCacheProvider := FileCacheProvider{
		credentials: &creds,
		cacheHome:   cacheHome,
	}

	_, err = fileCacheProvider.Retrieve(context.Background())
	if err != nil {
		t.Fatalf("fileCacheProvider.Retrieve returned non nil error on first call - %s", err)
	}
	if creds.retrieveCalls != 1 {
		t.Errorf("fileCacheProvider.Retrieve should call Retrieve on the credentials API exactly once with no cache but it was called %d times", creds.retrieveCalls)
	}

	_, err = fileCacheProvider.Retrieve(context.Background())
	if creds.retrieveCalls != 1 {
		t.Errorf("fileCacheProvider.Retrieve should call Retrieve on the credentials API exactly once with no cache but it was called %d times", creds.retrieveCalls)
	}
	if err != nil {
		t.Errorf("fileCacheProvider.Retrieve returned non nil error on second call - %s", err)
	}
}

func TestRetreiveCachingFileError(t *testing.T) {
	creds := credentialsMock{
		credentialsValue: aws.Credentials{
			AccessKeyID:     "DummyAccessKeyID",
			Expires:         time.Now().Add(1 * time.Minute),
			SecretAccessKey: "DummySecretAccessKey",
			SessionToken:    "DummySessionToken",
			Source:          "DummySource",
		},
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

	_, err = fileCacheProvider.Retrieve(context.Background())
	if err != nil {
		t.Errorf("fileCacheProvider.Retrieve returned non nil error on first call - %s", err)
	}

	err = ioutil.WriteFile(path.Join(cacheHome, "s3parcp", "credentials-cache.json"), []byte("junk"), os.ModePerm)
	if err != nil {
		t.Fatalf("error writing to cache file %s - %s", path.Join(cacheHome, "s3parcp", "credentials-cache.json"), err)
	}

	_, err = fileCacheProvider.Retrieve(context.Background())
	if creds.retrieveCalls != 2 {
		t.Error("expected fileCacheProvider.Retrieve to refresh credentials if credentials file is invalid")
	}
	if err != nil {
		t.Errorf("fileCacheProvider.Retrieve returned non nil error on second call - %s", err)
	}
}

func TestRetreiveCachingThreadSafety(t *testing.T) {
	creds := credentialsMock{
		credentialsValue: aws.Credentials{
			AccessKeyID: "DummyAccessKeyID",
			// set the credentials to always be expired, this will trigger a write to the cache with every call to Retrieve
			Expires:         time.Now(),
			SecretAccessKey: "DummySecretAccessKey",
			SessionToken:    "DummySessionToken",
			Source:          "DummySource",
		},
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
	fileReadErrors := int32(0)
	parsingErrors := int32(0)

	// call retrieve with many threads
	var wg sync.WaitGroup
	for i := 1; i <= 100; i++ {
		wg.Add(1)
		go (func() {
			defer wg.Done()
			_, err := fileCacheProvider.Retrieve(context.Background())
			if err != nil {
				atomic.AddInt32(&retrievalErrors, 1)
			}

			creds := aws.Credentials{}
			bytes, err := ioutil.ReadFile(fileCacheProvider.cacheFilename())
			if err != nil {
				atomic.AddInt32(&fileReadErrors, 1)
			}
			if json.Unmarshal(bytes, &creds) != nil {
				atomic.AddInt32(&parsingErrors, 1)
			}
		})()
	}
	wg.Wait()

	if retrievalErrors > 0 {
		t.Errorf("concurrent Retrieve calls returned an error %d/100 times", retrievalErrors)
	}

	if fileReadErrors > 0 {
		t.Errorf("concurrent Retrieve calls resulted in an unreadable cache file %d/100 times", fileReadErrors)
	}

	if parsingErrors > 0 {
		t.Errorf("concurrent Retrieve calls resulted in a corrupted cache file %d/100 times", parsingErrors)
	}
}
