package filecachedcredentials

import (
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws/credentials"
)

type credentialsMocker struct {
	credentialsValue credentials.Value
	err              error
	expiresAt        time.Time
	expiresAtCalls   int
	getCalls         int
	isExpiredCalls   int
}

func (c *credentialsMocker) ExpiresAt() (time.Time, error) {
	c.expiresAtCalls++
	return c.expiresAt, c.err
}

func (c *credentialsMocker) Get() (credentials.Value, error) {
	c.getCalls++
	return c.credentialsValue, c.err
}

func (c *credentialsMocker) IsExpired() bool {
	fmt.Println(c.isExpiredCalls)
	c.isExpiredCalls++
	fmt.Println(c.expiresAt)
	return c.expiresAt.Before(time.Now())
}

func TestIsExpiredExpired(t *testing.T) {
	c := credentialsMocker{
		err:       nil,
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
	c := credentialsMocker{
		err:       nil,
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

func TestCacheFileRaceCondition(t *testing.T) {

}
