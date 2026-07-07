package user

import "context"

type Repository interface {
	Create(ctx context.Context, user User) (User, error)
	FindByID(ctx context.Context, id int64) (User, error)
	FindAll(ctx context.Context) ([]User, error)
	Update(ctx context.Context, user User) (User, error)
	Delete(ctx context.Context, id int64) error
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	FindByEmail(ctx context.Context, email string) (User, error)
}
