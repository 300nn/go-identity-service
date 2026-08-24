package auth

import (
	"context"

	"github.com/300nn/go-identity-service/internal/user"
)

type Principal struct {
	UserID int64
	Email  string
	Role   user.Role
}

type contextKey string

const principalKey contextKey = "auth.principal"

func ContextWithUser(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalKey, principal)
}

func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(principalKey).(Principal)
	return principal, ok
}
