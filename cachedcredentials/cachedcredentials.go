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
		message := fmt.Sprintf("Error: encountered error while opening credentials file %s\n", cacheFilename)
		os.Stderr.WriteString(message)
		os.Stderr.WriteString(err.Error() + "\n")
		return err
	}

	err = syscall.Flock(fd, syscall.LOCK_EX)
	if err != nil {
		message := fmt.Sprintf("Error: encountered error while requesting a lock on credentials file %s\n", cacheFilename)
		os.Stderr.WriteString(message)
		os.Stderr.WriteString(err.Error() + "\n")
		return err
	}

	_, writeErr := syscall.Write(fd, data)

	err = syscall.Flock(fd, syscall.LOCK_UN)

	if writeErr != nil {
		message := fmt.Sprintf("Error: encountered error while writing credentials file %s\n", cacheFilename)
		os.Stderr.WriteString(message)
		os.Stderr.WriteString(err.Error() + "\n")
		return writeErr
	}

	if err != nil {
		message := fmt.Sprintf("Error: encountered error while unlocking credentials file %s\n", cacheFilename)
		os.Stderr.WriteString(message)
		os.Stderr.WriteString(err.Error() + "\n")
		return err
	}

	return nil
}

func readCacheFile(cacheFilename string) (cachedCredentials, error) {
	cachedCreds := cachedCredentials{}
	bytes, err := ioutil.ReadFile(cacheFilename)
	if err != nil {
		message := fmt.Sprintf("Error: encountered error while reading cached credentials file %s\n", cacheFilename)
		os.Stderr.WriteString(message)
		os.Stderr.WriteString(err.Error() + "\n")
		return cachedCreds, err
	}
	err = json.Unmarshal(bytes, &cachedCreds)
	if err != nil {
		message := fmt.Sprintf("Error: encountered error while parsing cached credentials file %s\n", cacheFilename)
		os.Stderr.WriteString(message)
		os.Stderr.WriteString(err.Error() + "\n")
		return cachedCreds, err
	}
	return cachedCreds, nil
}

func (f *FileCacheProvider) refreshCredentials(cacheFilename string) (cachedCredentials, error) {
	credentials, err := f.Creds.Get()
	if err != nil {
		message := "Encountered error while fetching credentials\n"
		os.Stderr.WriteString(message)
		os.Stderr.WriteString(err.Error() + "\n")
		return cachedCredentials{}, err
	}

	expiresAt, err := f.Creds.ExpiresAt()
	if err != nil {
		message := "Encountered error while fetching credential expiration date\n"
		os.Stderr.WriteString(message)
		os.Stderr.WriteString(err.Error() + "\n")
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
	cacheHome, err := os.UserCacheDir()
	if err != nil {
		message := "Error: encountered error while getting user cache directory\n"
		os.Stderr.WriteString(message)
		os.Stderr.WriteString(err.Error() + "\n")
		return credentials.Value{}, err
	}

	cacheDir := path.Join(cacheHome, "s3parcp")
	err = os.MkdirAll(cacheDir, os.ModePerm)
	if err != nil {
		return credentials.Value{}, err
	}

	cacheFilename := path.Join(cacheDir, "credentials-cache.json")

	useCache, err := fileExists(cacheFilename)
	if err != nil {
		message := fmt.Sprintf("Error: encountered error while checking for existence of cached credentials file %s\n", cacheFilename)
		os.Stderr.WriteString(message)
		os.Stderr.WriteString(err.Error() + "\n")
		return credentials.Value{}, err
	}

	cachedCreds := cachedCredentials{}

	if useCache {
		cachedCreds, err = readCacheFile(cacheFilename)
		if err != nil {
			message := "Warning: encountered error while reading cache file, ignoring and using fresh credentials\n"
			os.Stderr.WriteString(message)
		}
		useCache = useCache && err == nil
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
