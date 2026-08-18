package auth

import (
	"CrudTutorialProject/internal/response"
	"CrudTutorialProject/internal/user"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type MiddleWare struct {
	tokens *TokenManager
}

type ResourceIDExtractor func(r *http.Request) (int64, error)

func NewMiddleWare(tokens *TokenManager) *MiddleWare {
	return &MiddleWare{
		tokens: tokens,
	}
}

func (m *MiddleWare) RequireSelfOrRole(extractResourceUserID ResourceIDExtractor, roles ...user.Role) func(http.Handler) http.Handler {
	allowed := make(map[user.Role]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return m.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, ok := PrincipalFromContext(r.Context())
			if !ok {
				response.SendError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized")
				return
			}
			if _, ok := allowed[principal.Role]; ok {
				next.ServeHTTP(w, r)
				return
			}
			resourceUserID, err := extractResourceUserID(r)

			if err != nil {
				response.SendError(w, http.StatusBadRequest, "invalid_path_param", "Invalid path parameter")
				return
			}

			if principal.UserID != resourceUserID {
				response.SendError(w, http.StatusForbidden, "forbidden", "Forbidden")
				return
			}

			next.ServeHTTP(w, r)
		}))
	}
}

func (m *MiddleWare) RequireRole(roles ...user.Role) func(http.Handler) http.Handler {
	allowed := make(map[user.Role]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return m.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, ok := PrincipalFromContext(r.Context())
			if !ok {
				response.SendError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized")
				return
			}

			if _, ok := allowed[principal.Role]; !ok {
				response.SendError(w, http.StatusForbidden, "forbidden", "Forbidden")
				return
			}

			next.ServeHTTP(w, r)
		}))
	}
}

func (m *MiddleWare) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := strings.TrimSpace(r.Header.Get("Authorization"))

		if header == "" {
			response.SendError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized")
			return
		}

		const prefix = "Bearer "
		if !strings.HasPrefix(header, prefix) {
			response.SendError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized")
			return
		}

		tokenString := strings.TrimSpace(strings.TrimPrefix(header, prefix))

		if tokenString == "" {
			response.SendError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized")
			return
		}

		claims, err := m.tokens.Parse(tokenString)

		if err != nil {
			response.SendError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized")
			return
		}

		ctx := ContextWithUser(r.Context(), Principal{
			UserID: claims.UserID,
			Email:  claims.Email,
			Role:   user.Role(claims.Role),
		})

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func PathInt64Param(name string) ResourceIDExtractor {
	return func(r *http.Request) (int64, error) {
		raw := strings.TrimSpace(r.PathValue(name))
		if raw == "" {
			return 0, fmt.Errorf("path parameter %s is required", name)
		}
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("path parameter %s must be integer: %w", name, err)
		}

		if id <= 0 {
			return 0, fmt.Errorf("path parameter %s must be positive", name)
		}

		return id, nil
	}
}
