package security

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncryptDecryptStringRoundTrip(t *testing.T) {
	_, err := ConfigureFieldEncryption(EncryptionConfig{JWTSecret: "test-jwt-secret"})
	require.NoError(t, err)

	encrypted, err := EncryptString("refresh-token")
	require.NoError(t, err)
	require.True(t, IsEncryptedString(encrypted))
	require.NotEqual(t, "refresh-token", encrypted)

	plaintext, legacy, err := DecryptString(encrypted)
	require.NoError(t, err)
	require.False(t, legacy)
	require.Equal(t, "refresh-token", plaintext)
}

func TestEncryptStringUsesRandomNonce(t *testing.T) {
	_, err := ConfigureFieldEncryption(EncryptionConfig{JWTSecret: "test-jwt-secret"})
	require.NoError(t, err)

	first, err := EncryptString("same-secret")
	require.NoError(t, err)
	second, err := EncryptString("same-secret")
	require.NoError(t, err)

	require.NotEqual(t, first, second)
}

func TestDecryptStringSupportsLegacyPlaintext(t *testing.T) {
	_, err := ConfigureFieldEncryption(EncryptionConfig{JWTSecret: "test-jwt-secret"})
	require.NoError(t, err)

	plaintext, legacy, err := DecryptString("legacy-secret")
	require.NoError(t, err)
	require.True(t, legacy)
	require.Equal(t, "legacy-secret", plaintext)
}

func TestEncryptionKeyOverridesJWTSecret(t *testing.T) {
	key := base64.StdEncoding.EncodeToString([]byte("12345678901234567890123456789012"))

	status, err := ConfigureFieldEncryption(EncryptionConfig{
		EncryptionKey: key,
		JWTSecret:     "test-jwt-secret",
	})
	require.NoError(t, err)
	require.Equal(t, "encryption_key", status.Source)
}

func TestJWTSecretDerivationIsStableAndNotRawSecret(t *testing.T) {
	status, err := ConfigureFieldEncryption(EncryptionConfig{JWTSecret: "stable-jwt-secret"})
	require.NoError(t, err)
	require.Equal(t, "jwt_derived", status.Source)

	encrypted, err := EncryptString("secret")
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(encrypted, EncryptedValuePrefix))
	require.NotContains(t, encrypted, "stable-jwt-secret")
}

func TestInvalidEncryptionKeyFails(t *testing.T) {
	_, err := ConfigureFieldEncryption(EncryptionConfig{EncryptionKey: "too-short"})
	require.Error(t, err)
}

func TestMissingJWTSecretFailsWithoutEncryptionKey(t *testing.T) {
	_, err := ConfigureFieldEncryption(EncryptionConfig{})
	require.Error(t, err)
}
