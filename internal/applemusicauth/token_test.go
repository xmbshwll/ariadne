package applemusicauth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateDeveloperToken(t *testing.T) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	der, err := x509.MarshalPKCS8PrivateKey(privateKey)
	require.NoError(t, err)
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	keyPath := filepath.Join(t.TempDir(), "AuthKey_TEST12345.p8")
	require.NoError(t, osWriteFile(keyPath, pemBytes))

	now := time.Unix(1_700_000_000, 0).UTC()
	token, err := GenerateDeveloperToken(Config{
		KeyID:          "TEST12345",
		TeamID:         "TEAM123456",
		PrivateKeyPath: keyPath,
		TTL:            2 * time.Hour,
	}, now)
	require.NoError(t, err)

	parts := strings.Split(token, ".")
	require.Len(t, parts, 3)

	var header map[string]any
	decodeJSONPart(t, parts[0], &header)
	assert.Equal(t, "ES256", header["alg"])
	assert.Equal(t, "TEST12345", header["kid"])

	var claims map[string]any
	decodeJSONPart(t, parts[1], &claims)
	assert.Equal(t, "TEAM123456", claims["iss"])
	assert.Equal(t, float64(now.Unix()), claims["iat"])
	assert.Equal(t, float64(now.Add(2*time.Hour).Unix()), claims["exp"])

	signingInput := parts[0] + "." + parts[1]
	hash := sha256.Sum256([]byte(signingInput))
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	require.NoError(t, err)
	require.Len(t, signature, 64)
	r := new(big.Int).SetBytes(signature[:32])
	s := new(big.Int).SetBytes(signature[32:])
	assert.True(t, ecdsa.Verify(&privateKey.PublicKey, hash[:], r, s))
}

func TestGenerateDeveloperTokenRequiresConfig(t *testing.T) {
	_, err := GenerateDeveloperToken(Config{}, time.Now())
	require.Error(t, err)
}

func decodeJSONPart(t *testing.T, encoded string, target any) {
	t.Helper()
	payload, err := base64.RawURLEncoding.DecodeString(encoded)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(payload, target))
}

func osWriteFile(path string, data []byte) error {
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write file %s: %w", path, err)
	}
	return nil
}
