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
	ErrInProgress            = errors.New("idempotency request is still in progress")
	ErrUnknownStatus         = errors.New("unknown idempotency status")
	ErrRecordNotFound        = errors.New("idempotency record not found")
)

type Status string

const (
	StatusInProgress Status = "in_progress"
	StatusSuccess    Status = "success"
	StatusFail       Status = "fail"
)

type Record struct {
	Key       string
	Status    Status
	Result    []byte
	Error     string
	ExpiresAt time.Time
}

type Store interface {
	Claim(ctx context.Context, key string, expiresAt time.Time) (bool, error)
	Get(ctx context.Context, key string) (Record, error)
	Complete(ctx context.Context, record Record) error
}

type UnitOfWork interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
