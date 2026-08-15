package common

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
)

const encryptedSecretPrefix = "enc:v1:"

// EncryptSecret encrypts administrator-provided integration credentials before database persistence.
func EncryptSecret(plaintext string) (string, error) {
	if strings.TrimSpace(plaintext) == "" {
		return "", fmt.Errorf("secret cannot be empty")
	}
	key := sha256.Sum256([]byte(CryptoSecret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return encryptedSecretPrefix + base64.RawURLEncoding.EncodeToString(ciphertext), nil
}

// DecryptSecret returns a credential stored by EncryptSecret. The value is never safe to log.
func DecryptSecret(ciphertext string) (string, error) {
	if !strings.HasPrefix(ciphertext, encryptedSecretPrefix) {
		return "", fmt.Errorf("invalid encrypted secret format")
	}
	payload, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(ciphertext, encryptedSecretPrefix))
	if err != nil {
		return "", fmt.Errorf("decode encrypted secret: %w", err)
	}
	key := sha256.Sum256([]byte(CryptoSecret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(payload) < gcm.NonceSize() {
		return "", fmt.Errorf("invalid encrypted secret payload")
	}
	plaintext, err := gcm.Open(nil, payload[:gcm.NonceSize()], payload[gcm.NonceSize():], nil)
	if err != nil {
		return "", fmt.Errorf("decrypt secret: %w", err)
	}
	return string(plaintext), nil
}
