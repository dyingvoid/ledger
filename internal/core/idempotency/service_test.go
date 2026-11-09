package idempotency

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type fakeStore struct {
	records       map[string]Record
	claimCalls    int
	completeCalls int
	claimErr      error
	getErr        error
	completeErr   error
}

func newFakeStore() *fakeStore {
	return &fakeStore{records: make(map[string]Record)}
}

func (f *fakeStore) Claim(_ context.Context, key string, expiresAt time.Time) (bool, error) {
	f.claimCalls++
	if f.claimErr != nil {
		return false, f.claimErr
	}
	if record, ok := f.records[key]; ok && record.ExpiresAt.After(time.Now()) {
		return false, nil
	}
	f.records[key] = Record{
		Key:       key,
		Status:    StatusInProgress,
		ExpiresAt: expiresAt,
	}
	return true, nil
}

func (f *fakeStore) Get(_ context.Context, key string) (Record, bool, error) {
	if f.getErr != nil {
		return Record{}, false, f.getErr
	}
	record, ok := f.records[key]
	if !ok || !record.ExpiresAt.After(time.Now()) {
		return Record{}, false, nil
	}
	return record, true, nil
}

func (f *fakeStore) Complete(_ context.Context, record Record) error {
	f.completeCalls++
	if f.completeErr != nil {
		return f.completeErr
	}
	f.records[record.Key] = record
	return nil
}

type fakeUnitOfWork struct {
	calls int
}

func (f *fakeUnitOfWork) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	f.calls++
	return fn(ctx)
}

func newTestService(t *testing.T, store Store, uow UnitOfWork) *Service {
	t.Helper()
	return NewService(store, uow, Config{TTL: time.Hour})
}

func TestExecuteSuccess(t *testing.T) {
	t.Parallel()

	store := newFakeStore()
	uow := &fakeUnitOfWork{}
	service := newTestService(t, store, uow)

	var calls int
	result, replayed, err := service.execute(
		context.Background(),
		"create_account:1",
		func(context.Context) ([]byte, error) {
			calls++
			return []byte(`{"id":1}`), nil
		},
	)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if replayed {
		t.Fatal("Execute() replayed = true, want false")
	}
	if calls != 1 {
		t.Fatalf("business calls = %d, want 1", calls)
	}
	if string(result) != `{"id":1}` {
		t.Fatalf("result = %s, want %s", result, `{"id":1}`)
	}
	if uow.calls != 2 {
		t.Fatalf("WithinTransaction() calls = %d, want 2", uow.calls)
	}

	record, ok := store.records["create_account:1"]
	if !ok {
		t.Fatal("success record was not saved")
	}
	if record.Status != StatusSuccess {
		t.Fatalf("record status = %q, want %q", record.Status, StatusSuccess)
	}
	if record.ExpiresAt.Before(time.Now()) {
		t.Fatalf("record expires_at = %v, want in the future", record.ExpiresAt)
	}
}

func TestExecuteReplaysSuccess(t *testing.T) {
	t.Parallel()

	store := newFakeStore()
	store.records["deposit:7"] = Record{
		Key:       "deposit:7",
		Status:    StatusSuccess,
		Result:    []byte(`{"journal_entry_id":9}`),
		ExpiresAt: time.Now().Add(time.Hour),
	}
	uow := &fakeUnitOfWork{}
	service := newTestService(t, store, uow)

	result, replayed, err := service.execute(
		context.Background(),
		"deposit:7",
		func(context.Context) ([]byte, error) {
			t.Fatal("business logic must not run for a replayed success")
			return nil, nil
		},
	)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !replayed {
		t.Fatal("Execute() replayed = false, want true")
	}
	if string(result) != `{"journal_entry_id":9}` {
		t.Fatalf("result = %s, want %s", result, `{"journal_entry_id":9}`)
	}
	if store.completeCalls != 0 {
		t.Fatalf("Complete() calls = %d, want 0", store.completeCalls)
	}
}

func TestExecuteReplaysFailure(t *testing.T) {
	t.Parallel()

	store := newFakeStore()
	store.records["deposit:8"] = Record{
		Key:       "deposit:8",
		Status:    StatusFail,
		Error:     "insufficient funds",
		ExpiresAt: time.Now().Add(time.Hour),
	}
	service := newTestService(t, store, &fakeUnitOfWork{})

	_, _, err := service.execute(
		context.Background(),
		"deposit:8",
		func(context.Context) ([]byte, error) {
			t.Fatal("business logic must not run for a replayed failure")
			return nil, nil
		},
	)
	if !errors.Is(err, ErrPreviousAttemptFailed) {
		t.Fatalf("Execute() error = %v, want %v", err, ErrPreviousAttemptFailed)
	}
	if !strings.Contains(err.Error(), "insufficient funds") {
		t.Fatalf("Execute() error = %v, want stored failure message", err)
	}
}

func TestExecuteRecordsFailure(t *testing.T) {
	t.Parallel()

	store := newFakeStore()
	service := newTestService(t, store, &fakeUnitOfWork{})
	businessErr := errors.New("insufficient funds")

	result, replayed, err := service.execute(
		context.Background(),
		"deposit:9",
		func(context.Context) ([]byte, error) {
			return nil, businessErr
		},
	)
	if !errors.Is(err, businessErr) {
		t.Fatalf("Execute() error = %v, want %v", err, businessErr)
	}
	if result != nil {
		t.Fatalf("result = %s, want nil", result)
	}
	if replayed {
		t.Fatal("Execute() replayed = true, want false")
	}

	record, ok := store.records["deposit:9"]
	if !ok {
		t.Fatal("failure record was not saved")
	}
	if record.Status != StatusFail {
		t.Fatalf("record status = %q, want %q", record.Status, StatusFail)
	}
	if record.Error != businessErr.Error() {
		t.Fatalf("record error = %q, want %q", record.Error, businessErr.Error())
	}
}

func TestExecuteReclaimsExpiredRecord(t *testing.T) {
	t.Parallel()

	store := newFakeStore()
	store.records["create_account:2"] = Record{
		Key:       "create_account:2",
		Status:    StatusSuccess,
		Result:    []byte(`{"id":1}`),
		ExpiresAt: time.Now().Add(-time.Hour),
	}
	service := newTestService(t, store, &fakeUnitOfWork{})

	var calls int
	result, replayed, err := service.execute(
		context.Background(),
		"create_account:2",
		func(context.Context) ([]byte, error) {
			calls++
			return []byte(`{"id":2}`), nil
		},
	)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if replayed {
		t.Fatal("Execute() replayed = true, want false")
	}
	if calls != 1 {
		t.Fatalf("business calls = %d, want 1", calls)
	}
	if string(result) != `{"id":2}` {
		t.Fatalf("result = %s, want %s", result, `{"id":2}`)
	}
}

func TestExecuteValidatesKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		key  string
		want error
	}{
		{name: "empty", key: "", want: ErrKeyRequired},
		{name: "whitespace", key: "   ", want: ErrKeyRequired},
		{name: "too long", key: strings.Repeat("a", MaxKeyLength+1), want: ErrKeyTooLong},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			store := newFakeStore()
			service := newTestService(t, store, &fakeUnitOfWork{})

			_, _, err := service.execute(
				context.Background(),
				test.key,
				func(context.Context) ([]byte, error) {
					t.Fatal("business logic must not run for an invalid key")
					return nil, nil
				},
			)
			if !errors.Is(err, test.want) {
				t.Fatalf("Execute() error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestExecuteReturnsClaimError(t *testing.T) {
	t.Parallel()

	claimErr := errors.New("database unavailable")
	store := newFakeStore()
	store.claimErr = claimErr
	service := newTestService(t, store, &fakeUnitOfWork{})

	_, _, err := service.execute(
		context.Background(),
		"create_account:3",
		func(context.Context) ([]byte, error) {
			t.Fatal("business logic must not run when claiming fails")
			return nil, nil
		},
	)
	if !errors.Is(err, claimErr) {
		t.Fatalf("Execute() error = %v, want %v", err, claimErr)
	}
}

func TestExecuteReturnsCompleteError(t *testing.T) {
	t.Parallel()

	completeErr := errors.New("database unavailable")
	store := newFakeStore()
	store.completeErr = completeErr
	service := newTestService(t, store, &fakeUnitOfWork{})

	_, _, err := service.execute(
		context.Background(),
		"create_account:3",
		func(context.Context) ([]byte, error) {
			return []byte(`{"id":3}`), nil
		},
	)
	if !errors.Is(err, completeErr) {
		t.Fatalf("Execute() error = %v, want %v", err, completeErr)
	}
}

func TestExecuteRoundTripsValue(t *testing.T) {
	t.Parallel()

	type result struct {
		ID   int64
		Name string
	}

	store := newFakeStore()
	service := newTestService(t, store, &fakeUnitOfWork{})

	first, replayed, err := service.Execute(
		context.Background(),
		"create_account:4",
		func(context.Context) (result, error) {
			return result{ID: 4, Name: "Daily"}, nil
		},
	)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if replayed {
		t.Fatal("Execute() replayed = true, want false")
	}
	if first.ID != 4 || first.Name != "Daily" {
		t.Fatalf("Execute() = %+v, want %+v", first, result{ID: 4, Name: "Daily"})
	}

	second, replayed, err := service.Execute(
		context.Background(),
		"create_account:4",
		func(context.Context) (result, error) {
			t.Fatal("business logic must not run for a replayed success")
			return result{}, nil
		},
	)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !replayed {
		t.Fatal("Execute() replayed = false, want true")
	}
	if second != first {
		t.Fatalf("Execute() = %+v, want %+v", second, first)
	}
}
