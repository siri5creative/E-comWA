package crypto

import (
	"encoding/base64"
	"strings"
	"testing"
)

func testKey() string {
	return base64.StdEncoding.EncodeToString(make([]byte, 32))
}

func TestNewBoxEmptyKeyReturnsNil(t *testing.T) {
	box, err := NewBox("")
	if err != nil {
		t.Fatalf("NewBox(\"\") returned error: %v", err)
	}
	if box != nil {
		t.Fatalf("NewBox(\"\") = %v; want nil (encryption disabled)", box)
	}
}

func TestNewBoxInvalidBase64(t *testing.T) {
	if _, err := NewBox("not-valid-base64!!!"); err == nil {
		t.Fatal("NewBox with invalid base64 should return an error")
	}
}

func TestNewBoxWrongKeyLength(t *testing.T) {
	shortKey := base64.StdEncoding.EncodeToString([]byte("too-short"))
	if _, err := NewBox(shortKey); err == nil {
		t.Fatal("NewBox with a non-32-byte key should return an error")
	}
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	box, err := NewBox(testKey())
	if err != nil {
		t.Fatalf("NewBox: %v", err)
	}

	plaintext := `{"server_key":"SB-Mid-server-abc123","client_key":"SB-Mid-client-xyz789"}`
	ciphertext, err := box.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if strings.Contains(ciphertext, "server_key") || strings.Contains(ciphertext, "abc123") {
		t.Fatalf("ciphertext appears to contain plaintext: %q", ciphertext)
	}

	decrypted, err := box.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if decrypted != plaintext {
		t.Fatalf("Decrypt(Encrypt(x)) = %q; want %q", decrypted, plaintext)
	}
}

func TestEncryptProducesDifferentCiphertextEachTime(t *testing.T) {
	box, err := NewBox(testKey())
	if err != nil {
		t.Fatalf("NewBox: %v", err)
	}

	a, err := box.Encrypt("same plaintext")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	b, err := box.Encrypt("same plaintext")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if a == b {
		t.Fatal("Encrypt of the same plaintext twice produced identical ciphertext — nonce reuse")
	}
}

func TestDecryptRejectsTamperedCiphertext(t *testing.T) {
	box, err := NewBox(testKey())
	if err != nil {
		t.Fatalf("NewBox: %v", err)
	}

	ciphertext, err := box.Encrypt("secret credentials")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	raw, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		t.Fatalf("decode ciphertext: %v", err)
	}
	raw[len(raw)-1] ^= 0xFF // flip a bit in the auth tag
	tampered := base64.StdEncoding.EncodeToString(raw)

	if _, err := box.Decrypt(tampered); err == nil {
		t.Fatal("Decrypt accepted a tampered ciphertext without error")
	}
}

func TestDecryptWithWrongKeyFails(t *testing.T) {
	boxA, err := NewBox(testKey())
	if err != nil {
		t.Fatalf("NewBox: %v", err)
	}
	otherKey := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("1", 32)))
	boxB, err := NewBox(otherKey)
	if err != nil {
		t.Fatalf("NewBox: %v", err)
	}

	ciphertext, err := boxA.Encrypt("secret")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if _, err := boxB.Decrypt(ciphertext); err == nil {
		t.Fatal("Decrypt succeeded with the wrong key")
	}
}
