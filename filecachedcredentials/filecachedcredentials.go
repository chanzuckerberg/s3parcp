package filecachedcredentials

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
)

// FileCacheProvider provides credentials from a file cache with a fallback
type FileCacheProvider struct {
	credentials aws.CredentialsProvider
	cacheHome   string
}

type cachedCredentials struct {
	AccessKeyID     string
	ExpiresAt       time.Time
	SecretAccessKey string
	SessionToken    string
	Source          string
}

func (c cachedCredentials) IsExpired() bool {
	return c.ExpiresAt.Before(time.Now())
}

func fileExists(name string) (bool, error) {
	_, err := os.Stat(name)
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

func (f *FileCacheProvider) cacheFilename() string {
	return path.Join(f.cacheDirname(), "credentials-cache.json")
}

func (f *FileCacheProvider) cacheDirname() string {
	return path.Join(f.cacheHome, "s3parcp")
}

// saveCachedCredentials writes cachedCredentials to a file atomically.
// To make it atomic the data is written to a temporary file then that file is renamed to
// the name of the cache file. This prevents the cache file from being corrupted.
func (f *FileCacheProvider) saveCachedCredentials(cachedCreds cachedCredentials) error {
	data, err := json.Marshal(cachedCreds)
	if err != nil {
		log.Printf("error marshaling cached credentials json %s - %s\n", cachedCreds, err)
		return err
	}

	tmp, err := ioutil.TempFile(f.cacheDirname(), "tmp-credentials-cache-")
	if err != nil {
		log.Printf("error creating temporary file in %s - %s\n", f.cacheDirname(), err)
		return err
	}

	err = ioutil.WriteFile(tmp.Name(), data, os.ModePerm)
	if err != nil {
		log.Printf("error writing credentials to file %s - %s\n", tmp.Name(), err)
		return err
	}

	err = os.Rename(tmp.Name(), f.cacheFilename())
	if err != nil {
		log.Printf("error renaming temporary credentials file %s to %s - %s\n", tmp.Name(), f.cacheFilename(), err)
	}
	return err
}

func (f *FileCacheProvider) loadCachedCredentials() (cachedCredentials, error) {
	cachedCreds := cachedCredentials{}
	bytes, err := ioutil.ReadFile(f.cacheFilename())
	if err != nil {
		log.Printf("error while reading cached credentials file %s - %s\n", f.cacheFilename(), err)
		return cachedCreds, err
	}
	err = json.Unmarshal(bytes, &cachedCreds)
	if err != nil {
		log.Printf("error parsing cached credentials file %s - %s\n", f.cacheFilename(), err)
		return cachedCreds, err
	}
	return cachedCreds, nil
}

func (f *FileCacheProvider) refreshCredentials(ctx context.Context) (aws.Credentials, error) {
	newCredentials, err := f.credentials.Retrieve(ctx)
	if err != nil {
		log.Printf("error while fetching credentials - %s\n", err)
		return aws.Credentials{}, err
	}

	cachedCreds := cachedCredentials{
		AccessKeyID:     newCredentials.AccessKeyID,
		ExpiresAt:       newCredentials.Expires,
		SecretAccessKey: newCredentials.SecretAccessKey,
		SessionToken:    newCredentials.SessionToken,
		Source:          newCredentials.Source,
	}

	err = f.saveCachedCredentials(cachedCreds)
	if err != nil {
		log.Println("error saving credentials, credentials will not be cached")
	}

	return newCredentials, nil
}

// NewFileCacheProvider creates a new FileCacheProvider with the os.UserCacheDir as the cacheHome
func NewFileCacheProvider(credentials aws.CredentialsProvider) (FileCacheProvider, error) {
	cacheHome, err := os.UserCacheDir()
	if err != nil {
		log.Printf("error getting user cache directory - %s\n", err)
	}

	return FileCacheProvider{
		credentials: credentials,
		cacheHome:   cacheHome,
	}, err
}

// Retrieve retrieves credentials
func (f *FileCacheProvider) Retrieve(ctx context.Context) (aws.Credentials, error) {
	err := os.MkdirAll(f.cacheDirname(), os.ModePerm)
	if err != nil {
		return aws.Credentials{}, err
	}

	cacheFileExists, err := fileExists(f.cacheFilename())
	if err != nil {
		log.Printf("error while checking for existence of cached credentials file %s - %s, refreshing credentials\n", f.cacheFilename(), err)
	}
	if err != nil || !cacheFileExists {
		return f.refreshCredentials(ctx)
	}

	cachedCreds, err := f.loadCachedCredentials()
	if err != nil {
		log.Println("error loading cached credentials, refreshing credentials")
	}
	if err == nil && !cachedCreds.IsExpired() {
		return aws.Credentials{
			AccessKeyID:     cachedCreds.AccessKeyID,
			SecretAccessKey: cachedCreds.SecretAccessKey,
			SessionToken:    cachedCreds.SessionToken,
			Source:          cachedCreds.Source,
		}, err
	}

	return f.refreshCredentials(ctx)
}
