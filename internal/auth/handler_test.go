package auth_test

import (
	"CrudTutorialProject/internal/auth"
	"CrudTutorialProject/internal/validation"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type testAuthHTTPApp struct {
	handler http.Handler
	repo    auth.UserStore
	hasher  *auth.PasswordHasher
	tokens  *auth.TokenManager
}

func newTestAuthHTTPApp(t *testing.T) *testAuthHTTPApp {
	t.Helper()

	repo := newFakeUserRepository()
	hasher := auth.NewPasswordHasherWithCost(bcrypt.MinCost)
	tokens := auth.NewTokenManager(testJWTSecret, 15*time.Minute, "go-crud-api")

	service := auth.NewService(repo, hasher, tokens)

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	validator := validation.New()

	authHandler := auth.NewHandler(service, log, validator)
	authMiddleware := auth.NewMiddleWare(tokens)

	mux := http.NewServeMux()
	authHandler.RegisterRoutes(mux, authMiddleware)

	return &testAuthHTTPApp{
		handler: mux,
		repo:    repo,
		hasher:  hasher,
		tokens:  tokens,
	}
}

func executeAuthRequest(
	t *testing.T,
	handler http.Handler,
	method string,
	target string,
	body string,
	token string,
) *http.Response {
	t.Helper()

	req := httptest.NewRequest(method, target, strings.NewReader(body))

	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	return rr.Result()
}

func decodeAuthJSON[T any](t *testing.T, res *http.Response) T {
	t.Helper()
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(res.Body)

	var dst T
	if err := json.NewDecoder(res.Body).Decode(&dst); err != nil {
		t.Fatalf("failed to decode auth response: %s", err)
	}

	return dst
}

func TestHandler_Register_Success(t *testing.T) {
	app := newTestAuthHTTPApp(t)

	res := executeAuthRequest(
		t,
		app.handler,
		http.MethodPost,
		"/auth/register",
		`{"name":"Alex","email":"alex@example.com","age":25,"password":"password123"}`,
		"",
	)

	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected status code 201, got %v", res.StatusCode)
	}

	body := decodeAuthJSON[auth.AuthResponse](t, res)

	if body.AccessToken == "" {
		t.Fatal("expected access token to be set")
	}

	if body.TokenType != "Bearer" {
		t.Fatalf("expected token type %q, got %q", "Bearer", body.TokenType)
	}

	if body.User.ID == 0 {
		t.Fatal("expected user ID to be set")
	}

	if body.User.Email != "alex@example.com" {
		t.Fatalf("expected email %q, got %q", "alex@example.com", body.User.Email)
	}

	saved, err := app.repo.FindByEmail(t.Context(), "alex@example.com")

	if err != nil {
		t.Fatalf("failed to find saved user: %v", err)
	}

	if saved.ID != body.User.ID {
		t.Fatalf("expected saved user ID %d, got %d", body.User.ID, saved.ID)
	}

	if saved.PasswordHash == "" {
		t.Fatal("expected password hash to be set")
	}

	if saved.PasswordHash == "password123" {
		t.Fatal("password hash must not equal plain password")
	}

	if !app.hasher.Compare(saved.PasswordHash, "password123") {
		t.Fatal("expected saved hash to match password")
	}
}

func TestHandler_Register_ValidationError(t *testing.T) {
	app := newTestAuthHTTPApp(t)

	res := executeAuthRequest(
		t,
		app.handler,
		http.MethodPost,
		"/auth/register",
		`{"name":"A","email":"invalid-email","age":25,"password":"123"}`,
		"",
	)

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, res.StatusCode)
	}

	_ = res.Body.Close()
}

func TestHandler_Register_MalformedJSON(t *testing.T) {
	app := newTestAuthHTTPApp(t)

	res := executeAuthRequest(
		t,
		app.handler,
		http.MethodPost,
		"/auth/register",
		`{"name":"Alex","email":`,
		"",
	)

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, res.StatusCode)
	}

	_ = res.Body.Close()
}

func TestHandler_Register_DuplicateEmail(t *testing.T) {
	app := newTestAuthHTTPApp(t)

	first := executeAuthRequest(
		t,
		app.handler,
		http.MethodPost,
		"/auth/register",
		`{"name":"Alex","email":"alex@example.com","age":25,"password":"password123"}`,
		"",
	)

	if first.StatusCode != http.StatusCreated {
		t.Fatalf("expected first status %d, got %d", http.StatusCreated, first.StatusCode)
	}
	_ = first.Body.Close()

	second := executeAuthRequest(
		t,
		app.handler,
		http.MethodPost,
		"/auth/register",
		`{"name":"Another Alex","email":"Alex@Example.com","age":30,"password":"password456"}`,
		"",
	)

	if second.StatusCode != http.StatusConflict {
		t.Fatalf("expected second status %d, got %d", http.StatusConflict, second.StatusCode)
	}

	_ = second.Body.Close()
}

func TestHandler_Login_Success(t *testing.T) {
	app := newTestAuthHTTPApp(t)

	register := executeAuthRequest(
		t,
		app.handler,
		http.MethodPost,
		"/auth/register",
		`{"name":"Alex","email":"alex@example.com","age":25,"password":"password123"}`,
		"",
	)

	if register.StatusCode != http.StatusCreated {
		t.Fatalf("expected register status %d, got %d", http.StatusCreated, register.StatusCode)
	}
	_ = register.Body.Close()

	login := executeAuthRequest(
		t,
		app.handler,
		http.MethodPost,
		"/auth/login",
		`{"email":"alex@example.com","password":"password123"}`,
		"",
	)

	if login.StatusCode != http.StatusOK {
		t.Fatalf("expected login status %d, got %d", http.StatusOK, login.StatusCode)
	}

	body := decodeAuthJSON[auth.AuthResponse](t, login)

	if body.AccessToken == "" {
		t.Fatal("expected access token to be set")
	}

	if body.User.Email != "alex@example.com" {
		t.Fatalf("expected email %q, got %q", "alex@example.com", body.User.Email)
	}
}

func TestHandler_Login_WrongPassword(t *testing.T) {
	app := newTestAuthHTTPApp(t)

	register := executeAuthRequest(
		t,
		app.handler,
		http.MethodPost,
		"/auth/register",
		`{"name":"Alex","email":"alex@example.com","age":25,"password":"password123"}`,
		"",
	)

	if register.StatusCode != http.StatusCreated {
		t.Fatalf("expected register status %d, got %d", http.StatusCreated, register.StatusCode)
	}
	_ = register.Body.Close()

	login := executeAuthRequest(
		t,
		app.handler,
		http.MethodPost,
		"/auth/login",
		`{"email":"alex@example.com","password":"wrong-password"}`,
		"",
	)

	if login.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected login status %d, got %d", http.StatusUnauthorized, login.StatusCode)
	}

	_ = login.Body.Close()
}

func TestHandler_Me_MissingToken(t *testing.T) {
	app := newTestAuthHTTPApp(t)

	res := executeAuthRequest(
		t,
		app.handler,
		http.MethodGet,
		"/auth/me",
		"",
		"",
	)

	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, res.StatusCode)
	}

	_ = res.Body.Close()
}

func TestHandler_Me_Success(t *testing.T) {
	app := newTestAuthHTTPApp(t)

	register := executeAuthRequest(
		t,
		app.handler,
		http.MethodPost,
		"/auth/register",
		`{"name":"Alex","email":"alex@example.com","age":25,"password":"password123"}`,
		"",
	)

	if register.StatusCode != http.StatusCreated {
		t.Fatalf("expected register status %d, got %d", http.StatusCreated, register.StatusCode)
	}

	registerBody := decodeAuthJSON[auth.AuthResponse](t, register)

	me := executeAuthRequest(
		t,
		app.handler,
		http.MethodGet,
		"/auth/me",
		"",
		registerBody.AccessToken,
	)

	if me.StatusCode != http.StatusOK {
		t.Fatalf("expected me status %d, got %d", http.StatusOK, me.StatusCode)
	}

	meBody := decodeAuthJSON[auth.MeResponse](t, me)

	if meBody.User.ID != registerBody.User.ID {
		t.Fatalf("expected user id %d, got %d", registerBody.User.ID, meBody.User.ID)
	}

	if meBody.User.Email != "alex@example.com" {
		t.Fatalf("expected email %q, got %q", "alex@example.com", meBody.User.Email)
	}
}

func TestHandler_Me_InvalidToken(t *testing.T) {
	app := newTestAuthHTTPApp(t)

	res := executeAuthRequest(
		t,
		app.handler,
		http.MethodGet,
		"/auth/me",
		"",
		"invalid-token",
	)

	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, res.StatusCode)
	}

	_ = res.Body.Close()
}
