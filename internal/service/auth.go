package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"golang.org/x/crypto/bcrypt"
)

// AccessTokenTTL is deliberately short: leaked tokens expire fast,
// renewal goes through the rotating refresh token.
const AccessTokenTTL = 15 * time.Minute

// RefreshTokenTTL bounds the session lifetime without re-login.
const RefreshTokenTTL = 30 * 24 * time.Hour

// RoleAdmin is the only role today: full write access. The field exists
// so future read-only roles need no migration.
const RoleAdmin = "ADMIN"

// User is an admin account. The password never leaves the database hashed.
// ID is a native ObjectID (what MongoDB generates); on the wire it is
// always the hex string, never an object.
type User struct {
	ID           bson.ObjectID `bson:"_id" json:"-"`
	Username     string        `bson:"username" json:"username"`
	PasswordHash string        `bson:"password_hash" json:"-"`
	Role         string        `bson:"role" json:"role"`
	Disabled     bool          `bson:"disabled" json:"-"`
	CreatedAt    time.Time     `bson:"created_at" json:"created_at"`
}

// RefreshToken is stored hashed: a database leak alone never yields
// a usable token. The raw value exists only in transit to the client.
type RefreshToken struct {
	Hash      string    `bson:"_id"`
	UserID    string    `bson:"user_id"`
	ExpiresAt time.Time `bson:"expires_at"`
	CreatedAt time.Time `bson:"created_at"`
}

// HashPassword hashes with bcrypt (cost 12: slow for attackers, fast enough
// for a login endpoint behind rate limiting).
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// VerifyPassword compares in constant time; mismatch returns an error,
// never a boolean the caller could ignore.
func VerifyPassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// IssueAccessToken signs a short-lived JWT with the user id as subject.
func IssueAccessToken(secret, userID string, now time.Time) (string, error) {
	if secret == "" {
		return "", fmt.Errorf("jwt secret not configured")
	}
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(AccessTokenTTL)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ParseAccessToken verifies signature, method and expiry, returning the
// subject (user id). Any failure is a single error: callers answer 401.
func ParseAccessToken(secret, raw string, now time.Time) (string, error) {
	if secret == "" {
		return "", fmt.Errorf("jwt secret not configured")
	}
	token, err := jwt.ParseWithClaims(raw, &jwt.RegisteredClaims{}, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(secret), nil
	}, jwt.WithTimeFunc(func() time.Time { return now }))
	if err != nil {
		return "", err
	}
	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || !token.Valid || claims.Subject == "" {
		return "", fmt.Errorf("invalid token")
	}
	return claims.Subject, nil
}

// NewRefreshToken generates a raw token for the client and its hash for
// storage. Only the hash is ever persisted.
func NewRefreshToken() (raw string, hash string, err error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", err
	}
	raw = hex.EncodeToString(bytes)
	sum := sha256.Sum256([]byte(raw))
	return raw, hex.EncodeToString(sum[:]), nil
}

// HashRefreshToken maps a presented token to its stored hash.
func HashRefreshToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
