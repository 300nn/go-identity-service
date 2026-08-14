package auth

import (
	"CrudTutorialProject/internal/user"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

type UserStore interface {
	Create(ctx context.Context, u user.User) (user.User, error)
	FindByID(ctx context.Context, id int64) (user.User, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	FindByEmail(ctx context.Context, email string) (user.User, error)
}

type RefreshTokenStore interface {
	CreateRefreshToken(ctx context.Context, token RefreshToken) (RefreshToken, error)
	FindRefreshTokenByHash(ctx context.Context, hash string) (RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, id int64) error
}

type Service struct {
	userRepo        UserStore
	refreshRepo     RefreshTokenStore
	hasher          *PasswordHasher
	tokens          *TokenManager
	refreshTokens   *RefreshTokenManager
	refreshTokenTTL time.Duration
}

func NewService(
	userRepo UserStore,
	refreshRepo RefreshTokenStore,
	hasher *PasswordHasher,
	tokens *TokenManager,
	refreshTokens *RefreshTokenManager,
	refreshTokenTTL time.Duration,
) *Service {
	return &Service{
		userRepo:        userRepo,
		refreshRepo:     refreshRepo,
		hasher:          hasher,
		tokens:          tokens,
		refreshTokens:   refreshTokens,
		refreshTokenTTL: refreshTokenTTL,
	}
}

func (s *Service) Register(ctx context.Context, req RegisterRequest) (AuthResponse, error) {
	name := strings.TrimSpace(req.Name)
	email := strings.TrimSpace(strings.ToLower(req.Email))

	exist, err := s.userRepo.ExistsByEmail(ctx, email)
	if err != nil {
		return AuthResponse{}, fmt.Errorf("check email exists: %w", err)
	}
	if exist {
		return AuthResponse{}, user.NewEmailAlreadyExistsError()
	}

	passwordHash, err := s.hasher.Hash(req.Password)
	if err != nil {
		return AuthResponse{}, err
	}

	created, err := s.userRepo.Create(ctx, user.User{
		Name:         name,
		Email:        email,
		Age:          req.Age,
		PasswordHash: passwordHash,
		Role:         user.RoleUser,
	})

	if err != nil {
		if errors.Is(err, user.ErrEmailAlreadyExists) {
			return AuthResponse{}, user.NewEmailAlreadyExistsError()
		}
		return AuthResponse{}, err
	}

	accessToken, refreshToken, err := s.issueTokenPair(ctx, created)

	if err != nil {
		return AuthResponse{}, err
	}

	return AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		User:         user.ToResponse(created),
	}, nil
}

func (s *Service) Login(ctx context.Context, req LoginRequest) (AuthResponse, error) {
	email := strings.TrimSpace(strings.ToLower(req.Email))

	found, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			return AuthResponse{}, NewInvalidCredentialsError()
		}
		return AuthResponse{}, fmt.Errorf("find user by email: %w", err)
	}

	if !s.hasher.Compare(found.PasswordHash, req.Password) {
		return AuthResponse{}, NewInvalidCredentialsError()
	}

	accessToken, refreshToken, err := s.issueTokenPair(ctx, found)
	if err != nil {
		return AuthResponse{}, err
	}

	return AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		User:         user.ToResponse(found),
	}, nil
}

func (s *Service) Me(ctx context.Context, userId int64) (MeResponse, error) {
	found, err := s.userRepo.FindByID(ctx, userId)

	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			return MeResponse{}, user.NewUserNotFoundError()
		}
		return MeResponse{}, err
	}

	return MeResponse{
		User: user.ToResponse(found),
	}, nil
}

func (s *Service) Refresh(ctx context.Context, req RefreshRequest) (AuthResponse, error) {
	hash := s.refreshTokens.Hash(req.RefreshToken)

	storedToken, err := s.refreshRepo.FindRefreshTokenByHash(ctx, hash)

	if err != nil {
		if errors.Is(err, ErrRefreshTokenNotFound) {
			return AuthResponse{}, NewUnauthorizedError()
		}
		return AuthResponse{}, fmt.Errorf("find refresh token by hash: %w", err)
	}

	now := time.Now()

	if !storedToken.IsActive(now) {
		return AuthResponse{}, NewUnauthorizedError()
	}

	usr, err := s.userRepo.FindByID(ctx, storedToken.UserID)

	if err != nil {
		return AuthResponse{}, fmt.Errorf("find usr by id: %w", err)
	}

	if err := s.refreshRepo.RevokeRefreshToken(ctx, storedToken.ID); err != nil {
		return AuthResponse{}, fmt.Errorf("revoke old refresh token: %w", err)
	}

	accessToken, refreshToken, err := s.issueTokenPair(ctx, usr)
	if err != nil {
		return AuthResponse{}, err
	}

	return AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		User:         user.ToResponse(usr),
	}, nil
}

func (s *Service) Logout(ctx context.Context, req LogoutRequest) error {
	hash := s.refreshTokens.Hash(req.RefreshToken)

	storedToken, err := s.refreshRepo.FindRefreshTokenByHash(ctx, hash)

	if err != nil {
		if errors.Is(err, ErrRefreshTokenNotFound) {
			return nil
		}

		return fmt.Errorf("find refresh token: %w", err)
	}

	if storedToken.IsRevoked() {
		return nil
	}

	if err := s.refreshRepo.RevokeRefreshToken(ctx, storedToken.ID); err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}

	return nil
}

func (s *Service) issueTokenPair(ctx context.Context, user user.User) (string, string, error) {
	accessToken, err := s.tokens.Generate(
		user.ID,
		user.Email,
		string(user.Role),
	)

	if err != nil {
		return "", "", err
	}

	refreshToken, refreshTokenHash, err := s.refreshTokens.Generate()

	if err != nil {
		return "", "", err
	}

	_, err = s.refreshRepo.CreateRefreshToken(
		ctx,
		RefreshToken{
			UserID:    user.ID,
			TokenHash: refreshTokenHash,
			ExpiresAt: time.Now().UTC().Add(s.refreshTokenTTL),
		},
	)

	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}
