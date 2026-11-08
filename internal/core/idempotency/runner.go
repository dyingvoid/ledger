package idempotency

import (
	"context"
	"encoding/json"
	"fmt"
)

type Runner[T any] struct {
	service *Service
}

func NewRunner[T any](service *Service) *Runner[T] {
	return &Runner[T]{service: service}
}

func (r *Runner[T]) Do(
	ctx context.Context,
	key string,
	fn func(ctx context.Context) (T, error),
) (T, bool, error) {
	var zero T

	raw, replayed, err := r.service.Execute(ctx, key, func(ctx context.Context) ([]byte, error) {
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

	return value, replayed, nil
}
