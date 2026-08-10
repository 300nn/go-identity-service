package user

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type DBTX interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type PostgresRepository struct {
	db DBTX
}

func NewPostgresRepository(db DBTX) *PostgresRepository {
	return &PostgresRepository{
		db: db,
	}
}

func (r *PostgresRepository) Create(ctx context.Context, user User) (User, error) {
	const query = `
		insert into users (name, email, age)
		values ($1, $2, $3)
		returning id, name, email, age, created_at, updated_at
	`

	var created User

	err := r.db.QueryRow(ctx, query, user.Name, user.Email, user.Age).Scan(
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

	err := r.db.QueryRow(ctx, query, id).Scan(
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

	rows, err := r.db.Query(ctx, query)

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

	err := r.db.QueryRow(ctx, query, user.Name, user.Email, user.Age, user.ID).Scan(
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

	err := r.db.QueryRow(ctx, query, email).Scan(
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
	tag, err := r.db.Exec(ctx, query, id)
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

	if err := r.db.QueryRow(ctx, query, email).Scan(&exists); err != nil {
		return false, fmt.Errorf("check user exists by email: %w", err)
	}

	return exists, nil
}

func (r *PostgresRepository) CreateProfile(ctx context.Context, profile Profile) (Profile, error) {
	const query = `
		INSERT INTO user_profiles (user_id, bio)
		VALUES ($1, $2)
		RETURNING id, user_id, bio, created_at, updated_at
	`

	var created Profile

	err := r.db.QueryRow(ctx, query, profile.UserID, profile.Bio).Scan(
		&created.ID,
		&created.UserID,
		&created.Bio,
		&created.CreatedAt,
		&created.UpdatedAt,
	)
	if err != nil {
		return Profile{}, fmt.Errorf("insert user profile: %w", err)
	}

	return created, nil
}

func (r *PostgresRepository) CreateEvent(ctx context.Context, event Event) (Event, error) {
	const query = `
		INSERT INTO user_events (user_id, event_type, payload)
		VALUES ($1, $2, $3::jsonb)
		RETURNING id, user_id, event_type, payload::text, created_at
	`

	var created Event

	err := r.db.QueryRow(ctx, query, event.UserID, event.EventType, event.Payload).Scan(
		&created.ID,
		&created.UserID,
		&created.EventType,
		&created.Payload,
		&created.CreatedAt,
	)
	if err != nil {
		return Event{}, fmt.Errorf("insert user event: %w", err)
	}

	return created, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}

	return false
}
