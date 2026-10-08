package service

import (
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// AuthTokens is returned by both login and refresh: same shape, so the
// frontend handles both with one code path.
type AuthTokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	Username     string `json:"username"`
}

// LoginRequest carries the only human credential this API accepts:
// username + password. No email, no OAuth, no magic links.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Validate rejects malformed input before any database or bcrypt work,
// so attackers cannot use the endpoint as a timing oracle for hashing.
func (request LoginRequest) Validate() error {
	if !validUsername(request.Username) {
		return fmt.Errorf("invalid username")
	}
	if request.Password == "" {
		return fmt.Errorf("password is required")
	}
	return nil
}

// RefreshRequest carries the opaque token issued at login.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Validate rejects garbage without touching the database: the token must
// look like the 64 hex characters we issue.
func (request RefreshRequest) Validate() error {
	trimmed := strings.TrimSpace(request.RefreshToken)
	if len(trimmed) != 64 {
		return fmt.Errorf("invalid refresh token")
	}
	if _, err := hex.DecodeString(trimmed); err != nil {
		return fmt.Errorf("invalid refresh token")
	}
	return nil
}

// ChangePasswordRequest rotates the credential. Policy lives here so
// seed and endpoint share the exact same rule.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// ValidateNewPassword enforces the single password policy of the system.
func ValidateNewPassword(password string) error {
	if len([]rune(password)) < 8 {
		return fmt.Errorf("password must have at least 8 characters")
	}
	return nil
}

// PublicUser is the safe projection of User for responses: hash and flags
// can never leak because they do not exist on this type.
type PublicUser struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// Public projects a User to its response-safe shape.
func (user User) Public() PublicUser {
	return PublicUser{ID: user.ID, Username: user.Username, Role: user.Role, CreatedAt: user.CreatedAt}
}

func validUsername(raw string) bool {
	username := strings.TrimSpace(raw)
	if len(username) < 3 || len(username) > 64 {
		return false
	}
	return !strings.ContainsAny(username, " \t\n")
}
