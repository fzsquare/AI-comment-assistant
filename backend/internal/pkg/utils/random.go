package utils

import (
	"crypto/rand"
	"encoding/hex"
)

func RandomString(length int) string {
	if length <= 0 {
		return ""
	}
	bytes := make([]byte, (length+1)/2)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)[:length]
}
