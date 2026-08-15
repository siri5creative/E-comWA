// Package crypto encrypts payment gateway credentials at rest (PRD NFR
// section 9: "Kredensial payment gateway disimpan terenkripsi, tidak
// pernah diekspos ke frontend"). AES-256-GCM: authenticated encryption,
// standard library only, appropriate for encrypting a small credential
// blob with a server-held key.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

type Box struct {
	gcm cipher.AEAD
}

// NewBox builds a Box from a base64-encoded 32-byte (AES-256) key. Returns
// (nil, nil) if keyBase64 is empty — payment gateway settings are an
// enhancement (the feature is explicitly "disiapkan, belum aktif" per PRD
// section 6.8), not required to run the app; callers should treat a nil
// Box as "encryption not configured" rather than fail startup.
func NewBox(keyBase64 string) (*Box, error) {
	if keyBase64 == "" {
		return nil, nil
	}

	key, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil {
		return nil, fmt.Errorf("decode PAYMENT_GATEWAY_ENCRYPTION_KEY: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("PAYMENT_GATEWAY_ENCRYPTION_KEY must decode to 32 bytes (AES-256), got %d", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("build AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("build GCM: %w", err)
	}

	return &Box{gcm: gcm}, nil
}

// Encrypt returns a base64-encoded (nonce || ciphertext || tag).
func (b *Box) Encrypt(plaintext string) (string, error) {
	nonce := make([]byte, b.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}
	sealed := b.gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt reverses Encrypt.
func (b *Box) Decrypt(encoded string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("decode ciphertext: %w", err)
	}
	if len(data) < b.gcm.NonceSize() {
		return "", errors.New("ciphertext too short")
	}
	nonce, ciphertext := data[:b.gcm.NonceSize()], data[b.gcm.NonceSize():]
	plaintext, err := b.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}
	return string(plaintext), nil
}
