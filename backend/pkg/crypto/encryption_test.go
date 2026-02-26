package crypto

import (
	"bytes"
	"testing"
)

func TestEncryptDecryptRoundtrip(t *testing.T) {
	key := DeriveKey("test-master-key")
	plaintext := []byte("my-secret-api-key-12345")

	ciphertext, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Ciphertext must differ from plaintext
	if bytes.Equal(ciphertext, plaintext) {
		t.Fatal("ciphertext should differ from plaintext")
	}

	decrypted, err := Decrypt(ciphertext, key)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("decrypted text does not match original: got %q, want %q", decrypted, plaintext)
	}
}

func TestDecryptWithWrongKey(t *testing.T) {
	key1 := DeriveKey("correct-key")
	key2 := DeriveKey("wrong-key")
	plaintext := []byte("secret-data")

	ciphertext, err := Encrypt(plaintext, key1)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	_, err = Decrypt(ciphertext, key2)
	if err == nil {
		t.Fatal("Decrypt with wrong key should fail")
	}
}

func TestDecryptCorruptedCiphertext(t *testing.T) {
	key := DeriveKey("test-key")
	plaintext := []byte("secret-data")

	ciphertext, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Corrupt a byte in the middle of the ciphertext
	if len(ciphertext) > 15 {
		ciphertext[15] ^= 0xFF
	}

	_, err = Decrypt(ciphertext, key)
	if err == nil {
		t.Fatal("Decrypt with corrupted ciphertext should fail")
	}
}

func TestEncryptDecryptEmptyPlaintext(t *testing.T) {
	key := DeriveKey("test-key")
	plaintext := []byte{}

	ciphertext, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt empty plaintext failed: %v", err)
	}

	decrypted, err := Decrypt(ciphertext, key)
	if err != nil {
		t.Fatalf("Decrypt empty plaintext failed: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("decrypted empty text does not match: got %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptInvalidKeySize(t *testing.T) {
	shortKey := []byte("too-short")
	_, err := Encrypt([]byte("data"), shortKey)
	if err == nil {
		t.Fatal("Encrypt with short key should fail")
	}
}

func TestDecryptInvalidKeySize(t *testing.T) {
	shortKey := []byte("too-short")
	_, err := Decrypt([]byte("some-ciphertext-data-here-longer"), shortKey)
	if err == nil {
		t.Fatal("Decrypt with short key should fail")
	}
}

func TestDecryptTooShortCiphertext(t *testing.T) {
	key := DeriveKey("test-key")
	// GCM nonce is 12 bytes; ciphertext shorter than that should fail
	_, err := Decrypt([]byte("short"), key)
	if err == nil {
		t.Fatal("Decrypt with too-short ciphertext should fail")
	}
}

func TestDeriveKeyDeterministic(t *testing.T) {
	key1 := DeriveKey("same-input")
	key2 := DeriveKey("same-input")
	if !bytes.Equal(key1, key2) {
		t.Fatal("DeriveKey should be deterministic")
	}
}

func TestDeriveKeyLength(t *testing.T) {
	key := DeriveKey("any-string")
	if len(key) != 32 {
		t.Fatalf("DeriveKey should return 32 bytes, got %d", len(key))
	}
}

func TestDeriveKeyDifferentInputs(t *testing.T) {
	key1 := DeriveKey("input-1")
	key2 := DeriveKey("input-2")
	if bytes.Equal(key1, key2) {
		t.Fatal("DeriveKey with different inputs should produce different keys")
	}
}
