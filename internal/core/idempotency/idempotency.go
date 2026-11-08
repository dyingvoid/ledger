package idempotency

import (
	"context"
	"errors"
	"time"
)

const MaxKeyLength = 255

var (
	ErrKeyRequired           = errors.New("idempotency key is required")
	ErrKeyTooLong            = errors.New("idempotency key is too long")
	ErrPreviousAttemptFailed = errors.New("previous attempt with this idempotency key failed")
	ErrUnknownStatus         = errors.New("unknown idempotency status")
)

type Status string

const (
	StatusSuccess Status = "success"
	StatusFail    Status = "fail"
)

type Record struct {
	Key       string
	Status    Status
	Result    []byte
	Error     string
	ExpiresAt time.Time
}

type Store interface {
	Lock(ctx context.Context, key string) error
	Get(ctx context.Context, key string) (Record, bool, error)
	Save(ctx context.Context, record Record) error
}

type UnitOfWork interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
