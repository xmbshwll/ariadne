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
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGenerateDeveloperToken(t *testing.T) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate private key: %v", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatalf("marshal private key: %v", err)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	keyPath := filepath.Join(t.TempDir(), "AuthKey_TEST12345.p8")
	if err := osWriteFile(keyPath, pemBytes); err != nil {
		t.Fatalf("write private key: %v", err)
	}

	now := time.Unix(1_700_000_000, 0).UTC()
	token, err := GenerateDeveloperToken(Config{
		KeyID:          "TEST12345",
		TeamID:         "TEAM123456",
		PrivateKeyPath: keyPath,
		TTL:            2 * time.Hour,
	}, now)
	if err != nil {
		t.Fatalf("GenerateDeveloperToken error: %v", err)
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("token parts = %d, want 3", len(parts))
	}

	var header map[string]any
	decodeJSONPart(t, parts[0], &header)
	if header["alg"] != "ES256" {
		t.Fatalf("alg = %v", header["alg"])
	}
	if header["kid"] != "TEST12345" {
		t.Fatalf("kid = %v", header["kid"])
	}

	var claims map[string]any
	decodeJSONPart(t, parts[1], &claims)
	if claims["iss"] != "TEAM123456" {
		t.Fatalf("iss = %v", claims["iss"])
	}
	if claims["iat"] != float64(now.Unix()) {
		t.Fatalf("iat = %v", claims["iat"])
	}
	if claims["exp"] != float64(now.Add(2*time.Hour).Unix()) {
		t.Fatalf("exp = %v", claims["exp"])
	}

	signingInput := parts[0] + "." + parts[1]
	hash := sha256.Sum256([]byte(signingInput))
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		t.Fatalf("decode signature: %v", err)
	}
	if len(signature) != 64 {
		t.Fatalf("signature length = %d, want 64", len(signature))
	}
	r := new(big.Int).SetBytes(signature[:32])
	s := new(big.Int).SetBytes(signature[32:])
	if !ecdsa.Verify(&privateKey.PublicKey, hash[:], r, s) {
		t.Fatalf("signature verification failed")
	}
}

func TestGenerateDeveloperTokenRequiresConfig(t *testing.T) {
	_, err := GenerateDeveloperToken(Config{}, time.Now())
	if err == nil {
		t.Fatalf("expected configuration error")
	}
}

func decodeJSONPart(t *testing.T, encoded string, target any) {
	t.Helper()
	payload, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("decode token part: %v", err)
	}
	if err := json.Unmarshal(payload, target); err != nil {
		t.Fatalf("unmarshal token part: %v", err)
	}
}

func osWriteFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o600)
}
