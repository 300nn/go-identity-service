package user

import (
	"context"
	"sync"
	"time"
)

type MemoryRepository struct {
	mu     sync.RWMutex
	nextId int64
	users  map[int64]User
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		nextId: 1,
		users:  make(map[int64]User),
	}
}

func (r *MemoryRepository) Create(ctx context.Context, user User) (User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()

	user.ID = r.nextId
	user.CreatedAt = now
	user.UpdatedAt = now

	r.users[user.ID] = user
	r.nextId++

	return user, nil
}

func (r *MemoryRepository) FindByID(ctx context.Context, id int64) (User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, ok := r.users[id]

	if !ok {
		return User{}, ErrUserNotFound
	}

	return user, nil
}

func (r *MemoryRepository) FindAll(ctx context.Context) ([]User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	users := make([]User, 0, len(r.users))

	for _, u := range r.users {
		users = append(users, u)
	}

	return users, nil
}

func (r *MemoryRepository) Update(ctx context.Context, user User) (User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	existingUser, ok := r.users[user.ID]

	if !ok {
		return User{}, ErrUserNotFound
	}

	existingUser.Name = user.Name
	existingUser.Email = user.Email
	existingUser.Age = user.Age
	existingUser.UpdatedAt = time.Now().UTC()

	r.users[user.ID] = existingUser

	return existingUser, nil
}

func (r *MemoryRepository) Delete(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.users[id]; !ok {
		return ErrUserNotFound
	}

	delete(r.users, id)

	return nil
}

func (r *MemoryRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, user := range r.users {
		if user.Email == email {
			return true, nil
		}
	}

	return false, nil
}

func (r *MemoryRepository) FindByEmail(ctx context.Context, email string) (User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, val := range r.users {
		if val.Email == email {
			return val, nil
		}
	}

	return User{}, ErrUserNotFound
}
