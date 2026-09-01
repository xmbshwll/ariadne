package auth

import (
	"context"
	"errors"
	"strings"
	"sync"

	"golang.org/x/sync/singleflight"
)

// DiscoveredCredential is a Credential Token that is discovered rather than
// fetched from a token endpoint: the holder supplies a discover function - scan
// a public page, follow an asset script, whatever the Music Service needs -
// and this type caches the result, shares one in-flight discovery between
// concurrent callers, and supports invalidation when the service rejects the
// credential. Observe records a credential seen incidentally in a payload the
// holder fetched anyway, so a discovery round trip is often unnecessary.
//
// The zero value is not usable; build one with NewDiscoveredCredential.
type DiscoveredCredential struct {
	discover func(context.Context) (string, error)
	// invalidationErr classifies the service rejections that should drop the
	// cached credential; nil means every cached value stays until invalidated
	// explicitly.
	invalidationErr func(error) bool

	group singleflight.Group
	mu    sync.RWMutex
	value string
}

// NewDiscoveredCredential builds a discovered credential around a discover
// function. The function must return the credential or an error; it is called
// at most once per invalidation window.
func NewDiscoveredCredential(discover func(context.Context) (string, error), invalidationErr func(error) bool) *DiscoveredCredential {
	return &DiscoveredCredential{discover: discover, invalidationErr: invalidationErr}
}

// Credential returns the cached credential, discovering one when the cache is
// empty. A discovery error passes through unchanged.
func (d *DiscoveredCredential) Credential(ctx context.Context) (string, error) {
	if d == nil || d.discover == nil {
		return "", errDiscoveredCredentialNotConfigured
	}
	if value := d.cached(); value != "" {
		return value, nil
	}

	resultCh := d.group.DoChan("discovered", func() (any, error) {
		if value := d.cached(); value != "" {
			return value, nil
		}
		value, err := d.discover(ctx)
		if err != nil {
			return "", err
		}
		d.store(value)
		return value, nil
	})
	result := <-resultCh
	if result.Err != nil {
		return "", result.Err
	}
	value, ok := result.Val.(string)
	if !ok {
		return "", errDiscoveredCredentialNotConfigured
	}
	return value, nil
}

// Observe records a credential seen incidentally in a payload the holder
// fetched for other reasons, so discovery can be skipped entirely.
func (d *DiscoveredCredential) Observe(value string) {
	value = strings.TrimSpace(value)
	if value == "" || d == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.value == "" {
		d.value = value
	}
}

// Invalidate drops the cached credential. The next Credential call rediscovers.
func (d *DiscoveredCredential) Invalidate() {
	if d == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.value = ""
}

// ShouldInvalidate answers whether a service error means the cached credential
// was rejected and should be dropped.
func (d *DiscoveredCredential) ShouldInvalidate(err error) bool {
	if d == nil || d.invalidationErr == nil || err == nil {
		return false
	}
	return d.invalidationErr(err)
}

func (d *DiscoveredCredential) cached() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.value
}

func (d *DiscoveredCredential) store(value string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.value == "" {
		d.value = value
	}
}

var errDiscoveredCredentialNotConfigured = errors.New("discovered credential not configured")
