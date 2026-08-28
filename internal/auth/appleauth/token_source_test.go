package appleauth_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xmbshwll/ariadne/internal/auth/appleauth"
)

// writeKeyMaterial writes a real .p8 key once, since signing is the slow part.
func writeKeyMaterial(t *testing.T) string {
	t.Helper()
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	der, err := x509.MarshalPKCS8PrivateKey(privateKey)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "AuthKey_TEST.p8")
	require.NoError(t, os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), 0o600))
	return path
}

func TestTokenSourceCachesTheGeneratedToken(t *testing.T) {
	source := appleauth.NewTokenSource(appleauth.Config{
		KeyID:          "KEY",
		TeamID:         "TEAM",
		PrivateKeyPath: writeKeyMaterial(t),
	})

	first, err := source.AccessToken(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, first)

	second, err := source.AccessToken(context.Background())
	require.NoError(t, err)
	assert.Equal(t, first, second, "the cached token is returned while it is valid")
}

func TestTokenSourceReportsUnconfiguredKeyMaterial(t *testing.T) {
	source := appleauth.NewTokenSource(appleauth.Config{})

	token, err := source.AccessToken(context.Background())

	assert.Empty(t, token)
	require.Error(t, err)
	assert.ErrorIs(t, err, appleauth.ErrDeveloperTokenSourceNotConfigured)
	assert.False(t, source.Configured())
}

func TestTokenSourceAcceptsConfiguredKeyMaterial(t *testing.T) {
	source := appleauth.NewTokenSource(appleauth.Config{
		KeyID:          "KEY",
		TeamID:         "TEAM",
		PrivateKeyPath: writeKeyMaterial(t),
	})

	assert.True(t, source.Configured())
}
