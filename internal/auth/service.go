package auth

import (
	"CrudTutorialProject/internal/user"
	"context"
	"errors"
	"fmt"
	"strings"
)

type UserStore interface {
	Create(ctx context.Context, u user.User) (user.User, error)
	FindByID(ctx context.Context, id int64) (user.User, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	FindByEmail(ctx context.Context, email string) (user.User, error)
}

type Service struct {
	userRepo UserStore
	hasher   *PasswordHasher
	tokens   *TokenManager
}

func NewService(userRepo UserStore, hasher *PasswordHasher, tokens *TokenManager) *Service {
	return &Service{
		userRepo: userRepo,
		hasher:   hasher,
		tokens:   tokens,
	}
}

func (s *Service) Register(ctx context.Context, req RegisterRequest) (AuthResponse, error) {
	name := strings.TrimSpace(req.Name)
	email := strings.TrimSpace(req.Email)

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
	})

	if err != nil {
		if errors.Is(err, user.ErrEmailAlreadyExists) {
			return AuthResponse{}, user.NewEmailAlreadyExistsError()
		}
		return AuthResponse{}, err
	}

	token, err := s.tokens.Generate(created.ID, created.Email)

	if err != nil {
		return AuthResponse{}, err
	}

	return AuthResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		User:        user.ToResponse(created),
	}, nil
}

func (s *Service) Login(ctx context.Context, req LoginRequest) (AuthResponse, error) {
	email := strings.TrimSpace(req.Email)

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

	token, err := s.tokens.Generate(found.ID, found.Email)
	if err != nil {
		return AuthResponse{}, err
	}

	return AuthResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		User:        user.ToResponse(found),
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
