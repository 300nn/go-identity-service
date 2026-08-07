package user

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{
		pool: pool,
	}
}

func (r *PostgresRepository) Create(ctx context.Context, user User) (User, error) {
	const query = `
		insert into users (name, email, age)
		values ($1, $2, $3)
		returning id, name, email, age, created_at, updated_at
	`

	var created User

	err := r.pool.QueryRow(ctx, query, user.Name, user.Email, user.Age).Scan(
		&created.ID,
		&created.Name,
		&created.Email,
		&created.Age,
		&created.CreatedAt,
		&created.UpdatedAt,
	)

	if err != nil {
		if isUniqueViolation(err) {
			return User{}, ErrEmailAlreadyExists
		}

		return User{}, fmt.Errorf("insert user: %w", err)
	}

	return created, nil
}
func (r *PostgresRepository) FindByID(ctx context.Context, id int64) (User, error) {
	const query = `
		select id, name, email, age, created_at, updated_at 
		from users
		where id = $1
	`

	var user User

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Age,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrUserNotFound
		}

		return User{}, fmt.Errorf("select user by id: %w", err)
	}

	return user, nil
}

func (r *PostgresRepository) FindAll(ctx context.Context) ([]User, error) {
	const query = `
		select id, name, email, age, created_at, updated_at 
		from users
		order by id
	`

	rows, err := r.pool.Query(ctx, query)

	if err != nil {
		return nil, fmt.Errorf("select all users: %w", err)
	}

	defer rows.Close()

	users := make([]User, 0)

	var user User

	for rows.Next() {
		err := rows.Scan(
			&user.ID,
			&user.Name,
			&user.Email,
			&user.Age,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan user row: %w", err)
		}

		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user rows: %w", err)
	}

	return users, nil
}
func (r *PostgresRepository) Update(ctx context.Context, user User) (User, error) {
	const query = `
		UPDATE users
		SET name = $1,
		    email = $2,
		    age = $3,
		    updated_at = now()
		WHERE id = $4
		RETURNING id, name, email, age, created_at, updated_at
	`

	var updated User

	err := r.pool.QueryRow(ctx, query, user.Name, user.Email, user.Age, user.ID).Scan(
		&updated.ID,
		&updated.Name,
		&updated.Email,
		&updated.Age,
		&updated.CreatedAt,
		&updated.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrUserNotFound
		}

		if isUniqueViolation(err) {
			return User{}, ErrEmailAlreadyExists
		}

		return User{}, fmt.Errorf("update user: %w", err)
	}

	return updated, nil
}

func (r *PostgresRepository) FindByEmail(ctx context.Context, email string) (User, error) {
	const query = `
		select id, name, email, age, created_at, updated_at 
		from users
		where email = $1
	`

	var user User

	err := r.pool.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Age,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrUserNotFound
		}

		return User{}, fmt.Errorf("select user by email: %w", err)
	}

	return user, nil
}
func (r *PostgresRepository) Delete(ctx context.Context, id int64) error {
	const query = `
		delete from users where id = $1
	`
	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	return nil
}

func (r *PostgresRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1
			FROM users
			WHERE email = $1
		)
	`

	var exists bool

	if err := r.pool.QueryRow(ctx, query, email).Scan(&exists); err != nil {
		return false, fmt.Errorf("check user exists by email: %w", err)
	}

	return exists, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}

	return false
}
