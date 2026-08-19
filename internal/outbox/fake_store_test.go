package outbox_test

import (
	"context"
	"errors"
	"time"

	"CrudTutorialProject/internal/outbox"
)

var (
	errFetchBatchFailed    = errors.New("fetch batch failed")
	errMarkProcessedFailed = errors.New("mark processed failed")
	errMarkFailedFailed    = errors.New("mark failed failed")
)

type fakeStore struct {
	events []outbox.Event

	fetchErr         error
	markProcessedErr error
	markFailedErr    error

	fetchCalls         int
	markProcessedCalls int
	markFailedCalls    int

	processedIDs []int64
	failedIDs    []int64
	lastReason   string
}

func (s *fakeStore) Create(ctx context.Context, event outbox.Event) (outbox.Event, error) {
	return event, nil
}

func (s *fakeStore) FetchBatch(ctx context.Context, limit int, lockTimeout time.Duration) ([]outbox.Event, error) {
	s.fetchCalls++

	if s.fetchErr != nil {
		return nil, s.fetchErr
	}

	if limit > 0 && len(s.events) > limit {
		return s.events[:limit], nil
	}

	return s.events, nil
}

func (s *fakeStore) MarkProcessed(ctx context.Context, id int64) error {
	s.markProcessedCalls++

	if s.markProcessedErr != nil {
		return s.markProcessedErr
	}

	s.processedIDs = append(s.processedIDs, id)
	return nil
}

func (s *fakeStore) MarkFailed(ctx context.Context, id int64, reason string, maxAttempts int) error {
	s.markFailedCalls++

	if s.markFailedErr != nil {
		return s.markFailedErr
	}

	s.failedIDs = append(s.failedIDs, id)
	s.lastReason = reason
	return nil
}
