package auth

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"CrudTutorialProject/internal/eventcodec"
	"CrudTutorialProject/internal/outbox"
	"CrudTutorialProject/internal/user"
)

const dummyPasswordHash = "$2a$12$e80yq9gIe67Cqg4a0d9I6.L971nJ9.xP7pB/64QZz.7iG8JgP1G2m"

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
	txFactory       TxFactory
	hasher          *PasswordHasher
	tokens          *TokenManager
	refreshTokens   *RefreshTokenManager
	refreshTokenTTL time.Duration
}

func NewService(
	userRepo UserStore,
	refreshRepo RefreshTokenStore,
	txFactory TxFactory,
	hasher *PasswordHasher,
	tokens *TokenManager,
	refreshTokens *RefreshTokenManager,
	refreshTokenTTL time.Duration,
) *Service {
	return &Service{
		userRepo:        userRepo,
		refreshRepo:     refreshRepo,
		txFactory:       txFactory,
		hasher:          hasher,
		tokens:          tokens,
		refreshTokens:   refreshTokens,
		refreshTokenTTL: refreshTokenTTL,
	}
}

func (s *Service) Register(ctx context.Context, req RegisterRequest) (AuthResponse, error) {
	name := strings.TrimSpace(req.Name)
	email := strings.TrimSpace(strings.ToLower(req.Email))

	passwordHash, err := s.hasher.Hash(req.Password)
	if err != nil {
		return AuthResponse{}, err
	}

	var result AuthResponse

	err = s.txFactory.WithinTx(ctx, func(stores TxStores) error {
		exist, err := stores.UserStore.ExistsByEmail(ctx, email)
		if err != nil {
			return fmt.Errorf("check email exists: %w", err)
		}
		if exist {
			return user.NewEmailAlreadyExistsError()
		}

		created, err := stores.UserStore.Create(ctx, user.User{
			Name:         name,
			Email:        email,
			Age:          req.Age,
			PasswordHash: passwordHash,
			Role:         user.RoleUser,
		})

		if err != nil {
			if errors.Is(err, user.ErrEmailAlreadyExists) {
				return user.NewEmailAlreadyExistsError()
			}
			return err
		}

		accessToken, refreshToken, err := s.issueTokenPairWithStore(ctx, stores.RefreshTokenStore, created)

		if err != nil {
			return err
		}

		payload, err := eventcodec.MarshalUserRegistered(
			created.ID,
			created.Email,
			string(created.Role),
		)

		if err != nil {
			return err
		}

		_, err = stores.OutboxStore.Create(ctx, outbox.Event{
			EventType:     outbox.EventTypeUserRegistered,
			AggregateType: outbox.AggregateUser,
			AggregateID:   strconv.FormatInt(created.ID, 10),
			Payload:       payload,
			ContentType:   eventcodec.ContentTypeProtobuf,
			ProtoMessage:  eventcodec.ProtoMessageUserRegistered,
			EventVersion:  eventcodec.EventVersionV1,
		})

		if err != nil {
			return fmt.Errorf("create user registered outbox event: %w", err)
		}

		result = AuthResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			TokenType:    "Bearer",
			User:         user.ToResponse(created),
		}

		return nil
	})

	if err != nil {
		return AuthResponse{}, err
	}

	return result, nil
}

func (s *Service) Login(ctx context.Context, req LoginRequest) (AuthResponse, error) {
	email := strings.TrimSpace(strings.ToLower(req.Email))

	found, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		s.hasher.Compare(dummyPasswordHash, req.Password)
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

	var result AuthResponse

	err := s.txFactory.WithinTx(ctx, func(stores TxStores) error {
		storedToken, err := stores.RefreshTokenStore.FindRefreshTokenByHash(ctx, hash)
		if err != nil {
			if errors.Is(err, ErrRefreshTokenNotFound) {
				return NewUnauthorizedError()
			}
			return fmt.Errorf("find refresh token by hash: %w", err)
		}

		now := time.Now().UTC()

		if !storedToken.IsActive(now) {
			return NewUnauthorizedError()
		}

		usr, err := stores.UserStore.FindByID(ctx, storedToken.UserID)
		if err != nil {
			if errors.Is(err, user.ErrUserNotFound) {
				return NewUnauthorizedError()
			}
			return fmt.Errorf("find user by id: %w", err)
		}

		accessToken, refreshToken, err := s.issueTokenPairWithStore(ctx, stores.RefreshTokenStore, usr)

		if err != nil {
			return err
		}

		if err := stores.RefreshTokenStore.RevokeRefreshToken(ctx, storedToken.ID); err != nil {
			return fmt.Errorf("revoke old refresh token: %w", err)
		}

		result = AuthResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			TokenType:    "Bearer",
			User:         user.ToResponse(usr),
		}

		return nil
	})

	if err != nil {
		return AuthResponse{}, err
	}

	return result, nil
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

func (s *Service) issueTokenPairWithStore(
	ctx context.Context,
	refreshRepo RefreshTokenStore,
	usr user.User,
) (string, string, error) {
	accessToken, err := s.tokens.Generate(
		usr.ID,
		usr.Email,
		string(usr.Role),
	)

	if err != nil {
		return "", "", err
	}

	refreshToken, refreshTokenHash, err := s.refreshTokens.Generate()

	if err != nil {
		return "", "", err
	}

	_, err = refreshRepo.CreateRefreshToken(
		ctx,
		RefreshToken{
			UserID:    usr.ID,
			TokenHash: refreshTokenHash,
			ExpiresAt: time.Now().UTC().Add(s.refreshTokenTTL),
		},
	)

	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func (s *Service) issueTokenPair(ctx context.Context, user user.User) (string, string, error) {
	return s.issueTokenPairWithStore(ctx, s.refreshRepo, user)
}
