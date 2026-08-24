package grpcapi

import (
	"context"

	"github.com/300nn/go-identity-service/internal/auth"
	"github.com/300nn/go-identity-service/internal/user"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func requirePrincipal(ctx context.Context) (auth.Principal, error) {
	principal, ok := auth.PrincipalFromContext(ctx)
	if !ok {
		return auth.Principal{}, status.Error(codes.Unauthenticated, "unauthenticated")
	}

	return principal, nil
}

func requireRole(ctx context.Context, roles ...user.Role) error {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return err
	}

	for _, role := range roles {
		if principal.Role == role {
			return nil
		}
	}

	return status.Error(codes.PermissionDenied, "permission denied")
}

func requireSelfOrRole(ctx context.Context, resourceUserID int64, roles ...user.Role) error {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return err
	}

	for _, role := range roles {
		if principal.Role == role {
			return nil
		}
	}

	if principal.UserID == resourceUserID {
		return nil
	}

	return status.Error(codes.PermissionDenied, "permission denied")
}
