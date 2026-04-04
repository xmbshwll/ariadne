package applemusicauth

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"
)

const defaultTokenTTL = time.Hour

type Config struct {
	KeyID          string
	TeamID         string
	PrivateKeyPath string
	TTL            time.Duration
}

func GenerateDeveloperToken(cfg Config, now time.Time) (string, error) {
	if err := cfg.validate(); err != nil {
		return "", err
	}
	privateKey, err := loadPrivateKey(cfg.PrivateKeyPath)
	if err != nil {
		return "", err
	}

	ttl := cfg.TTL
	if ttl <= 0 {
		ttl = defaultTokenTTL
	}

	header, err := json.Marshal(map[string]any{
		"alg": "ES256",
		"kid": cfg.KeyID,
		"typ": "JWT",
	})
	if err != nil {
		return "", fmt.Errorf("encode apple music token header: %w", err)
	}
	claims, err := json.Marshal(map[string]any{
		"iss": cfg.TeamID,
		"iat": now.Unix(),
		"exp": now.Add(ttl).Unix(),
	})
	if err != nil {
		return "", fmt.Errorf("encode apple music token claims: %w", err)
	}

	encodedHeader := base64.RawURLEncoding.EncodeToString(header)
	encodedClaims := base64.RawURLEncoding.EncodeToString(claims)
	signingInput := encodedHeader + "." + encodedClaims
	hash := sha256.Sum256([]byte(signingInput))
	r, s, err := ecdsa.Sign(rand.Reader, privateKey, hash[:])
	if err != nil {
		return "", fmt.Errorf("sign apple music token: %w", err)
	}
	signature, err := joseSignature(r, s, 32)
	if err != nil {
		return "", err
	}
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func (c Config) validate() error {
	if strings.TrimSpace(c.KeyID) == "" {
		return errors.New("apple music key id is required")
	}
	if strings.TrimSpace(c.TeamID) == "" {
		return errors.New("apple music team id is required")
	}
	if strings.TrimSpace(c.PrivateKeyPath) == "" {
		return errors.New("apple music private key path is required")
	}
	return nil
}

func loadPrivateKey(path string) (*ecdsa.PrivateKey, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read apple music private key: %w", err)
	}
	block, _ := pem.Decode(content)
	if block == nil {
		return nil, errors.New("decode apple music private key pem: no pem block found")
	}

	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		ecdsaKey, ok := key.(*ecdsa.PrivateKey)
		if !ok {
			return nil, errors.New("apple music private key is not an ecdsa key")
		}
		return ecdsaKey, nil
	}

	key, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse apple music private key: %w", err)
	}
	return key, nil
}

func joseSignature(r *big.Int, s *big.Int, size int) ([]byte, error) {
	rb := r.Bytes()
	sb := s.Bytes()
	if len(rb) > size || len(sb) > size {
		return nil, errors.New("apple music signature component too large")
	}
	signature := make([]byte, size*2)
	copy(signature[size-len(rb):size], rb)
	copy(signature[size*2-len(sb):], sb)
	return signature, nil
}
