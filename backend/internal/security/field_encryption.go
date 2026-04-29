package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"golang.org/x/crypto/hkdf"
)

const (
	EncryptedValuePrefix = "enc:v1:"
	hkdfInfo             = "firemailplus/db-field-encryption/v1"
	defaultJWTSecret     = "your-secret-key"
)

var (
	ErrEncryptionNotConfigured = errors.New("field encryption is not configured")

	globalMu        sync.RWMutex
	globalEncryptor *FieldEncryptor
	globalSource    string
)

type FieldEncryptor struct {
	aead cipher.AEAD
}

type EncryptionConfig struct {
	EncryptionKey string
	JWTSecret     string
	Environment   string
}

type EncryptionStatus struct {
	Configured bool
	Source     string
	WeakKey    bool
}

func ConfigureFieldEncryption(cfg EncryptionConfig) (EncryptionStatus, error) {
	key, source, err := resolveEncryptionKey(cfg)
	if err != nil {
		return EncryptionStatus{}, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return EncryptionStatus{}, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return EncryptionStatus{}, fmt.Errorf("failed to create AES-GCM cipher: %w", err)
	}

	globalMu.Lock()
	globalEncryptor = &FieldEncryptor{aead: aead}
	globalSource = source
	globalMu.Unlock()

	return EncryptionStatus{
		Configured: true,
		Source:     source,
		WeakKey:    source == "jwt_derived" && IsWeakJWTSecret(cfg.JWTSecret),
	}, nil
}

func ConfigureFieldEncryptionFromEnv() (EncryptionStatus, error) {
	return ConfigureFieldEncryption(EncryptionConfig{
		EncryptionKey: os.Getenv("ENCRYPTION_KEY"),
		JWTSecret:     getEnv("JWT_SECRET", defaultJWTSecret),
		Environment:   os.Getenv("ENV"),
	})
}

func FieldEncryptionStatus() EncryptionStatus {
	globalMu.RLock()
	defer globalMu.RUnlock()

	return EncryptionStatus{
		Configured: globalEncryptor != nil,
		Source:     globalSource,
	}
}

func IsWeakJWTSecret(secret string) bool {
	switch strings.TrimSpace(secret) {
	case "", defaultJWTSecret, "your_jwt_secret_key_here", "your_jwt_secret_key_change_this_in_production":
		return true
	default:
		return false
	}
}

func IsEncryptedString(value string) bool {
	return strings.HasPrefix(value, EncryptedValuePrefix)
}

func EncryptString(plaintext string) (string, error) {
	if plaintext == "" || IsEncryptedString(plaintext) {
		return plaintext, nil
	}

	encryptor, err := getGlobalEncryptor()
	if err != nil {
		return "", err
	}
	return encryptor.EncryptString(plaintext)
}

func DecryptString(value string) (plaintext string, legacy bool, err error) {
	if value == "" {
		return "", false, nil
	}
	if !IsEncryptedString(value) {
		return value, true, nil
	}

	encryptor, err := getGlobalEncryptor()
	if err != nil {
		return "", false, err
	}

	plaintext, err = encryptor.DecryptString(value)
	return plaintext, false, err
}

func (e *FieldEncryptor) EncryptString(plaintext string) (string, error) {
	nonce := make([]byte, e.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate encryption nonce: %w", err)
	}

	ciphertext := e.aead.Seal(nil, nonce, []byte(plaintext), nil)
	payload := append(nonce, ciphertext...)
	return EncryptedValuePrefix + base64.RawURLEncoding.EncodeToString(payload), nil
}

func (e *FieldEncryptor) DecryptString(value string) (string, error) {
	encoded := strings.TrimPrefix(value, EncryptedValuePrefix)
	payload, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("invalid encrypted value encoding: %w", err)
	}

	nonceSize := e.aead.NonceSize()
	if len(payload) <= nonceSize {
		return "", errors.New("invalid encrypted value payload")
	}

	nonce := payload[:nonceSize]
	ciphertext := payload[nonceSize:]
	plaintext, err := e.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt encrypted value: %w", err)
	}

	return string(plaintext), nil
}

func getGlobalEncryptor() (*FieldEncryptor, error) {
	globalMu.RLock()
	encryptor := globalEncryptor
	globalMu.RUnlock()
	if encryptor != nil {
		return encryptor, nil
	}

	if _, err := ConfigureFieldEncryptionFromEnv(); err != nil {
		return nil, err
	}

	globalMu.RLock()
	defer globalMu.RUnlock()
	if globalEncryptor == nil {
		return nil, ErrEncryptionNotConfigured
	}
	return globalEncryptor, nil
}

func resolveEncryptionKey(cfg EncryptionConfig) ([]byte, string, error) {
	if strings.TrimSpace(cfg.EncryptionKey) != "" {
		key, err := parseEncryptionKey(cfg.EncryptionKey)
		if err != nil {
			return nil, "", err
		}
		return key, "encryption_key", nil
	}

	jwtSecret := strings.TrimSpace(cfg.JWTSecret)
	if jwtSecret == "" {
		return nil, "", errors.New("JWT_SECRET is required for database field encryption")
	}

	reader := hkdf.New(sha256.New, []byte(jwtSecret), nil, []byte(hkdfInfo))
	key := make([]byte, 32)
	if _, err := io.ReadFull(reader, key); err != nil {
		return nil, "", fmt.Errorf("failed to derive encryption key from JWT_SECRET: %w", err)
	}
	return key, "jwt_derived", nil
}

func parseEncryptionKey(value string) ([]byte, error) {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) == 32 {
		return []byte(trimmed), nil
	}

	decoders := []struct {
		name   string
		decode func(string) ([]byte, error)
	}{
		{name: "base64url_raw", decode: base64.RawURLEncoding.DecodeString},
		{name: "base64url", decode: base64.URLEncoding.DecodeString},
		{name: "base64_raw", decode: base64.RawStdEncoding.DecodeString},
		{name: "base64", decode: base64.StdEncoding.DecodeString},
		{name: "hex", decode: hex.DecodeString},
	}

	for _, decoder := range decoders {
		decoded, err := decoder.decode(trimmed)
		if err == nil && len(decoded) == 32 {
			return decoded, nil
		}
	}

	return nil, errors.New("ENCRYPTION_KEY must decode to exactly 32 bytes")
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
