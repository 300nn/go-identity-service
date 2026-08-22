package auth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"CrudTutorialProject/internal/auth"
	"CrudTutorialProject/internal/user"
)

func TestMiddleware_RequireAuth_MissingHeader(t *testing.T) {
	middleware := auth.NewMiddleWare(
		auth.NewTokenManager(testJWTSecret, 15*time.Minute, "go-crud-api"),
	)

	nextCalled := false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rr := httptest.NewRecorder()

	middleware.RequireAuth(next).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}

	if nextCalled {
		t.Fatal("next handler must not be called")
	}
}

func TestMiddleware_RequireAuth_ValidToken(t *testing.T) {
	tokenManager := auth.NewTokenManager(testJWTSecret, 15*time.Minute, "go-crud-api")
	middleware := auth.NewMiddleWare(tokenManager)

	token, err := tokenManager.Generate(123, "alex@example.com", "ADMIN")

	if err != nil {
		t.Fatalf("error generating token: %v", err)
	}

	nextCalled := false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true

		principal, ok := auth.PrincipalFromContext(r.Context())

		if !ok {
			t.Fatal("principal not found in context")
		}

		if principal.UserID != 123 {
			t.Fatalf("expected principal user id %d, got %d", 123, principal.UserID)
		}

		if principal.Email != "alex@example.com" {
			t.Fatalf("expected principal email %s, got %s", "alex@example.com", principal.Email)
		}

		if principal.Role != user.RoleAdmin {
			t.Fatalf("expected principal role %s, got %s", user.RoleAdmin, principal.Role)
		}

		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	middleware.RequireAuth(next).ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rr.Code)
	}
	if !nextCalled {
		t.Fatal("expected next handler to be called")
	}
}

func TestMiddleware_RequireAuth_InvalidToken(t *testing.T) {
	tokenManager := auth.NewTokenManager(testJWTSecret, 15*time.Minute, "go-crud-api")
	middleware := auth.NewMiddleWare(tokenManager)

	nextCalled := false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")

	rr := httptest.NewRecorder()

	middleware.RequireAuth(next).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}

	if nextCalled {
		t.Fatal("next handler must not be called")
	}
}

func TestMiddleware_RequireRole_AllowsAdmin(t *testing.T) {
	tokenManager := auth.NewTokenManager(testJWTSecret, 15*time.Minute, "go-crud-api")
	middleware := auth.NewMiddleWare(tokenManager)

	token, err := tokenManager.Generate(123, "admin@example.com", string(user.RoleAdmin))
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	nextCalled := false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()

	middleware.RequireRole(user.RoleAdmin)(next).ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rr.Code)
	}

	if !nextCalled {
		t.Fatal("expected next handler to be called")
	}
}

func TestMiddleware_RequireRole_ForbidsUser(t *testing.T) {
	tokenManager := auth.NewTokenManager(testJWTSecret, 15*time.Minute, "go-crud-api")
	middleware := auth.NewMiddleWare(tokenManager)

	token, err := tokenManager.Generate(123, "user@example.com", string(user.RoleUser))
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	nextCalled := false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()

	middleware.RequireRole(user.RoleAdmin)(next).ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rr.Code)
	}

	if nextCalled {
		t.Fatal("next handler must not be called")
	}
}

func TestMiddleware_RequireSelfOrRole_AllowsAdmin(t *testing.T) {
	tokenManager := auth.NewTokenManager(testJWTSecret, 15*time.Minute, "go-crud-api")
	middleware := auth.NewMiddleWare(tokenManager)

	token, err := tokenManager.Generate(1, "admin@example.com", string(user.RoleAdmin))
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	nextCalled := false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/users/999", nil)
	req.SetPathValue("id", "999")
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()

	middleware.RequireSelfOrRole(
		auth.PathInt64Param("id"),
		user.RoleAdmin,
	)(next).ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rr.Code)
	}

	if !nextCalled {
		t.Fatal("expected next handler to be called")
	}
}

func TestMiddleware_RequireSelfOrRole_AllowsSelf(t *testing.T) {
	tokenManager := auth.NewTokenManager(testJWTSecret, 15*time.Minute, "go-crud-api")
	middleware := auth.NewMiddleWare(tokenManager)

	token, err := tokenManager.Generate(123, "user@example.com", string(user.RoleUser))
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	nextCalled := false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	req.SetPathValue("id", "123")
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()

	middleware.RequireSelfOrRole(
		auth.PathInt64Param("id"),
		user.RoleAdmin,
	)(next).ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rr.Code)
	}

	if !nextCalled {
		t.Fatal("expected next handler to be called")
	}
}

func TestMiddleware_RequireSelfOrRole_ForbidsOtherUser(t *testing.T) {
	tokenManager := auth.NewTokenManager(testJWTSecret, 15*time.Minute, "go-crud-api")
	middleware := auth.NewMiddleWare(tokenManager)

	token, err := tokenManager.Generate(123, "user@example.com", string(user.RoleUser))
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	nextCalled := false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	req := httptest.NewRequest(http.MethodGet, "/users/999", nil)
	req.SetPathValue("id", "999")
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()

	middleware.RequireSelfOrRole(
		auth.PathInt64Param("id"),
		user.RoleAdmin,
	)(next).ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rr.Code)
	}

	if nextCalled {
		t.Fatal("next handler must not be called")
	}
}

func TestMiddleware_RequireSelfOrRole_MissingToken(t *testing.T) {
	tokenManager := auth.NewTokenManager(testJWTSecret, 15*time.Minute, "go-crud-api")
	middleware := auth.NewMiddleWare(tokenManager)

	nextCalled := false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	req.SetPathValue("id", "123")

	rr := httptest.NewRecorder()

	middleware.RequireSelfOrRole(
		auth.PathInt64Param("id"),
		user.RoleAdmin,
	)(next).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}

	if nextCalled {
		t.Fatal("next handler must not be called")
	}
}

func TestMiddleware_RequireSelfOrRole_InvalidPathID(t *testing.T) {
	tokenManager := auth.NewTokenManager(testJWTSecret, 15*time.Minute, "go-crud-api")
	middleware := auth.NewMiddleWare(tokenManager)

	token, err := tokenManager.Generate(123, "user@example.com", string(user.RoleUser))
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	nextCalled := false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	req := httptest.NewRequest(http.MethodGet, "/users/abc", nil)
	req.SetPathValue("id", "abc")
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()

	middleware.RequireSelfOrRole(
		auth.PathInt64Param("id"),
		user.RoleAdmin,
	)(next).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	if nextCalled {
		t.Fatal("next handler must not be called")
	}
}
