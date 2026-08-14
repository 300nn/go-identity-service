package user_test

import (
	"CrudTutorialProject/internal/user"
	"context"
	"sort"
	"strings"
	"time"
)

type FakeRepository struct {
	nextUserID    int64
	nextProfileID int64
	nextEventID   int64

	users    map[int64]user.User
	profiles map[int64]user.Profile
	events   map[int64]user.Event
}

func newFakeRepository() *FakeRepository {
	return &FakeRepository{
		nextUserID:    1,
		nextProfileID: 1,
		nextEventID:   1,
		users:         make(map[int64]user.User),
		profiles:      make(map[int64]user.Profile),
		events:        make(map[int64]user.Event),
	}
}

func (r *FakeRepository) Create(ctx context.Context, usr user.User) (user.User, error) {
	if exists, _ := r.ExistsByEmail(ctx, usr.Email); exists {
		return user.User{}, user.ErrEmailAlreadyExists
	}

	now := time.Now().UTC()

	usr.ID = r.nextUserID
	usr.CreatedAt = now
	usr.UpdatedAt = now
	if usr.Role == "" {
		usr.Role = user.RoleUser
	}

	r.users[usr.ID] = usr
	r.nextUserID++

	return usr, nil
}

func (r *FakeRepository) FindByID(ctx context.Context, id int64) (user.User, error) {
	found, ok := r.users[id]
	if !ok {
		return user.User{}, user.ErrUserNotFound
	}

	return found, nil
}

func (r *FakeRepository) FindByEmail(ctx context.Context, email string) (user.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	for _, u := range r.users {
		if strings.ToLower(u.Email) == email {
			return u, nil
		}
	}

	return user.User{}, user.ErrUserNotFound
}

func (r *FakeRepository) FindAll(ctx context.Context) ([]user.User, error) {
	users := make([]user.User, 0, len(r.users))

	for _, u := range r.users {
		users = append(users, u)
	}

	sort.Slice(users, func(i, j int) bool {
		return users[i].ID < users[j].ID
	})

	return users, nil
}

func (r *FakeRepository) List(ctx context.Context, filter user.ListUsersFilter) (user.ListUsersResult, error) {
	email := strings.ToLower(strings.TrimSpace(filter.Email))

	users := make([]user.User, 0, len(r.users))

	for _, u := range r.users {
		if email != "" && !strings.Contains(strings.ToLower(u.Email), email) {
			continue
		}

		users = append(users, u)
	}

	sort.Slice(users, func(i, j int) bool {
		switch filter.Sort {
		case "id_desc":
			return users[i].ID > users[j].ID
		case "email_asc":
			return users[i].Email < users[j].Email
		case "email_desc":
			return users[i].Email > users[j].Email
		case "created_at_asc":
			return users[i].CreatedAt.Before(users[j].CreatedAt)
		case "created_at_desc":
			return users[i].CreatedAt.After(users[j].CreatedAt)
		case "id_asc":
			fallthrough
		default:
			return users[i].ID < users[j].ID
		}
	})

	total := int64(len(users))

	start := filter.Offset
	if start > len(users) {
		start = len(users)
	}

	end := start + filter.Limit
	if end > len(users) {
		end = len(users)
	}

	return user.ListUsersResult{
		Users: users[start:end],
		Total: total,
	}, nil
}

func (r *FakeRepository) Update(ctx context.Context, usr user.User) (user.User, error) {
	existing, ok := r.users[usr.ID]
	if !ok {
		return user.User{}, user.ErrUserNotFound
	}

	for _, u := range r.users {
		if u.ID != usr.ID && u.Email == usr.Email {
			return user.User{}, user.ErrEmailAlreadyExists
		}
	}

	usr.CreatedAt = existing.CreatedAt
	usr.UpdatedAt = time.Now().UTC()

	r.users[usr.ID] = usr

	return usr, nil
}

func (r *FakeRepository) Delete(ctx context.Context, id int64) error {
	if _, ok := r.users[id]; !ok {
		return user.ErrUserNotFound
	}

	delete(r.users, id)
	return nil
}

func (r *FakeRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	for _, u := range r.users {
		if strings.ToLower(u.Email) == email {
			return true, nil
		}
	}

	return false, nil
}

func (r *FakeRepository) CreateProfile(ctx context.Context, profile user.Profile) (user.Profile, error) {
	now := time.Now().UTC()

	profile.ID = r.nextProfileID
	profile.CreatedAt = now
	profile.UpdatedAt = now

	r.profiles[profile.ID] = profile
	r.nextProfileID++

	return profile, nil
}

func (r *FakeRepository) CreateEvent(ctx context.Context, event user.Event) (user.Event, error) {
	event.ID = r.nextEventID
	event.CreatedAt = time.Now().UTC()

	r.events[event.ID] = event
	r.nextEventID++

	return event, nil
}
