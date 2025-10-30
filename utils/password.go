package utils

import (
    "crypto/rand"
    "math/big"
)

const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()-_=+[]{}"

func GeneratePassword(n int) (string, error) {
    b := make([]byte, n)
    for i := 0; i < n; i++ {
        idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
        if err != nil { return "", err }
        b[i] = alphabet[idx.Int64()]
    }
    return string(b), nil
}
