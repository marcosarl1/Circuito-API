package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/marcosarl1/Circuito-API/internal/service"
	"go.mongodb.org/mongo-driver/v2/bson"
)

var testClock = time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)

func testAuthConfig(store *fakeUserStore) AuthConfig {
	return AuthConfig{UserStore: store, JWTSecret: "test-jwt-secret-32-chars-minimum", Now: func() time.Time { return testClock }}
}

func seedTestUser(t *testing.T, store *fakeUserStore) *service.User {
	t.Helper()
	hash, err := service.HashPassword("senha-correta-123")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	user, err := store.CreateUser(t.Context(), service.User{
		ID:        bson.NewObjectID(),
		Username:  "admincircuito",
		Role:      service.RoleAdmin,
		CreatedAt: testClock,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	user.PasswordHash = hash
	return user
}

func loginTokens(t *testing.T, auth AuthConfig, username, password string) service.AuthTokens {
	t.Helper()
	request := newTestRequest(http.MethodPost, "/api/v1/auth/login",
		`{"username":"`+username+`","password":"`+password+`"}`)
	recorder := httptest.NewRecorder()
	Login(auth)(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("login: expected 200, got %d; body=%s", recorder.Code, recorder.Body.String())
	}
	var tokens service.AuthTokens
	if err := json.NewDecoder(recorder.Body).Decode(&tokens); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if tokens.AccessToken == "" || tokens.RefreshToken == "" || tokens.ExpiresIn != 900 {
		t.Fatalf("tokens incompletos: %+v", tokens)
	}
	return tokens
}

func TestLoginIssuesTokensAndCookie(t *testing.T) {
	store := newFakeUserStore()
	seedTestUser(t, store)
	auth := testAuthConfig(store)

	request := newTestRequest(http.MethodPost, "/api/v1/auth/login",
		`{"username":"admincircuito","password":"senha-correta-123"}`)
	recorder := httptest.NewRecorder()
	Login(auth)(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body=%s", recorder.Code, recorder.Body.String())
	}
	cookie := recorder.Header().Get("Set-Cookie")
	for _, part := range []string{"refresh_token=", "HttpOnly", "Path=/api"} {
		if !strings.Contains(cookie, part) {
			t.Fatalf("cookie sem %q: %q", part, cookie)
		}
	}
}

func TestLoginRejectsWithoutOracle(t *testing.T) {
	store := newFakeUserStore()
	seedTestUser(t, store)
	auth := testAuthConfig(store)

	for name, body := range map[string]string{
		"unknown user": `{"username":"ninguem","password":"senha-correta-123"}`,
		"wrong pass":   `{"username":"admincircuito","password":"errada"}`,
		"bad json":     `{"username":"admincircuito"`,
	} {
		request := newTestRequest(http.MethodPost, "/api/v1/auth/login", body)
		recorder := httptest.NewRecorder()
		Login(auth)(recorder, request)
		if recorder.Code != http.StatusUnauthorized && recorder.Code != http.StatusBadRequest {
			t.Fatalf("%s: esperado 401/400, obtido %d", name, recorder.Code)
		}
	}

	disabled, _ := store.FindUserByUsername(t.Context(), "admincircuito")
	disabled.Disabled = true
	request := newTestRequest(http.MethodPost, "/api/v1/auth/login",
		`{"username":"admincircuito","password":"senha-correta-123"}`)
	recorder := httptest.NewRecorder()
	Login(auth)(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("disabled: esperado 403, obtido %d", recorder.Code)
	}
}

func TestRefreshRotatesAndRejectsReuse(t *testing.T) {
	store := newFakeUserStore()
	seedTestUser(t, store)
	auth := testAuthConfig(store)
	tokens := loginTokens(t, auth, "admincircuito", "senha-correta-123")

	second := newTestRequest(http.MethodPost, "/api/v1/auth/refresh",
		`{"refresh_token":"`+tokens.RefreshToken+`"}`)
	secondRecorder := httptest.NewRecorder()
	Refresh(auth)(secondRecorder, second)
	if secondRecorder.Code != http.StatusOK {
		t.Fatalf("refresh: esperado 200, obtido %d", secondRecorder.Code)
	}

	reuse := newTestRequest(http.MethodPost, "/api/v1/auth/refresh",
		`{"refresh_token":"`+tokens.RefreshToken+`"}`)
	reuseRecorder := httptest.NewRecorder()
	Refresh(auth)(reuseRecorder, reuse)
	if reuseRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("reuse: esperado 401, obtido %d", reuseRecorder.Code)
	}

	garbage := newTestRequest(http.MethodPost, "/api/v1/auth/refresh", `{"refresh_token":"lixo"}`)
	garbageRecorder := httptest.NewRecorder()
	Refresh(auth)(garbageRecorder, garbage)
	if garbageRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("garbage: esperado 401, obtido %d", garbageRecorder.Code)
	}
}

func TestLogoutRevokesSession(t *testing.T) {
	store := newFakeUserStore()
	seedTestUser(t, store)
	auth := testAuthConfig(store)
	tokens := loginTokens(t, auth, "admincircuito", "senha-correta-123")

	request := newTestRequest(http.MethodPost, "/api/v1/auth/logout",
		`{"refresh_token":"`+tokens.RefreshToken+`"}`)
	recorder := httptest.NewRecorder()
	Logout(auth)(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("logout: esperado 204, obtido %d", recorder.Code)
	}
	if cookie := recorder.Header().Get("Set-Cookie"); !strings.Contains(cookie, "Max-Age=0") {
		t.Fatalf("cookie não limpo: %q", cookie)
	}

	after := newTestRequest(http.MethodPost, "/api/v1/auth/refresh",
		`{"refresh_token":"`+tokens.RefreshToken+`"}`)
	afterRecorder := httptest.NewRecorder()
	Refresh(auth)(afterRecorder, after)
	if afterRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("pós-logout: esperado 401, obtido %d", afterRecorder.Code)
	}
}

func TestMeRequiresBearer(t *testing.T) {
	store := newFakeUserStore()
	seedTestUser(t, store)
	auth := testAuthConfig(store)
	tokens := loginTokens(t, auth, "admincircuito", "senha-correta-123")

	anonymous := newTestRequest(http.MethodGet, "/api/v1/auth/me", "")
	anonymousRecorder := httptest.NewRecorder()
	RequireUser(auth)(Me())(anonymousRecorder, anonymous)
	if anonymousRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("sem token: esperado 401, obtido %d", anonymousRecorder.Code)
	}

	request := newTestRequest(http.MethodGet, "/api/v1/auth/me", "")
	request.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	recorder := httptest.NewRecorder()
	RequireUser(auth)(Me())(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("com token: esperado 200, obtido %d; body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestUserOrKeyAcceptsBoth(t *testing.T) {
	store := newFakeUserStore()
	seedTestUser(t, store)
	auth := testAuthConfig(store)
	tokens := loginTokens(t, auth, "admincircuito", "senha-correta-123")
	guard := RequireUserOrKey(auth, "service-key")

	next := func(writer http.ResponseWriter, _ *http.Request) { writer.WriteHeader(http.StatusNoContent) }

	bearer := newTestRequest(http.MethodPost, "/api/v1/eventos", "")
	bearer.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	bearerRecorder := httptest.NewRecorder()
	guard(next)(bearerRecorder, bearer)
	if bearerRecorder.Code != http.StatusNoContent {
		t.Fatalf("bearer: esperado 204, obtido %d", bearerRecorder.Code)
	}

	keyed := newTestRequest(http.MethodPost, "/api/v1/eventos", "")
	keyed.Header.Set("X-API-Key", "service-key")
	keyedRecorder := httptest.NewRecorder()
	guard(next)(keyedRecorder, keyed)
	if keyedRecorder.Code != http.StatusNoContent {
		t.Fatalf("key: esperado 204, obtido %d", keyedRecorder.Code)
	}

	naked := newTestRequest(http.MethodPost, "/api/v1/eventos", "")
	nakedRecorder := httptest.NewRecorder()
	guard(next)(nakedRecorder, naked)
	if nakedRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("sem credencial: esperado 401, obtido %d", nakedRecorder.Code)
	}
}

func TestUserOrKeyRejectsNonAdmin(t *testing.T) {
	store := newFakeUserStore()
	hash, err := service.HashPassword("senha-leitura-123")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	created, err := store.CreateUser(t.Context(), service.User{
		ID: bson.NewObjectID(), Username: "leitor", Role: "READER", CreatedAt: testClock,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	reader, _ := store.FindUserByID(t.Context(), created.ID.Hex())
	reader.PasswordHash = hash
	auth := testAuthConfig(store)
	tokens := loginTokens(t, auth, "leitor", "senha-leitura-123")

	guard := RequireUserOrKey(auth, "service-key")
	next := func(writer http.ResponseWriter, _ *http.Request) { writer.WriteHeader(http.StatusNoContent) }

	request := newTestRequest(http.MethodPost, "/api/v1/eventos", "")
	request.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	recorder := httptest.NewRecorder()
	guard(next)(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("non-admin: esperado 403, obtido %d", recorder.Code)
	}
}

func TestChangePassword(t *testing.T) {
	store := newFakeUserStore()
	seedTestUser(t, store)
	auth := testAuthConfig(store)
	tokens := loginTokens(t, auth, "admincircuito", "senha-correta-123")

	change := newTestRequest(http.MethodPatch, "/api/v1/auth/password",
		`{"current_password":"senha-correta-123","new_password":"nova-senha-456"}`)
	change.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	changeRecorder := httptest.NewRecorder()
	RequireUser(auth)(ChangePassword(auth))(changeRecorder, change)
	if changeRecorder.Code != http.StatusNoContent {
		t.Fatalf("troca: esperado 204, obtido %d; body=%s", changeRecorder.Code, changeRecorder.Body.String())
	}

	old := newTestRequest(http.MethodPost, "/api/v1/auth/login",
		`{"username":"admincircuito","password":"senha-correta-123"}`)
	oldRecorder := httptest.NewRecorder()
	Login(auth)(oldRecorder, old)
	if oldRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("senha antiga deveria falhar, obtido %d", oldRecorder.Code)
	}

	loginTokens(t, auth, "admincircuito", "nova-senha-456")

	weak := newTestRequest(http.MethodPatch, "/api/v1/auth/password",
		`{"current_password":"nova-senha-456","new_password":"curta"}`)
	weak.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	weakRecorder := httptest.NewRecorder()
	RequireUser(auth)(ChangePassword(auth))(weakRecorder, weak)
	if weakRecorder.Code != http.StatusBadRequest {
		t.Fatalf("senha fraca: esperado 400, obtido %d", weakRecorder.Code)
	}
}
