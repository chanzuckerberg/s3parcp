package filecachedcredentials

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path"
	"time"

	"github.com/aws/aws-sdk-go/aws/credentials"
)

// awsCredentials is an interface with the required functions from credentials.credentials.
// An interface version is required for mocking
type awsCredentials interface {
	ExpiresAt() (time.Time, error)
	Get() (credentials.Value, error)
	IsExpired() bool
}

// FileCacheProvider provides credentials from a file cache with a fallback
type FileCacheProvider struct {
	credentials awsCredentials
	cacheHome   string
}

type cachedCredentials struct {
	AccessKeyID     string
	ExpiresAt       time.Time
	ProviderName    string
	SecretAccessKey string
	SessionToken    string
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
	return path.Join(f.cacheHome, "s3parcp", "credentials-cache.json")
}

// writeCacheFile writes cachedCredentials to a file atomically
func saveCachedCredentials(cacheFilename string, cachedCreds cachedCredentials) error {
	data, err := json.Marshal(cachedCreds)
	if err != nil {

		return err
	}

	tmp, err := ioutil.TempFile(path.Dir(cacheFilename), "tmp-credentials-cache-")
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(tmp.Name(), data, os.ModePerm)
	if err != nil {
		return err
	}

	return os.Rename(tmp.Name(), cacheFilename)
}

func loadCachedCredentials(cacheFilename string) (cachedCredentials, error) {
	cachedCreds := cachedCredentials{}
	bytes, err := ioutil.ReadFile(cacheFilename)
	if err != nil {
		log.Printf("error while reading cached credentials file %s - %s\n", cacheFilename, err)
		return cachedCreds, err
	}
	err = json.Unmarshal(bytes, &cachedCreds)
	if err != nil {
		log.Printf("error parsing cached credentials file %s - %s\n", cacheFilename, err)
		return cachedCreds, err
	}
	return cachedCreds, nil
}

func (f *FileCacheProvider) refreshCredentials(cacheFilename string) (credentials.Value, error) {
	newCredentials, err := f.credentials.Get()
	if err != nil {
		log.Printf("error while fetching credentials - %s", err)
		return credentials.Value{}, err
	}

	expiresAt, err := f.credentials.ExpiresAt()

	if err != nil {
		log.Printf("error fetching credential expiry - %s, credentials will not be cached", err)
		return newCredentials, nil
	}

	cachedCreds := cachedCredentials{
		AccessKeyID:     newCredentials.AccessKeyID,
		ExpiresAt:       expiresAt,
		ProviderName:    newCredentials.ProviderName,
		SecretAccessKey: newCredentials.SecretAccessKey,
		SessionToken:    newCredentials.SessionToken,
	}

	err = writeCacheFile(cacheFilename, cachedCreds)
	if err != nil {
		log.Printf("error writing credential cache file - %s, credentials will not be cached", err)
	}

	return newCredentials, nil
}

// NewFileCacheProvider creates a new FileCacheProvider with the os.UserCacheDir as the cacheHome
func NewFileCacheProvider(credentials awsCredentials) (FileCacheProvider, error) {
	cacheHome, err := os.UserCacheDir()
	if err != nil {
		log.Printf("error getting user cache directory - %s", err)
	}

	return FileCacheProvider{
		credentials: credentials,
		cacheHome:   cacheHome,
	}, err
}

// Retrieve retrieves credentials
func (f *FileCacheProvider) Retrieve() (credentials.Value, error) {
	cacheDir := path.Join(f.cacheHome, "s3parcp")
	err := os.MkdirAll(cacheDir, os.ModePerm)
	if err != nil {
		return credentials.Value{}, err
	}

	cacheFilename := path.Join(cacheDir, "credentials-cache.json")

	cacheFileExists, err := fileExists(cacheFilename)
	if err != nil {
		log.Printf("error while checking for existence of cached credentials file %s - %s, refreshing credentials", cacheFilename, err)
	}
	if err != nil || !cacheFileExists {
		return f.refreshCredentials(cacheFilename)
	}

	cachedCreds, err := loadCachedCredentials(cacheFilename)
	if err != nil {
		log.Println("error loading cached credentials, refreshing credentials")
	}
	if err == nil && !cachedCreds.IsExpired() {
		return credentials.Value{
			AccessKeyID:     cachedCreds.AccessKeyID,
			ProviderName:    cachedCreds.ProviderName,
			SecretAccessKey: cachedCreds.SecretAccessKey,
			SessionToken:    cachedCreds.SessionToken,
		}, err
	}

	return f.refreshCredentials(cacheFilename)
}

// IsExpired checks if the credentials are expired
func (f *FileCacheProvider) IsExpired() bool {
	return f.credentials.IsExpired()
}
