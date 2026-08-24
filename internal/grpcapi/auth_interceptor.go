package grpcapi

import (
	"context"
	"strings"

	"github.com/300nn/go-identity-service/internal/auth"
	"github.com/300nn/go-identity-service/internal/user"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type AuthInterceptor struct {
	tokens *auth.TokenManager
}

func NewAuthInterceptor(tokens *auth.TokenManager) *AuthInterceptor {
	return &AuthInterceptor{
		tokens: tokens,
	}
}

func (i *AuthInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		header := authorizationHeader(ctx)

		if header == "" {
			return handler(ctx, req)
		}

		const prefix = "Bearer "
		if !strings.HasPrefix(header, prefix) {
			return nil, status.Error(codes.Unauthenticated, "invalid authorization header")
		}

		tokenString := strings.TrimSpace(strings.TrimPrefix(header, prefix))
		if tokenString == "" {
			return nil, status.Error(codes.Unauthenticated, "empty bearer token")
		}

		claims, err := i.tokens.Parse(tokenString)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}

		role := user.Role(claims.Role)
		if !role.IsValid() {
			return nil, status.Error(codes.Unauthenticated, "invalid token role")
		}

		ctx = auth.ContextWithUser(ctx, auth.Principal{
			UserID: claims.UserID,
			Email:  claims.Email,
			Role:   role,
		})

		return handler(ctx, req)
	}
}

func authorizationHeader(ctx context.Context) string {
	values := metadata.ValueFromIncomingContext(ctx, "authorization")
	if len(values) == 0 {
		return ""
	}

	return strings.TrimSpace(values[0])
}
