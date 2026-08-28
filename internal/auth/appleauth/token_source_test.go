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

// TestTokenSourceAnswersTokensForItsKeyMaterial covers the caching source
// seam the applemusic adapter consumes: configured material yields a token and
// reuses it, unconfigured material answers the sentinel.
func TestTokenSourceAnswersTokensForItsKeyMaterial(t *testing.T) {
	keyPath := writeKeyMaterial(t)

	tests := []struct {
		name        string
		cfg         appleauth.Config
		wantErrText string
	}{
		{
			name: "configured material issues a token",
			cfg:  appleauth.Config{KeyID: "KEY", TeamID: "TEAM", PrivateKeyPath: keyPath},
		},
		{
			name:        "missing key material answers the sentinel",
			cfg:         appleauth.Config{},
			wantErrText: "apple music developer token source not configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := appleauth.NewTokenSource(tt.cfg)

			first, err := source.AccessToken(context.Background())

			if tt.wantErrText != "" {
				require.ErrorContains(t, err, tt.wantErrText, tt.name)
				assert.False(t, source.Configured(), tt.name)
				return
			}

			require.NoError(t, err, tt.name)
			assert.NotEmpty(t, first, tt.name)
			assert.True(t, source.Configured(), tt.name)

			second, err := source.AccessToken(context.Background())
			require.NoError(t, err, tt.name)
			assert.Equal(t, first, second, "the cached token is returned while it is valid")
		})
	}
}
