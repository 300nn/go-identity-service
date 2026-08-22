package user_test

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"CrudTutorialProject/internal/auth"
	"CrudTutorialProject/internal/response"
	"CrudTutorialProject/internal/user"
	"CrudTutorialProject/internal/validation"
)

type testUserApp struct {
	handler http.Handler
	repo    *FakeRepository
}

func newTestUserApp(t *testing.T) testUserApp {
	t.Helper()

	repo := newFakeRepository()
	hasher := auth.NewPasswordHasher()
	service := user.NewService(repo, nil, hasher)

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	validate := validation.New()

	handler := user.NewHandler(service, log, validate)

	mux := http.NewServeMux()

	handler.RegisterRoutes(
		mux,
		func(next http.Handler) http.Handler {
			return next
		},
		func(next http.Handler) http.Handler {
			return next
		},
	)

	return testUserApp{
		handler: mux,
		repo:    repo,
	}
}

func executeRequest(t *testing.T, handler http.Handler, method string, target string, body string) *http.Response {
	t.Helper()

	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	return rr.Result()
}

func decodeJSON[T any](t *testing.T, res *http.Response) T {
	t.Helper()

	var dst T

	if err := json.NewDecoder(res.Body).Decode(&dst); err != nil {
		t.Fatalf("decode response body: %v", err)
	}

	return dst
}

func closeResponseBody(t *testing.T, body io.ReadCloser) {
	t.Helper()

	if err := body.Close(); err != nil {
		t.Fatalf("closing body: %v", err)
	}
}

func TestHandler_CreateUser_Success(t *testing.T) {
	app := newTestUserApp(t)

	res := executeRequest(
		t,
		app.handler,
		http.MethodPost,
		"/users",
		`{"name":"Alex","email":"alex@example.com","age":25,"password":"strongpassword123"}`,
	)
	defer closeResponseBody(t, res.Body)

	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, res.StatusCode)
	}

	if contentType := res.Header.Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("expected Content-Type %q, got %q", "application/json", contentType)
	}

	got := decodeJSON[user.UserResponse](t, res)

	if got.ID == 0 {
		t.Fatal("expected user ID to be set")
	}

	if got.Name != "Alex" {
		t.Errorf("expected name %q, got %q", "Alex", got.Name)
	}

	if got.Email != "alex@example.com" {
		t.Errorf("expected email %q, got %q", "alex@example.com", got.Email)
	}

	if got.Age != 25 {
		t.Errorf("expected age %d, got %d", 25, got.Age)
	}
}

func TestHandler_CreateUser_ValidationError(t *testing.T) {
	app := newTestUserApp(t)

	res := executeRequest(
		t,
		app.handler,
		http.MethodPost,
		"/users",
		`{"name":"","email":"wrong","age":-1,"password":"123"}`,
	)
	defer closeResponseBody(t, res.Body)

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, res.StatusCode)
	}

	got := decodeJSON[response.ErrorBody](t, res)

	if got.Error.Code != "validation_error" {
		t.Fatalf("expected error code %q, got %q", "validation_error", got.Error.Code)
	}

	if got.Error.Fields["name"] == "" {
		t.Fatal("expected validation error for name")
	}

	if got.Error.Fields["email"] == "" {
		t.Fatal("expected validation error for email")
	}

	if got.Error.Fields["age"] == "" {
		t.Fatal("expected validation error for age")
	}

	if got.Error.Fields["password"] == "" {
		t.Fatal("expected validation error for password")
	}
}

func TestHandler_CreateUser_MalformedJSON(t *testing.T) {
	app := newTestUserApp(t)

	res := executeRequest(
		t,
		app.handler,
		http.MethodPost,
		"/users",
		`{"name":`,
	)
	defer closeResponseBody(t, res.Body)

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, res.StatusCode)
	}

	got := decodeJSON[response.ErrorBody](t, res)

	if got.Error.Code != "invalid_json" {
		t.Fatalf("expected error code %q, got %q", "invalid_json", got.Error.Code)
	}
}

func TestHandler_CreateUser_UnknownField(t *testing.T) {
	app := newTestUserApp(t)

	res := executeRequest(
		t,
		app.handler,
		http.MethodPost,
		"/users",
		`{"name":"Alex","email":"alex@example.com","age":25,"password":"strongpassword123","role":"admin"}`,
	)
	defer closeResponseBody(t, res.Body)

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, res.StatusCode)
	}

	got := decodeJSON[response.ErrorBody](t, res)

	if got.Error.Code != "unknown_json_field" {
		t.Fatalf("expected error code %q, got %q", "unknown_json_field", got.Error.Code)
	}
}

func TestHandler_GetUserByID_Success(t *testing.T) {
	app := newTestUserApp(t)

	created, err := app.repo.Create(t.Context(), user.User{
		Name:  "Alex",
		Email: "alex@example.com",
		Age:   25,
	})
	if err != nil {
		t.Fatalf("create fake user: %v", err)
	}

	res := executeRequest(
		t,
		app.handler,
		http.MethodGet,
		fmt.Sprintf("/users/%d", created.ID),
		"",
	)
	defer closeResponseBody(t, res.Body)

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.StatusCode)
	}

	got := decodeJSON[user.UserResponse](t, res)

	if got.ID != created.ID {
		t.Fatalf("expected id %d, got %d", created.ID, got.ID)
	}
}

func TestHandler_GetUserByID_NotFound(t *testing.T) {
	app := newTestUserApp(t)

	res := executeRequest(
		t,
		app.handler,
		http.MethodGet,
		"/users/999",
		"",
	)
	defer closeResponseBody(t, res.Body)

	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, res.StatusCode)
	}

	got := decodeJSON[response.ErrorBody](t, res)

	if got.Error.Code != "user_not_found" {
		t.Fatalf("expected error code %q, got %q", "user_not_found", got.Error.Code)
	}
}

func TestHandler_ListUsers(t *testing.T) {
	app := newTestUserApp(t)

	_, _ = app.repo.Create(t.Context(), user.User{
		Name:  "Alex",
		Email: "alex@example.com",
		Age:   25,
	})

	_, _ = app.repo.Create(t.Context(), user.User{
		Name:  "Bob",
		Email: "bob@example.com",
		Age:   30,
	})

	_, _ = app.repo.Create(t.Context(), user.User{
		Name:  "Alice",
		Email: "alice@test.com",
		Age:   22,
	})

	res := executeRequest(
		t,
		app.handler,
		http.MethodGet,
		"/users?limit=1&offset=0&email=example&sort=id_asc",
		"",
	)
	defer closeResponseBody(t, res.Body)

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.StatusCode)
	}

	got := decodeJSON[response.Page[user.UserResponse]](t, res)

	if got.Pagination.Total != 2 {
		t.Fatalf("expected total %d, got %d", 2, got.Pagination.Total)
	}

	if len(got.Items) != 1 {
		t.Fatalf("expected items len %d, got %d", 1, len(got.Items))
	}

	if got.Items[0].Email != "alex@example.com" {
		t.Fatalf("expected first email %q, got %q", "alex@example.com", got.Items[0].Email)
	}
}

func TestHandler_ListUsers_InvalidLimit(t *testing.T) {
	app := newTestUserApp(t)

	res := executeRequest(
		t,
		app.handler,
		http.MethodGet,
		"/users?limit=abc",
		"",
	)
	defer closeResponseBody(t, res.Body)

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, res.StatusCode)
	}

	got := decodeJSON[response.ErrorBody](t, res)

	if got.Error.Code != "validation_error" {
		t.Fatalf("expected error code %q, got %q", "validation_error", got.Error.Code)
	}

	if got.Error.Fields["limit"] == "" {
		t.Fatal("expected validation error for limit")
	}
}

func TestHandler_ListUsers_InvalidSort(t *testing.T) {
	app := newTestUserApp(t)

	sortValue := url.QueryEscape("id;drop table users")

	res := executeRequest(
		t,
		app.handler,
		http.MethodGet,
		"/users?sort="+sortValue,
		"",
	)
	defer closeResponseBody(t, res.Body)

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, res.StatusCode)
	}

	got := decodeJSON[response.ErrorBody](t, res)

	if got.Error.Code != "validation_error" {
		t.Fatalf("expected error code %q, got %q", "validation_error", got.Error.Code)
	}

	if got.Error.Fields["sort"] == "" {
		t.Fatal("expected validation error for sort")
	}
}
