package app

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"math/big"
	"strings"
)

const base62Alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

func randomToken(bytes int) (string, error) {
	buf := make([]byte, bytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func randomBase62(length int) (string, error) {
	var b strings.Builder
	b.Grow(length)
	max := big.NewInt(int64(len(base62Alphabet)))
	for b.Len() < length {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		b.WriteByte(base62Alphabet[n.Int64()])
	}
	return b.String(), nil
}

func sha256Bytes(data []byte) []byte {
	sum := sha256.Sum256(data)
	return sum[:]
}

func hmacHex(secret []byte, parts ...string) string {
	mac := hmac.New(sha256.New, secret)
	for _, part := range parts {
		mac.Write([]byte(part))
		mac.Write([]byte{0})
	}
	return hex.EncodeToString(mac.Sum(nil))
}

func constantEqual(a, b string) bool {
	return hmac.Equal([]byte(a), []byte(b))
}

func validatePublicID(id string) bool {
	if len(id) != 32 {
		return false
	}
	for _, r := range id {
		if !strings.ContainsRune(base62Alphabet, r) {
			return false
		}
	}
	return true
}

func decodeOwnerToken(token string) error {
	if len(token) < 32 || len(token) > 128 {
		return errors.New("invalid owner token")
	}
	if strings.ContainsAny(token, " \t\r\n") {
		return errors.New("invalid owner token")
	}
	return nil
}
