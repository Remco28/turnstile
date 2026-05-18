package token

import (
	"crypto/rand"
	"encoding/base64"
)

const prefix = "tsk_live_"

func New() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return prefix + base64.RawURLEncoding.EncodeToString(buf), nil
}

func Prefix(value string) string {
	if len(value) <= 12 {
		return value
	}
	return value[:12]
}
