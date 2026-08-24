package user_test

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"CrudTutorialProject/internal/auth"
	"CrudTutorialProject/internal/eventcodec"
	"CrudTutorialProject/internal/outbox"
	"CrudTutorialProject/internal/testutils"
	"CrudTutorialProject/internal/user"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

func countRows(t *testing.T, ctx context.Context, pool *pgxpool.Pool, query string, args ...any) int64 {
	t.Helper()

	var count int64
	if err := pool.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		t.Fatalf("count rows: %v", err)
	}

	return count
}

func TestPostgresTxRepositoryFactory_WithinTx_CommitsUserAndOutbox(t *testing.T) {
	ctx := t.Context()
	pool := testutils.NewTestPostgresPool(t)
	txFactory := user.NewPostgresTxRepositoryFactory(pool)

	var createdID int64

	err := txFactory.WithinTx(ctx, func(stores user.TxStores) error {
		created, err := stores.UserRepo.Create(ctx, user.User{
			Name:         "Alex",
			Email:        "alex.tx.commit@example.com",
			Age:          25,
			PasswordHash: "password-hash",
		})
		if err != nil {
			return err
		}

		createdID = created.ID

		_, err = stores.OutBoxStore.Create(ctx, outbox.Event{
			EventType:     outbox.EventTypeUserRegistered,
			AggregateType: outbox.AggregateUser,
			AggregateID:   strconv.FormatInt(created.ID, 10),
			Payload:       []byte(`{"source":"test"}`),
			ContentType:   outbox.ContentTypeJSON,
			EventVersion:  eventcodec.EventVersionV1,
		})
		return err
	})
	if err != nil {
		t.Fatalf("WithinTx returned error: %v", err)
	}

	repo := user.NewPostgresRepository(pool)
	found, err := repo.FindByID(ctx, createdID)
	if err != nil {
		t.Fatalf("FindByID after commit returned error: %v", err)
	}
	if found.Email != "alex.tx.commit@example.com" {
		t.Fatalf("expected email %q, got %q", "alex.tx.commit@example.com", found.Email)
	}

	outboxCount := countRows(
		t,
		ctx,
		pool,
		`SELECT count(*) FROM outbox_events WHERE aggregate_type = $1 AND aggregate_id = $2`,
		outbox.AggregateUser,
		strconv.FormatInt(createdID, 10),
	)
	if outboxCount != 1 {
		t.Fatalf("expected outbox events count 1, got %d", outboxCount)
	}
}

func TestPostgresTxRepositoryFactory_WithinTx_RollsBackUserAndOutbox(t *testing.T) {
	ctx := t.Context()
	pool := testutils.NewTestPostgresPool(t)
	txFactory := user.NewPostgresTxRepositoryFactory(pool)

	expectedErr := errors.New("force rollback")

	err := txFactory.WithinTx(ctx, func(stores user.TxStores) error {
		created, err := stores.UserRepo.Create(ctx, user.User{
			Name:         "Rollback",
			Email:        "rollback@example.com",
			Age:          25,
			PasswordHash: "password-hash",
		})
		if err != nil {
			return err
		}

		_, err = stores.OutBoxStore.Create(ctx, outbox.Event{
			EventType:     outbox.EventTypeUserRegistered,
			AggregateType: outbox.AggregateUser,
			AggregateID:   strconv.FormatInt(created.ID, 10),
			Payload:       []byte(`{"source":"test"}`),
			ContentType:   outbox.ContentTypeJSON,
			EventVersion:  eventcodec.EventVersionV1,
		})
		if err != nil {
			return err
		}

		return expectedErr
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected rollback error %v, got %v", expectedErr, err)
	}

	repo := user.NewPostgresRepository(pool)
	_, err = repo.FindByEmail(ctx, "rollback@example.com")
	if err == nil {
		t.Fatal("expected user not to be persisted after rollback")
	}
	if !errors.Is(err, user.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound after rollback, got %v", err)
	}

	outboxCount := countRows(t, ctx, pool, `SELECT count(*) FROM outbox_events`)
	if outboxCount != 0 {
		t.Fatalf("expected outbox events to be rolled back, got %d", outboxCount)
	}
}

func TestService_CreateUser_CommitsUserAndOutbox(t *testing.T) {
	ctx := t.Context()
	pool := testutils.NewTestPostgresPool(t)

	repo := user.NewPostgresRepository(pool)
	txFactory := user.NewPostgresTxRepositoryFactory(pool)
	hasher := auth.NewPasswordHasherWithCost(bcrypt.MinCost)
	service := user.NewService(repo, txFactory, hasher)

	created, err := service.CreateUser(ctx, user.CreateUserInput{
		Name:     "Alex",
		Email:    "alex.create@example.com",
		Age:      25,
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}

	usersCount := countRows(t, ctx, pool, `SELECT count(*) FROM users WHERE id = $1`, created.ID)
	if usersCount != 1 {
		t.Fatalf("expected users count 1, got %d", usersCount)
	}

	assertPostgresUserRegisteredOutboxEvent(t, ctx, pool, created)
}

func TestService_CreateUserWithProfile_CommitsUserProfileAndOutbox(t *testing.T) {
	ctx := t.Context()
	pool := testutils.NewTestPostgresPool(t)

	repo := user.NewPostgresRepository(pool)
	txFactory := user.NewPostgresTxRepositoryFactory(pool)
	hasher := auth.NewPasswordHasherWithCost(bcrypt.MinCost)
	service := user.NewService(repo, txFactory, hasher)

	created, err := service.CreateUserWithProfile(ctx, user.CreateUserWithProfileInput{
		Name:     "Alex",
		Email:    "alex.with.profile@example.com",
		Age:      25,
		Password: "password123",
		Bio:      "Go backend developer",
	})
	if err != nil {
		t.Fatalf("CreateUserWithProfile returned error: %v", err)
	}

	if created.User.ID == 0 {
		t.Fatal("expected user ID to be set")
	}
	if created.Profile.ID == 0 {
		t.Fatal("expected profile ID to be set")
	}
	if created.Profile.UserID != created.User.ID {
		t.Fatalf("expected profile user_id %d, got %d", created.User.ID, created.Profile.UserID)
	}

	usersCount := countRows(t, ctx, pool, `SELECT count(*) FROM users WHERE id = $1`, created.User.ID)
	if usersCount != 1 {
		t.Fatalf("expected users count 1, got %d", usersCount)
	}

	profilesCount := countRows(t, ctx, pool, `SELECT count(*) FROM user_profiles WHERE user_id = $1`, created.User.ID)
	if profilesCount != 1 {
		t.Fatalf("expected profiles count 1, got %d", profilesCount)
	}

	assertPostgresUserRegisteredOutboxEvent(t, ctx, pool, created.User)
}

func TestService_CreateUserWithProfile_DuplicateEmail_DoesNotCreateSecondOutboxEvent(t *testing.T) {
	ctx := t.Context()
	pool := testutils.NewTestPostgresPool(t)

	repo := user.NewPostgresRepository(pool)
	txFactory := user.NewPostgresTxRepositoryFactory(pool)
	hasher := auth.NewPasswordHasherWithCost(bcrypt.MinCost)
	service := user.NewService(repo, txFactory, hasher)

	first, err := service.CreateUserWithProfile(ctx, user.CreateUserWithProfileInput{
		Name:     "Alex",
		Email:    "duplicate.profile@example.com",
		Age:      25,
		Password: "password123",
		Bio:      "First profile",
	})
	if err != nil {
		t.Fatalf("first CreateUserWithProfile returned error: %v", err)
	}

	_, err = service.CreateUserWithProfile(ctx, user.CreateUserWithProfileInput{
		Name:     "Another Alex",
		Email:    "duplicate.profile@example.com",
		Age:      30,
		Password: "password456",
		Bio:      "Second profile",
	})
	if err == nil {
		t.Fatal("expected duplicate email error, got nil")
	}
	if !errors.Is(err, user.ErrEmailAlreadyExists) {
		t.Fatalf("expected ErrEmailAlreadyExists, got %v", err)
	}

	usersCount := countRows(t, ctx, pool, `SELECT count(*) FROM users WHERE email = $1`, "duplicate.profile@example.com")
	if usersCount != 1 {
		t.Fatalf("expected users count 1, got %d", usersCount)
	}

	profilesCount := countRows(
		t,
		ctx,
		pool,
		`SELECT count(*) FROM user_profiles WHERE user_id = $1`,
		first.User.ID,
	)
	if profilesCount != 1 {
		t.Fatalf("expected profiles count 1, got %d", profilesCount)
	}

	outboxCount := countRows(
		t,
		ctx,
		pool,
		`SELECT count(*) FROM outbox_events WHERE aggregate_type = $1 AND aggregate_id = $2`,
		outbox.AggregateUser,
		strconv.FormatInt(first.User.ID, 10),
	)
	if outboxCount != 1 {
		t.Fatalf("expected exactly 1 outbox event, got %d", outboxCount)
	}
}

func assertPostgresUserRegisteredOutboxEvent(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	usr user.User,
) {
	t.Helper()

	var (
		eventType     string
		aggregateType string
		aggregateID   string
		payload       []byte
		contentType   string
		protoMessage  string
		eventVersion  string
		status        outbox.Status
	)

	err := pool.QueryRow(
		ctx,
		`SELECT event_type, aggregate_type, aggregate_id, payload_bytes, content_type, proto_message, event_version, status
		 FROM outbox_events
		 WHERE aggregate_type = $1 AND aggregate_id = $2`,
		outbox.AggregateUser,
		strconv.FormatInt(usr.ID, 10),
	).Scan(
		&eventType,
		&aggregateType,
		&aggregateID,
		&payload,
		&contentType,
		&protoMessage,
		&eventVersion,
		&status,
	)
	if err != nil {
		t.Fatalf("read outbox event: %v", err)
	}

	if eventType != outbox.EventTypeUserRegistered {
		t.Fatalf("expected event type %q, got %q", outbox.EventTypeUserRegistered, eventType)
	}
	if aggregateType != outbox.AggregateUser {
		t.Fatalf("expected aggregate type %q, got %q", outbox.AggregateUser, aggregateType)
	}
	if aggregateID != strconv.FormatInt(usr.ID, 10) {
		t.Fatalf("expected aggregate id %q, got %q", strconv.FormatInt(usr.ID, 10), aggregateID)
	}
	if contentType != eventcodec.ContentTypeProtobuf {
		t.Fatalf("expected content type %q, got %q", eventcodec.ContentTypeProtobuf, contentType)
	}
	if protoMessage != eventcodec.ProtoMessageUserRegistered {
		t.Fatalf("expected proto message %q, got %q", eventcodec.ProtoMessageUserRegistered, protoMessage)
	}
	if eventVersion != eventcodec.EventVersionV1 {
		t.Fatalf("expected event version %q, got %q", eventcodec.EventVersionV1, eventVersion)
	}
	if status != outbox.StatusNew {
		t.Fatalf("expected status %q, got %q", outbox.StatusNew, status)
	}

	decoded, err := eventcodec.UnmarshalUserRegistered(payload)
	if err != nil {
		t.Fatalf("UnmarshalUserRegistered returned error: %v", err)
	}
	if decoded.GetUserId() != usr.ID {
		t.Fatalf("expected payload user id %d, got %d", usr.ID, decoded.GetUserId())
	}
	if decoded.GetEmail() != usr.Email {
		t.Fatalf("expected payload email %q, got %q", usr.Email, decoded.GetEmail())
	}
	if decoded.GetRole() != string(usr.Role) {
		t.Fatalf("expected payload role %q, got %q", usr.Role, decoded.GetRole())
	}
}
