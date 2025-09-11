package utils

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "crypto/sha256"
    "encoding/base64"
    "errors"
    "io"
)

// DeriveKey derives a 32-byte key from passphrase using SHA-256 (for demo; replace with KDF like scrypt in prod).
func DeriveKey(passphrase string) []byte {
    h := sha256.Sum256([]byte(passphrase))
    return h[:]
}

func EncryptString(plaintext, passphrase string) (string, error) {
    key := DeriveKey(passphrase)
    block, err := aes.NewCipher(key)
    if err != nil { return "", err }
    aesGCM, err := cipher.NewGCM(block)
    if err != nil { return "", err }
    nonce := make([]byte, aesGCM.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil { return "", err }
    ct := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)
    return base64.StdEncoding.EncodeToString(ct), nil
}

func DecryptString(ciphertextB64, passphrase string) (string, error) {
    key := DeriveKey(passphrase)
    raw, err := base64.StdEncoding.DecodeString(ciphertextB64)
    if err != nil { return "", err }
    block, err := aes.NewCipher(key)
    if err != nil { return "", err }
    aesGCM, err := cipher.NewGCM(block)
    if err != nil { return "", err }
    if len(raw) < aesGCM.NonceSize() { return "", errors.New("ciphertext too short") }
    nonce, ct := raw[:aesGCM.NonceSize()], raw[aesGCM.NonceSize():]
    pt, err := aesGCM.Open(nil, nonce, ct, nil)
    if err != nil { return "", err }
    return string(pt), nil
}
