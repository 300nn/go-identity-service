package user

import (
	"CrudTutorialProject/internal/apperror"
	"context"
	"errors"
	"fmt"
	"net/mail"
	"sort"
	"strings"
)

type Service struct {
	repo      Repository
	txFactory TxRepositoryFactory
}

func NewService(repo Repository, txFactory TxRepositoryFactory) *Service {
	return &Service{
		repo:      repo,
		txFactory: txFactory,
	}
}

type CreateUserInput struct {
	Name  string
	Email string
	Age   int
}

type UpdateUserInput struct {
	Name  string
	Email string
	Age   int
}

type CreateUserWithProfileInput struct {
	Name  string
	Email string
	Age   int
	Bio   string
}

type ProfileWithUser struct {
	User    User
	Profile Profile
}

func (s *Service) CreateUser(ctx context.Context, request CreateUserInput) (User, error) {
	name := strings.TrimSpace(request.Name)
	email := strings.TrimSpace(request.Email)
	age := request.Age

	if err := validateUserInput(name, email, age); err != nil {
		return User{}, err
	}

	exists, err := s.repo.ExistsByEmail(ctx, email)

	if err != nil {
		return User{}, fmt.Errorf("create user: %w", err)
	}

	if exists {
		return User{}, NewEmailAlreadyExistsError()
	}

	created, err := s.repo.Create(ctx, User{
		Name:  name,
		Email: email,
		Age:   age,
	})

	if err != nil {
		return User{}, fmt.Errorf("create user: %w", err)
	}

	return created, nil
}

func (s *Service) GetUser(ctx context.Context, id int64) (User, error) {
	if id <= 0 {
		return User{}, NewInvalidUserIDError()
	}

	user, err := s.repo.FindByID(ctx, id)

	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return User{}, NewUserNotFoundError()
		}
		return User{}, fmt.Errorf("find user by id %d: %w", id, err)
	}

	return user, nil
}

func (s *Service) ListUsers(ctx context.Context) ([]User, error) {
	users, err := s.repo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("find all users: %w", err)
	}

	sort.Slice(users, func(i, j int) bool {
		return users[i].ID < users[j].ID
	})

	return users, nil
}

func (s *Service) UpdateUser(ctx context.Context, id int64, user UpdateUserInput) (User, error) {
	if id <= 0 {
		return User{}, NewInvalidUserIDError()
	}

	name := strings.TrimSpace(user.Name)
	email := strings.TrimSpace(user.Email)
	age := user.Age

	if err := validateUserInput(name, email, age); err != nil {
		return User{}, err
	}

	existing, err := s.repo.FindByID(ctx, id)

	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return User{}, NewUserNotFoundError()
		}
		return User{}, fmt.Errorf("update user by id: %d, %w", id, err)
	}

	if existing.Email != user.Email {
		exists, err := s.repo.ExistsByEmail(ctx, user.Email)
		if err != nil {
			return User{}, fmt.Errorf("checked email exists: %w", err)
		}

		if exists {
			return User{}, NewEmailAlreadyExistsError()
		}
	}

	updated, err := s.repo.Update(ctx, User{
		ID:    id,
		Name:  name,
		Email: email,
		Age:   age,
	})

	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return User{}, NewUserNotFoundError()
		}
		return User{}, fmt.Errorf("update user by id %d: %w ", id, err)
	}

	return updated, nil
}

func (s *Service) DeleteUser(ctx context.Context, id int64) error {
	if id <= 0 {
		return NewInvalidUserIDError()
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return NewUserNotFoundError()
		}

		return fmt.Errorf("delete user by id %d: %w", id, err)
	}

	return nil
}

func (s *Service) GetUserByEmail(ctx context.Context, email string) (User, error) {
	email = strings.TrimSpace(email)

	if email == "" {
		return User{}, apperror.NewFieldValidationError(map[string]string{
			"email": "is required",
		})
	}

	user, err := s.repo.FindByEmail(ctx, email)

	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return User{}, NewUserNotFoundError()
		}
		return User{}, fmt.Errorf("get user by email %s: %w", email, err)
	}

	return user, nil
}

func (s *Service) CreateUserWithProfile(ctx context.Context, input CreateUserWithProfileInput) (ProfileWithUser, error) {
	if s.txFactory == nil {
		return ProfileWithUser{}, fmt.Errorf("tx factory is nil")
	}

	if err := validateUserInput(input.Name, input.Email, input.Age); err != nil {
		return ProfileWithUser{}, err
	}

	name := strings.TrimSpace(input.Name)
	email := strings.TrimSpace(strings.ToLower(input.Email))
	age := input.Age
	bio := strings.TrimSpace(input.Bio)

	var res ProfileWithUser

	err := s.txFactory.WithinTx(ctx, func(repo Repository) error {
		exists, err := repo.ExistsByEmail(ctx, email)
		if err != nil {
			return fmt.Errorf("check email exists: %w", err)
		}

		if exists {
			return NewEmailAlreadyExistsError()
		}

		createdUser, err := repo.Create(ctx, User{
			Name:  name,
			Email: email,
			Age:   age,
		})
		if err != nil {
			if errors.Is(err, ErrEmailAlreadyExists) {
				return NewEmailAlreadyExistsError()
			}

			return fmt.Errorf("create user: %w", err)
		}

		createdProfile, err := repo.CreateProfile(ctx, Profile{
			UserID: createdUser.ID,
			Bio:    bio,
		})
		if err != nil {
			return fmt.Errorf("create user profile: %w", err)
		}

		_, err = repo.CreateEvent(ctx, Event{
			UserID:    createdUser.ID,
			EventType: EventTypeUserCreated,
			Payload:   `{"source":"api"}`,
		})
		if err != nil {
			return fmt.Errorf("create user event: %w", err)
		}

		res = ProfileWithUser{
			User:    createdUser,
			Profile: createdProfile,
		}

		return nil
	})
	if err != nil {
		return ProfileWithUser{}, err
	}

	return res, nil
}

func validateUserInput(name string, email string, age int) error {
	fields := make(map[string]string)

	if strings.TrimSpace(name) == "" {
		fields["name"] = "is required"
	} else if len(name) < 2 || len(name) > 100 {
		fields["name"] = "min len 2 and max len 100"
	}

	if strings.TrimSpace(email) == "" {
		fields["email"] = "is required"
	} else if len(email) > 255 {
		fields["email"] = "max email len is 255"
	} else if _, err := mail.ParseAddress(email); err != nil {
		fields["email"] = "not valid email"
	}

	if age < 0 || age > 150 {
		fields["age"] = "must be greater than or equal to 0 and lower than 150"
	}

	if len(fields) > 0 {
		return apperror.NewFieldValidationError(fields)
	}

	return nil
}
