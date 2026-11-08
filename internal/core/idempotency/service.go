package idempotency

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Service struct {
	store Store
	uow   UnitOfWork
	ttl   time.Duration
}

func NewService(store Store, uow UnitOfWork, cfg Config) *Service {
	ttl := cfg.TTL
	if ttl <= 0 {
		ttl = DefaultTTL
	}
	return &Service{store: store, uow: uow, ttl: ttl}
}

func (s *Service) Execute(
	ctx context.Context,
	key string,
	fn func(ctx context.Context) ([]byte, error),
) ([]byte, bool, error) {
	if err := validateKey(key); err != nil {
		return nil, false, err
	}

	var (
		result      []byte
		replayed    bool
		businessErr error
	)

	err := s.uow.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := s.store.Lock(txCtx, key); err != nil {
			return fmt.Errorf("locking idempotency key: %w", err)
		}

		record, found, err := s.store.Get(txCtx, key)
		if err != nil {
			return fmt.Errorf("reading idempotency record: %w", err)
		}
		if found {
			switch record.Status {
			case StatusSuccess:
				result = record.Result
				replayed = true
				return nil
			case StatusFail:
				return fmt.Errorf("%w: %s", ErrPreviousAttemptFailed, record.Error)
			default:
				return fmt.Errorf("%w: %q", ErrUnknownStatus, record.Status)
			}
		}

		businessErr = s.uow.WithinTransaction(txCtx, func(businessCtx context.Context) error {
			value, err := fn(businessCtx)
			if err != nil {
				return err
			}
			result = value
			return nil
		})
		if businessErr != nil {
			failed := Record{
				Key:       key,
				Status:    StatusFail,
				Error:     businessErr.Error(),
				ExpiresAt: s.expiresAt(),
			}
			if err := s.store.Save(txCtx, failed); err != nil {
				return errors.Join(businessErr, fmt.Errorf("saving failed idempotency record: %w", err))
			}
			return nil
		}

		succeeded := Record{
			Key:       key,
			Status:    StatusSuccess,
			Result:    result,
			ExpiresAt: s.expiresAt(),
		}
		if err := s.store.Save(txCtx, succeeded); err != nil {
			return fmt.Errorf("saving idempotency record: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	if businessErr != nil {
		return nil, false, businessErr
	}

	return result, replayed, nil
}

func (s *Service) expiresAt() time.Time {
	return time.Now().Add(s.ttl)
}

func validateKey(key string) error {
	if strings.TrimSpace(key) == "" {
		return ErrKeyRequired
	}
	if len(key) > MaxKeyLength {
		return fmt.Errorf("%w: %d > %d", ErrKeyTooLong, len(key), MaxKeyLength)
	}
	return nil
}
