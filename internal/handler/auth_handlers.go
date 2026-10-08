package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/marcosarl1/Circuito-API/internal/service"
)

// refreshCookieName is read by browsers automatically, which is also what
// makes a future EventSource subscription work without query-string keys.
const refreshCookieName = "refresh_token"

// refreshCookieMaxAge matches service.RefreshTokenTTL in seconds.
const refreshCookieMaxAge = 30 * 24 * 3600

type userContextKey struct{}

// UserFromContext returns the authenticated user stored by RequireUser.
func UserFromContext(request *http.Request) (*service.User, bool) {
	user, ok := request.Context().Value(userContextKey{}).(*service.User)
	return user, ok && user != nil
}

// AuthConfig carries the secrets and clock the auth handlers need.
// Now is a func so tests control time without waiting for expiry.
type AuthConfig struct {
	UserStore UserStore
	JWTSecret string
	Now       func() time.Time
}

func (config AuthConfig) now() time.Time {
	if config.Now != nil {
		return config.Now()
	}
	return time.Now()
}

// Login authenticates email+password and issues a token pair. Unknown user
// and wrong password share one 401 message to avoid account enumeration.
func Login(auth AuthConfig) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if auth.JWTSecret == "" {
			writeError(writer, http.StatusInternalServerError, "Autenticação não configurada")
			return
		}
		var input service.LoginRequest
		if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
			writeError(writer, http.StatusBadRequest, "Corpo inválido")
			return
		}
		if err := input.Validate(); err != nil {
			writeError(writer, http.StatusBadRequest, "Credenciais inválidas")
			return
		}
		user, err := auth.UserStore.FindUserByUsername(request.Context(), strings.TrimSpace(input.Username))
		if err != nil || service.VerifyPassword(user.PasswordHash, input.Password) != nil {
			writeError(writer, http.StatusUnauthorized, "Credenciais inválidas")
			return
		}
		if user.Disabled {
			writeError(writer, http.StatusForbidden, "Conta desabilitada")
			return
		}
		tokens, err := issueTokens(request.Context(), auth, user)
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "Erro interno")
			return
		}
		setRefreshCookie(writer, tokens.RefreshToken)
		writeJSON(writer, http.StatusOK, tokens)
	}
}

// issueTokens creates a fresh access + refresh pair and persists the hash.
func issueTokens(requestContext context.Context, auth AuthConfig, user *service.User) (service.AuthTokens, error) {
	now := auth.now()
	access, err := service.IssueAccessToken(auth.JWTSecret, user.ID.Hex(), now)
	if err != nil {
		return service.AuthTokens{}, err
	}
	rawRefresh, hash, err := service.NewRefreshToken()
	if err != nil {
		return service.AuthTokens{}, err
	}
	err = auth.UserStore.StoreRefreshToken(requestContext, service.RefreshToken{
		Hash:      hash,
		UserID:    user.ID.Hex(),
		ExpiresAt: now.Add(service.RefreshTokenTTL),
		CreatedAt: now,
	})
	if err != nil {
		return service.AuthTokens{}, err
	}
	return service.AuthTokens{
		AccessToken:  access,
		RefreshToken: rawRefresh,
		ExpiresIn:    int64(service.AccessTokenTTL / time.Second),
		Username:     user.Username,
	}, nil
}

// Refresh rotates the session: the presented token is consumed atomically
// and a new pair issued. Reuse of an old token always fails closed.
func Refresh(auth AuthConfig) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if auth.JWTSecret == "" {
			writeError(writer, http.StatusInternalServerError, "Autenticação não configurada")
			return
		}
		raw := refreshTokenFrom(request)
		var input service.RefreshRequest
		if raw != "" {
			input = service.RefreshRequest{RefreshToken: raw}
		} else if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
			writeError(writer, http.StatusBadRequest, "Corpo inválido")
			return
		}
		if err := input.Validate(); err != nil {
			writeError(writer, http.StatusUnauthorized, "Sessão inválida")
			return
		}
		token, err := auth.UserStore.TakeRefreshToken(request.Context(), service.HashRefreshToken(input.RefreshToken))
		if err != nil {
			writeError(writer, http.StatusUnauthorized, "Sessão inválida")
			return
		}
		user, err := auth.UserStore.FindUserByID(request.Context(), token.UserID)
		if err != nil || user.Disabled {
			writeError(writer, http.StatusUnauthorized, "Sessão inválida")
			return
		}
		tokens, err := issueTokens(request.Context(), auth, user)
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "Erro interno")
			return
		}
		setRefreshCookie(writer, tokens.RefreshToken)
		writeJSON(writer, http.StatusOK, tokens)
	}
}

// refreshTokenFrom prefers the HttpOnly cookie, falling back to the JSON
// body for non-browser clients.
func refreshTokenFrom(request *http.Request) string {
	if cookie, err := request.Cookie(refreshCookieName); err == nil {
		return cookie.Value
	}
	return ""
}

// Logout revokes the session idempotently: always 204, even without token.
func Logout(auth AuthConfig) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		raw := refreshTokenFrom(request)
		if raw == "" {
			var input service.RefreshRequest
			if err := json.NewDecoder(request.Body).Decode(&input); err == nil {
				raw = input.RefreshToken
			}
		}
		if raw != "" {
			_ = auth.UserStore.RevokeRefreshToken(request.Context(), service.HashRefreshToken(raw))
		}
		clearRefreshCookie(writer)
		writer.WriteHeader(http.StatusNoContent)
	}
}

// Me returns the authenticated account in its public shape.
func Me() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		user, ok := UserFromContext(request)
		if !ok {
			writeError(writer, http.StatusUnauthorized, "Não autenticado")
			return
		}
		writeJSON(writer, http.StatusOK, user.Public())
	}
}

// ChangePassword verifies the current credential before accepting the new one.
func ChangePassword(auth AuthConfig) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		user, ok := UserFromContext(request)
		if !ok {
			writeError(writer, http.StatusUnauthorized, "Não autenticado")
			return
		}
		var input service.ChangePasswordRequest
		if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
			writeError(writer, http.StatusBadRequest, "Corpo inválido")
			return
		}
		if err := service.ValidateNewPassword(input.NewPassword); err != nil {
			writeError(writer, http.StatusBadRequest, err.Error())
			return
		}
		stored, err := auth.UserStore.FindUserByID(request.Context(), user.ID.Hex())
		if err != nil || service.VerifyPassword(stored.PasswordHash, input.CurrentPassword) != nil {
			writeError(writer, http.StatusUnauthorized, "Credenciais inválidas")
			return
		}
		hash, err := service.HashPassword(input.NewPassword)
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "Erro interno")
			return
		}
		if err := auth.UserStore.UpdateUserPassword(request.Context(), user.ID.Hex(), hash); err != nil {
			writeError(writer, http.StatusInternalServerError, "Erro interno")
			return
		}
		writer.WriteHeader(http.StatusNoContent)
	}
}

// RequireUser enforces Bearer JWT authentication, loading the account so
// handlers and disabled checks share one lookup.
func RequireUser(auth AuthConfig) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(writer http.ResponseWriter, request *http.Request) {
			if auth.JWTSecret == "" {
				writeError(writer, http.StatusInternalServerError, "Autenticação não configurada")
				return
			}
			const prefix = "Bearer "
			header := request.Header.Get("Authorization")
			if !strings.HasPrefix(header, prefix) {
				writeError(writer, http.StatusUnauthorized, "Não autenticado")
				return
			}
			userID, err := service.ParseAccessToken(auth.JWTSecret, strings.TrimPrefix(header, prefix), auth.now())
			if err != nil {
				writeError(writer, http.StatusUnauthorized, "Não autenticado")
				return
			}
			user, err := auth.UserStore.FindUserByID(request.Context(), userID)
			if err != nil {
				writeError(writer, http.StatusUnauthorized, "Não autenticado")
				return
			}
			if user.Disabled {
				writeError(writer, http.StatusForbidden, "Conta desabilitada")
				return
			}
			next(writer, request.WithContext(withUser(request, user)))
		}
	}
}

// RequireAdmin enforces an authenticated ADMIN user. All write routes
// are admin operations today; RequireUser stays for self-service routes
// (/me, password change) that any valid account may call.
func RequireAdmin(auth AuthConfig) func(http.HandlerFunc) http.HandlerFunc {
	requireUser := RequireUser(auth)
	return func(next http.HandlerFunc) http.HandlerFunc {
		return requireUser(func(writer http.ResponseWriter, request *http.Request) {
			user, ok := UserFromContext(request)
			if !ok || user.Role != service.RoleAdmin {
				writeError(writer, http.StatusForbidden, "Acesso restrito")
				return
			}
			next(writer, request)
		})
	}
}

// RequireUserOrKey accepts an ADMIN JWT or, during the transition from
// hardcoded keys, the service X-API-Key. New integrations must use JWT;
// the key path exists only for existing service callers.
func RequireUserOrKey(auth AuthConfig, apiKey string) func(http.HandlerFunc) http.HandlerFunc {
	requireKey := RequireAPIKey(apiKey)
	requireAdmin := RequireAdmin(auth)
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(writer http.ResponseWriter, request *http.Request) {
			if strings.HasPrefix(request.Header.Get("Authorization"), "Bearer ") {
				requireAdmin(next)(writer, request)
				return
			}
			requireKey(next)(writer, request)
		}
	}
}

func withUser(request *http.Request, user *service.User) context.Context {
	return context.WithValue(request.Context(), userContextKey{}, user)
}

const refreshCookiePath = "/api"

func setRefreshCookie(writer http.ResponseWriter, raw string) {
	http.SetCookie(writer, &http.Cookie{
		Name:     refreshCookieName,
		Value:    raw,
		Path:     refreshCookiePath,
		MaxAge:   refreshCookieMaxAge,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
}

func clearRefreshCookie(writer http.ResponseWriter) {
	http.SetCookie(writer, &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     refreshCookiePath,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
}
