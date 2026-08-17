package auth_test

import (
	"CrudTutorialProject/internal/auth"
	"CrudTutorialProject/internal/ratelimit"
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
	handler     http.Handler
	userRepo    *fakeUserRepository
	refreshRepo *fakeRefreshTokenRepository
	txFactory   *fakeTxFactory
	hasher      *auth.PasswordHasher
	tokens      *auth.TokenManager
	refresh     *auth.RefreshTokenManager
}

func newTestAuthHTTPApp(t *testing.T) *testAuthHTTPApp {
	t.Helper()

	repo := newFakeUserRepository()
	hasher := auth.NewPasswordHasherWithCost(bcrypt.MinCost)
	tokens := auth.NewTokenManager(testJWTSecret, 15*time.Minute, "go-crud-api")
	refreshTokens := auth.NewRefreshTokenManager()
	refreshTokenRepo := newFakeRefreshTokenRepository()
	txFactory := newFakeTxFactory(repo, refreshTokenRepo)

	service := auth.NewService(repo, refreshTokenRepo, txFactory, hasher, tokens, refreshTokens, 15*time.Hour)

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	validator := validation.New()

	limiter := ratelimit.NewLimiter()
	limiterConfig := auth.RateLimitConfig{
		LoginLimit:     100,
		LoginWindow:    time.Minute,
		RegisterLimit:  100,
		RegisterWindow: time.Minute,
		RefreshLimit:   100,
		RefreshWindow:  time.Minute,
	}

	authHandler := auth.NewHandler(service, log, validator, limiter, limiterConfig)
	authMiddleware := auth.NewMiddleWare(tokens)

	mux := http.NewServeMux()
	authHandler.RegisterRoutes(mux, authMiddleware)

	return &testAuthHTTPApp{
		handler:     mux,
		userRepo:    repo,
		refreshRepo: refreshTokenRepo,
		txFactory:   txFactory,
		hasher:      hasher,
		tokens:      tokens,
		refresh:     refreshTokens,
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
			t.Fatalf("failed to close response body: %v", err)
		}
	}(res.Body)

	var dst T
	if err := json.NewDecoder(res.Body).Decode(&dst); err != nil {
		t.Fatalf("failed to decode auth response: %v", err)
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

	if body.RefreshToken == "" {
		t.Fatal("expected refresh token to be set")
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

	saved, err := app.userRepo.FindByEmail(t.Context(), "alex@example.com")

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

	if body.RefreshToken == "" {
		t.Fatal("expected refresh token to be set")
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

func TestHandler_Refresh_Success(t *testing.T) {
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

	refresh := executeAuthRequest(
		t,
		app.handler,
		http.MethodPost,
		"/auth/refresh",
		`{"refreshToken":"`+registerBody.RefreshToken+`"}`,
		"",
	)

	if refresh.StatusCode != http.StatusOK {
		t.Fatalf("expected refresh status %d, got %d", http.StatusOK, refresh.StatusCode)
	}

	refreshBody := decodeAuthJSON[auth.AuthResponse](t, refresh)

	if refreshBody.AccessToken == "" {
		t.Fatal("expected access token to be set")
	}

	if refreshBody.RefreshToken == "" {
		t.Fatal("expected refresh token to be set")
	}

	if refreshBody.RefreshToken == registerBody.RefreshToken {
		t.Fatal("expected refresh token rotation")
	}
}

func TestHandler_Refresh_RejectsOldTokenAfterRotation(t *testing.T) {
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

	first := executeAuthRequest(
		t,
		app.handler,
		http.MethodPost,
		"/auth/refresh",
		`{"refreshToken":"`+registerBody.RefreshToken+`"}`,
		"",
	)

	if first.StatusCode != http.StatusOK {
		t.Fatalf("expected first refresh status %d, got %d", http.StatusOK, first.StatusCode)
	}
	_ = first.Body.Close()

	second := executeAuthRequest(
		t,
		app.handler,
		http.MethodPost,
		"/auth/refresh",
		`{"refreshToken":"`+registerBody.RefreshToken+`"}`,
		"",
	)

	if second.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected second refresh status %d, got %d", http.StatusUnauthorized, second.StatusCode)
	}

	_ = second.Body.Close()
}

func TestHandler_Logout_Success(t *testing.T) {
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

	logout := executeAuthRequest(
		t,
		app.handler,
		http.MethodPost,
		"/auth/logout",
		`{"refreshToken":"`+registerBody.RefreshToken+`"}`,
		"",
	)

	if logout.StatusCode != http.StatusNoContent {
		t.Fatalf("expected logout status %d, got %d", http.StatusNoContent, logout.StatusCode)
	}
	_ = logout.Body.Close()

	refresh := executeAuthRequest(
		t,
		app.handler,
		http.MethodPost,
		"/auth/refresh",
		`{"refreshToken":"`+registerBody.RefreshToken+`"}`,
		"",
	)

	if refresh.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected refresh after logout status %d, got %d", http.StatusUnauthorized, refresh.StatusCode)
	}

	_ = refresh.Body.Close()
}
