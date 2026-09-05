package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"time"
)

// TouchInterval — как часто активность сессии записывается в БД.
const TouchInterval = time.Hour

const tokenBytes = 32

// GenerateToken возвращает bearer-токен для клиента и его хеш для БД.
func GenerateToken() (string, string) {
	buf := make([]byte, tokenBytes)
	// crypto/rand.Read не возвращает ошибку начиная с Go 1.24.
	_, _ = rand.Read(buf)
	plain := base64.RawURLEncoding.EncodeToString(buf)
	return plain, HashToken(plain)
}

// HashToken — hex(sha256(token)), в таком виде токен лежит в sessions.token_hash.
func HashToken(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}
