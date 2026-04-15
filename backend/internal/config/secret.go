package config

import (
	"crypto/rand"
	"encoding/base64"
)

func randomSecret(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "pichub-dev-secret-change-me"
	}
	return base64.URLEncoding.EncodeToString(buf)
}
