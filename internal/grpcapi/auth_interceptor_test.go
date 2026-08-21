package grpcapi_test

import (
	"context"
	"testing"
	"time"

	"CrudTutorialProject/internal/auth"
	"CrudTutorialProject/internal/grpcapi"
	"CrudTutorialProject/internal/user"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const testJWTSecret = "test-secret-must-be-at-least-32-chars"

func TestAuthInterceptor_Unary_AttachesPrincipal(t *testing.T) {
	ctx := t.Context()

	tokens := auth.NewTokenManager(testJWTSecret, time.Minute, "test-service")

	token, err := tokens.Generate(123, "alex@example.com", string(user.RoleAdmin))
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	ctx = metadata.NewIncomingContext(ctx, metadata.Pairs(
		"authorization", "Bearer "+token,
	))

	interceptor := grpcapi.NewAuthInterceptor(tokens)

	_, err = interceptor.Unary()(
		ctx,
		nil,
		&grpc.UnaryServerInfo{
			FullMethod: "/api.user.v1.UserService/GetUser",
		},
		func(ctx context.Context, req any) (any, error) {
			principal, ok := auth.PrincipalFromContext(ctx)
			if !ok {
				t.Fatal("expected principal in context")
			}

			if principal.UserID != 123 {
				t.Fatalf("expected user id 123, got %d", principal.UserID)
			}

			if principal.Role != user.RoleAdmin {
				t.Fatalf("expected role ADMIN, got %s", principal.Role)
			}

			return "ok", nil
		},
	)
	if err != nil {
		t.Fatalf("interceptor returned error: %v", err)
	}
}

func TestAuthInterceptor_Unary_InvalidToken(t *testing.T) {
	ctx := metadata.NewIncomingContext(t.Context(), metadata.Pairs(
		"authorization", "Bearer invalid-token",
	))

	tokens := auth.NewTokenManager(testJWTSecret, time.Minute, "test-service")
	interceptor := grpcapi.NewAuthInterceptor(tokens)

	_, err := interceptor.Unary()(
		ctx,
		nil,
		&grpc.UnaryServerInfo{
			FullMethod: "/api.user.v1.UserService/GetUser",
		},
		func(ctx context.Context, req any) (any, error) {
			t.Fatal("handler should not be called")
			return nil, nil
		},
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestAuthInterceptor_Unary_NoAuthorizationHeader_CallsHandler(t *testing.T) {
	tokens := auth.NewTokenManager(testJWTSecret, time.Minute, "test-service")
	interceptor := grpcapi.NewAuthInterceptor(tokens)

	called := false

	_, err := interceptor.Unary()(
		t.Context(),
		nil,
		&grpc.UnaryServerInfo{
			FullMethod: "/grpc.reflection.v1.ServerReflection/ServerReflectionInfo",
		},
		func(ctx context.Context, req any) (any, error) {
			called = true
			return "ok", nil
		},
	)
	if err != nil {
		t.Fatalf("interceptor returned error: %v", err)
	}

	if !called {
		t.Fatal("expected handler to be called")
	}
}
