package cachedcredentials

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go/aws/credentials"
)

// FileCacheProvider provides credentials from a file cache with a fallback
type FileCacheProvider struct {
	Creds *credentials.Credentials
}

type cachedCredentials struct {
	AccessKeyID     string
	ExpiresAt       time.Time
	ProviderName    string
	SecretAccessKey string
	SessionToken    string
}

func fileExists(name string) (bool, error) {
	_, err := os.Stat(name)
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

// writeCacheFile writes cachedCredentials to a file atomically
func writeCacheFile(cacheFilename string, cachedCreds cachedCredentials) error {
	data, err := json.Marshal(cachedCreds)
	if err != nil {
		return err
	}

	fd, err := syscall.Open(cacheFilename, syscall.O_CREAT|syscall.O_RDWR, 0600)
	defer syscall.Close(fd)
	if err != nil {
		return err
	}

	err = syscall.Flock(fd, syscall.LOCK_EX)
	defer syscall.Flock(fd, syscall.LOCK_UN)
	if err != nil {
		return err
	}

	_, err = syscall.Write(fd, data)
	return err
}

func readCacheFile(cacheFilename string) (cachedCredentials, error) {
	cachedCreds := cachedCredentials{}
	bytes, err := ioutil.ReadFile(cacheFilename)
	if err != nil {
		message := fmt.Sprintf("Encountered error while reading cached credentials file %s\n", cacheFilename)
		os.Stderr.WriteString(message)
		os.Stderr.WriteString(err.Error() + "\n")
		return cachedCreds, err
	}
	err = json.Unmarshal(bytes, &cachedCreds)
	if err != nil {
		message := fmt.Sprintf("Encountered error while parsing cached credentials file %s\n", cacheFilename)
		os.Stderr.WriteString(message)
		os.Stderr.WriteString(err.Error() + "\n")
		return cachedCreds, err
	}
	return cachedCreds, err
}

func (f *FileCacheProvider) refreshCredentials(cacheFilename string) (cachedCredentials, error) {
	credentials, err := f.Creds.Get()
	if err != nil {
		return cachedCredentials{}, err
	}

	expiresAt, err := f.Creds.ExpiresAt()
	if err != nil {
		return cachedCredentials{}, err
	}

	cachedCreds := cachedCredentials{
		AccessKeyID:     credentials.AccessKeyID,
		ExpiresAt:       expiresAt,
		ProviderName:    credentials.ProviderName,
		SecretAccessKey: credentials.SecretAccessKey,
		SessionToken:    credentials.SessionToken,
	}

	err = writeCacheFile(cacheFilename, cachedCreds)
	return cachedCreds, err
}

// Retrieve retrieves credentials
func (f *FileCacheProvider) Retrieve() (credentials.Value, error) {
	cacheDir := path.Join(os.Getenv("HOME"), ".s3parcp")
	err := os.MkdirAll(cacheDir, os.ModePerm)
	if err != nil {
		return credentials.Value{}, err
	}

	cacheFilename := path.Join(cacheDir, "credentials-cache.json")

	useCache, err := fileExists(cacheFilename)
	if err != nil {
		return credentials.Value{}, err
	}

	cachedCreds := cachedCredentials{}

	if useCache {
		cachedCreds, err = readCacheFile(cacheFilename)
		useCache = useCache && err != nil
	}

	useCache = useCache && cachedCreds.ExpiresAt.After(time.Now())

	if !useCache {
		cachedCreds, err = f.refreshCredentials(cacheFilename)
	}

	return credentials.Value{
		AccessKeyID:     cachedCreds.AccessKeyID,
		ProviderName:    cachedCreds.ProviderName,
		SecretAccessKey: cachedCreds.SecretAccessKey,
		SessionToken:    cachedCreds.SessionToken,
	}, err
}

// IsExpired checks if the credentials are expired
func (f *FileCacheProvider) IsExpired() bool {
	return f.Creds.IsExpired()
}
