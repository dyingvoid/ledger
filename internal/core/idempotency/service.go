package idempotency

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dyingvoid/ledger/internal/logger"
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

func (s *Service) Execute[T any](
	ctx context.Context,
	key string,
	fn func(ctx context.Context) (T, error),
) (T, bool, error) {
	var zero T

	raw, replayed, err := s.execute(ctx, key, func(ctx context.Context) ([]byte, error) {
		value, err := fn(ctx)
		if err != nil {
			return nil, err
		}
		data, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("marshalling idempotent result: %w", err)
		}
		return data, nil
	})
	if err != nil {
		return zero, replayed, err
	}

	var value T
	if err := json.Unmarshal(raw, &value); err != nil {
		return zero, replayed, fmt.Errorf("unmarshalling idempotent result: %w", err)
	}

	if replayed {
		logger.FromContext(ctx).InfoContext(ctx, "idempotent request replayed", "idempotency_key", key)
	}

	return value, replayed, nil
}

func (s *Service) execute(
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
		existing, err := s.store.Get(txCtx, key)
		switch {
		case err == nil:
			result, replayed, err = replay(existing)
			return err
		case errors.Is(err, ErrRecordNotFound):
		default:
			return fmt.Errorf("reading idempotency record: %w", err)
		}

		claimed, err := s.store.Claim(txCtx, key, s.expiresAt())
		if err != nil {
			return fmt.Errorf("claiming idempotency key: %w", err)
		}
		if !claimed {
			existing, err := s.store.Get(txCtx, key)
			if err != nil {
				if errors.Is(err, ErrRecordNotFound) {
					return fmt.Errorf("%w: %q", ErrRecordNotFound, key)
				}
				return fmt.Errorf("reading idempotency record: %w", err)
			}
			result, replayed, err = replay(existing)
			return err
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
			if err := s.store.Complete(txCtx, failed); err != nil {
				return errors.Join(businessErr, fmt.Errorf("completing failed idempotency record: %w", err))
			}
			return nil
		}

		succeeded := Record{
			Key:       key,
			Status:    StatusSuccess,
			Result:    result,
			ExpiresAt: s.expiresAt(),
		}
		if err := s.store.Complete(txCtx, succeeded); err != nil {
			return fmt.Errorf("completing idempotency record: %w", err)
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

func replay(record Record) ([]byte, bool, error) {
	switch record.Status {
	case StatusSuccess:
		return record.Result, true, nil
	case StatusFail:
		return nil, false, fmt.Errorf("%w: %s", ErrPreviousAttemptFailed, record.Error)
	default:
		return nil, false, fmt.Errorf("%w: %q", ErrUnknownStatus, record.Status)
	}
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
