package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"

	"golang.org/x/crypto/scrypt"
)

// scrypt constants
const (
	scryptN     = 32768
	scryptR     = 8
	scryptP     = 1
	scryptKeyLn = 32 // 32 bytes for AES-256
	saltBytes   = 16
)

// DeriveKey uses scrypt to derive a 32-byte key.
// Salt is randomly generated if not provided.
func DeriveKey(passphrase string, salt []byte) ([]byte, []byte, error) {
	if salt == nil {
		salt = make([]byte, saltBytes)
		if _, err := io.ReadFull(rand.Reader, salt); err != nil {
			return nil, nil, err
		}
	}

	key, err := scrypt.Key([]byte(passphrase), salt, scryptN, scryptR, scryptP, scryptKeyLn)
	if err != nil {
		return nil, nil, err
	}

	return key, salt, nil
}

// EncryptString encrypts plaintext using AES-GCM with a scrypt-derived key.
// The salt is prepended to the ciphertext.
func EncryptString(plaintext, passphrase string) (string, error) {
	key, salt, err := DeriveKey(passphrase, nil)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// Encrypt the data
	ciphertext := aesGCM.Seal(nil, nonce, []byte(plaintext), nil)

	// Prepend salt and nonce to the ciphertext
	// Format: [salt][nonce][ciphertext]
	finalPayload := append(salt, nonce...)
	finalPayload = append(finalPayload, ciphertext...)

	return base64.StdEncoding.EncodeToString(finalPayload), nil
}

// DecryptString decrypts ciphertext using AES-GCM with a scrypt-derived key.
// It expects the salt to be prepended to the ciphertext.
func DecryptString(ciphertextB64, passphrase string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(make([]byte, 32)) // dummy key
	if err != nil {
		return "", err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Calculate expected offsets
	nonceSize := aesGCM.NonceSize()
	if len(raw) < saltBytes+nonceSize {
		return "", errors.New("ciphertext too short")
	}

	// Extract salt, nonce, and ciphertext
	salt := raw[:saltBytes]
	nonce := raw[saltBytes : saltBytes+nonceSize]
	ct := raw[saltBytes+nonceSize:]

	// Derive the correct key using the extracted salt
	key, _, err := DeriveKey(passphrase, salt)
	if err != nil {
		return "", err
	}

	// Re-create cipher with the correct key
	block, err = aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	aesGCM, err = cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Decrypt
	pt, err := aesGCM.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", err // This error indicates decryption failure (e.g., bad key)
	}

	return string(pt), nil
}
