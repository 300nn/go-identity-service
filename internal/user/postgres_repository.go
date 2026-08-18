package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

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
		insert into users (name, email, age, password_hash, role)
		values ($1, $2, $3, $4, $5)
		returning id, name, email, age, role, created_at, updated_at
	`

	var created User

	if user.Role == "" {
		user.Role = RoleUser
	}

	err := r.db.QueryRow(ctx, query, user.Name, user.Email, user.Age, user.PasswordHash, user.Role).Scan(
		&created.ID,
		&created.Name,
		&created.Email,
		&created.Age,
		&created.Role,
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
		select id, name, email, age, role, created_at, updated_at 
		from users
		where id = $1
	`

	var user User

	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Age,
		&user.Role,
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
		select id, name, email, age, role, created_at, updated_at 
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
			&user.Role,
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

func (r *PostgresRepository) List(ctx context.Context, filter ListUsersFilter) (ListUsersResult, error) {
	email := strings.TrimSpace(filter.Email)

	const countQuery = `
		select count(*)
		from users
		where ($1 = '' or email ilike '%' || $1 || '%');
	`
	var total int64

	if err := r.db.QueryRow(ctx, countQuery, email).Scan(&total); err != nil {
		return ListUsersResult{}, fmt.Errorf("count users: %w", err)
	}

	orderBy := usersOrderBy(filter.Sort)

	query := fmt.Sprintf(`
		select id, name, email, age, role, created_at, updated_at 
		from users
		where ($1 = '' or email ILIKE '%%' || $1 || '%%')
		order by %s
		limit $2 offset $3
	`, orderBy)

	rows, err := r.db.Query(ctx, query, email, filter.Limit, filter.Offset)
	if err != nil {
		return ListUsersResult{}, err
	}
	defer rows.Close()

	users := make([]User, 0)

	for rows.Next() {
		var user User
		if err := rows.Scan(
			&user.ID,
			&user.Name,
			&user.Email,
			&user.Age,
			&user.Role,
			&user.CreatedAt,
			&user.UpdatedAt,
		); err != nil {
			return ListUsersResult{}, fmt.Errorf("scan user row: %w", err)
		}

		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return ListUsersResult{}, fmt.Errorf("iterate user rows: %w", err)
	}
	return ListUsersResult{
		Users: users,
		Total: total,
	}, nil
}

func (r *PostgresRepository) Update(ctx context.Context, user User) (User, error) {
	const query = `
		UPDATE users
		SET name = $1,
		    email = $2,
		    age = $3,
		    updated_at = now()
		WHERE id = $4
		RETURNING id, name, email, age, role, created_at, updated_at
	`

	var updated User

	err := r.db.QueryRow(ctx, query, user.Name, user.Email, user.Age, user.ID).Scan(
		&updated.ID,
		&updated.Name,
		&updated.Email,
		&updated.Age,
		&updated.Role,
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
		select id, name, email, age, role, password_hash, created_at, updated_at 
		from users
		where email = $1
	`

	var user User

	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Age,
		&user.Role,
		&user.PasswordHash,
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

func usersOrderBy(sort string) string {
	switch sort {
	case "id_asc":
		return "id ASC"
	case "id_desc":
		return "id DESC"
	case "email_asc":
		return "email ASC"
	case "email_desc":
		return "email DESC"
	case "created_at_asc":
		return "created_at ASC"
	case "created_at_desc":
		return "created_at DESC"
	default:
		return "id ASC"
	}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}

	return false
}
