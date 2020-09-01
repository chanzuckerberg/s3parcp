package filecachedcredentials

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/aws/aws-sdk-go/aws/credentials"
)

type credentialsGetter interface {
	ExpiresAt() (time.Time, error)
	Get() (credentials.Value, error)
	IsExpired() bool
}

// FileCacheProvider provides credentials from a file cache with a fallback
type FileCacheProvider struct {
	credentials credentialsGetter
	cacheHome   string
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
	credentials, err := f.credentials.Get()
	if err != nil {
		message := "Encountered error while fetching credentials\n"
		os.Stderr.WriteString(message)
		os.Stderr.WriteString(err.Error() + "\n")
		return cachedCredentials{}, err
	}

	expiresAt, expirationError := f.credentials.ExpiresAt()

	cachedCreds := cachedCredentials{
		AccessKeyID:     credentials.AccessKeyID,
		ExpiresAt:       expiresAt,
		ProviderName:    credentials.ProviderName,
		SecretAccessKey: credentials.SecretAccessKey,
		SessionToken:    credentials.SessionToken,
	}

	// If we get an error fetching the expiry don't save credentials
	//   but still return new credentials. If they were saved they
	//   would just be expired the next time so no point in saving them.
	if expirationError == nil {
		err = writeCacheFile(cacheFilename, cachedCreds)
	}
	return cachedCreds, err
}

// NewFileCacheProvider creates a new FileCacheProvider with the os.UserCacheDir as the cacheHome
func NewFileCacheProvider(credentials credentialsGetter) (FileCacheProvider, error) {
	cacheHome, err := os.UserCacheDir()
	if err != nil {
		message := "Error: encountered error while getting user cache directory\n"
		os.Stderr.WriteString(message)
		os.Stderr.WriteString(err.Error() + "\n")
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
	return f.credentials.IsExpired()
}
