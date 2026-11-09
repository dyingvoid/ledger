//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	postgrescontainer "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/dyingvoid/ledger/internal/core/idempotency"
	pgpostgres "github.com/dyingvoid/ledger/internal/infrastructure/postgres"
	"github.com/dyingvoid/ledger/internal/infrastructure/postgres/repositories"
)

var (
	testPool    *pgxpool.Pool
	testStore   *repositories.Idempotency
	testService *idempotency.Service
)

func TestMain(m *testing.M) {
	code, err := runSuite(m)
	if err != nil {
		log.Printf("integration suite failed: %v", err)
		os.Exit(1)
	}
	os.Exit(code)
}

func runSuite(m *testing.M) (int, error) {
	ctx := context.Background()

	container, err := postgrescontainer.Run(
		ctx,
		"postgres:17.6-alpine",
		postgrescontainer.WithDatabase("ledger"),
		postgrescontainer.WithUsername("ledger"),
		postgrescontainer.WithPassword("ledger"),
		postgrescontainer.BasicWaitStrategies(),
	)
	if err != nil {
		return 0, fmt.Errorf("starting postgres: %w", err)
	}
	defer func() {
		if err := container.Terminate(context.Background()); err != nil {
			log.Printf("terminating postgres: %v", err)
		}
	}()

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return 0, fmt.Errorf("building connection string: %w", err)
	}
	if err := applyMigrations(ctx, dsn); err != nil {
		return 0, err
	}

	testPool, err = pgxpool.New(ctx, dsn)
	if err != nil {
		return 0, fmt.Errorf("creating pool: %w", err)
	}
	defer testPool.Close()

	testStore = repositories.NewIdempotency(testPool)
	testService = idempotency.NewService(
		testStore,
		pgpostgres.NewUnitOfWork(testPool),
		idempotency.Config{TTL: time.Hour},
	)

	return m.Run(), nil
}

func applyMigrations(ctx context.Context, dsn string) error {
	entries, err := os.ReadDir("../migrations")
	if err != nil {
		return fmt.Errorf("reading migrations: %w", err)
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".up.sql") {
			files = append(files, filepath.Join("..", "migrations", entry.Name()))
		}
	}
	sort.Strings(files)

	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		return fmt.Errorf("parsing postgres config: %w", err)
	}
	cfg.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	conn, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		return fmt.Errorf("connecting for migrations: %w", err)
	}
	defer conn.Close(context.Background())

	for _, file := range files {
		query, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("reading %s: %w", file, err)
		}
		if _, err := conn.Exec(ctx, string(query)); err != nil {
			return fmt.Errorf("applying %s: %w", file, err)
		}
	}

	return nil
}

type accountResult struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

func TestConcurrentClaimWaitsForFirstTransaction(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	key := testKey(t)
	expiresAt := time.Now().Add(time.Hour).UTC()

	tx1, err := testPool.Begin(ctx)
	if err != nil {
		t.Fatalf("beginning first transaction: %v", err)
	}
	defer tx1.Rollback(context.Background())

	txCtx1 := pgpostgres.WithExecutor(ctx, tx1)

	claimed, err := testStore.Claim(txCtx1, key, expiresAt)
	if err != nil {
		t.Fatalf("first Claim() error = %v", err)
	}
	if !claimed {
		t.Fatal("first Claim() = false, want true")
	}

	type secondOutcome struct {
		claimed bool
		found   bool
		record  idempotency.Record
		err     error
	}

	claimStarted := make(chan struct{})
	secondDone := make(chan secondOutcome, 1)

	go func() {
		tx2, err := testPool.Begin(ctx)
		if err != nil {
			secondDone <- secondOutcome{err: err}
			return
		}
		defer tx2.Rollback(context.Background())

		txCtx2 := pgpostgres.WithExecutor(ctx, tx2)

		close(claimStarted)
		claimed, err := testStore.Claim(txCtx2, key, expiresAt)
		if err != nil {
			secondDone <- secondOutcome{err: err}
			return
		}

		record, found, err := testStore.Get(txCtx2, key)
		secondDone <- secondOutcome{claimed: claimed, found: found, record: record, err: err}
	}()

	<-claimStarted

	select {
	case outcome := <-secondDone:
		t.Fatalf("second transaction finished before the first committed: %+v", outcome)
	case <-time.After(500 * time.Millisecond):
	}

	if err := testStore.Complete(txCtx1, idempotency.Record{
		Key:       key,
		Status:    idempotency.StatusSuccess,
		Result:    []byte(`{"id":42,"name":"first"}`),
		ExpiresAt: expiresAt,
	}); err != nil {
		t.Fatalf("first Complete() error = %v", err)
	}
	if err := tx1.Commit(ctx); err != nil {
		t.Fatalf("committing first transaction: %v", err)
	}

	select {
	case outcome := <-secondDone:
		if outcome.err != nil {
			t.Fatalf("second transaction error = %v", outcome.err)
		}
		if outcome.claimed {
			t.Fatal("second Claim() = true, want false")
		}
		if !outcome.found {
			t.Fatal("second Get() found = false, want true")
		}
		if outcome.record.Status != idempotency.StatusSuccess {
			t.Fatalf("second record status = %q, want %q", outcome.record.Status, idempotency.StatusSuccess)
		}
		var got accountResult
		if err := json.Unmarshal(outcome.record.Result, &got); err != nil {
			t.Fatalf("unmarshalling second record result: %v", err)
		}
		if got != (accountResult{ID: 42, Name: "first"}) {
			t.Fatalf("second record result = %+v, want %+v", got, accountResult{ID: 42, Name: "first"})
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for the second transaction")
	}
}

func TestConcurrentClaimSucceedsAfterFirstRollsBack(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	key := testKey(t)
	expiresAt := time.Now().Add(time.Hour).UTC()

	tx1, err := testPool.Begin(ctx)
	if err != nil {
		t.Fatalf("beginning first transaction: %v", err)
	}

	txCtx1 := pgpostgres.WithExecutor(ctx, tx1)
	claimed, err := testStore.Claim(txCtx1, key, expiresAt)
	if err != nil {
		t.Fatalf("first Claim() error = %v", err)
	}
	if !claimed {
		t.Fatal("first Claim() = false, want true")
	}

	type secondOutcome struct {
		claimed bool
		err     error
	}

	claimStarted := make(chan struct{})
	secondDone := make(chan secondOutcome, 1)

	go func() {
		tx2, err := testPool.Begin(ctx)
		if err != nil {
			secondDone <- secondOutcome{err: err}
			return
		}
		defer tx2.Rollback(context.Background())

		txCtx2 := pgpostgres.WithExecutor(ctx, tx2)

		close(claimStarted)
		claimed, err := testStore.Claim(txCtx2, key, expiresAt)
		secondDone <- secondOutcome{claimed: claimed, err: err}
	}()

	<-claimStarted

	select {
	case outcome := <-secondDone:
		t.Fatalf("second transaction finished before the first rolled back: %+v", outcome)
	case <-time.After(500 * time.Millisecond):
	}

	if err := tx1.Rollback(ctx); err != nil {
		t.Fatalf("rolling back first transaction: %v", err)
	}

	select {
	case outcome := <-secondDone:
		if outcome.err != nil {
			t.Fatalf("second transaction error = %v", outcome.err)
		}
		if !outcome.claimed {
			t.Fatal("second Claim() = false, want true after the first rolled back")
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for the second transaction")
	}
}

func TestServiceExecuteReplaysSecondConcurrentRequest(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	key := testKey(t)

	type outcome struct {
		value    accountResult
		replayed bool
		err      error
	}

	var (
		calls           atomic.Int32
		releaseOnce     sync.Once
		businessStarted = make(chan struct{})
		releaseBusiness = make(chan struct{})
		firstDone       = make(chan outcome, 1)
		secondDone      = make(chan outcome, 1)
	)

	release := func() {
		releaseOnce.Do(func() {
			close(releaseBusiness)
		})
	}
	defer release()

	go func() {
		value, replayed, err := testService.Execute(ctx, key, func(context.Context) (accountResult, error) {
			calls.Add(1)
			close(businessStarted)
			<-releaseBusiness
			return accountResult{ID: 42, Name: "first"}, nil
		})
		firstDone <- outcome{value: value, replayed: replayed, err: err}
	}()

	select {
	case <-businessStarted:
	case <-ctx.Done():
		t.Fatal("timed out waiting for the first business function")
	}

	go func() {
		value, replayed, err := testService.Execute(ctx, key, func(context.Context) (accountResult, error) {
			calls.Add(1)
			return accountResult{ID: 99, Name: "second"}, nil
		})
		secondDone <- outcome{value: value, replayed: replayed, err: err}
	}()

	select {
	case second := <-secondDone:
		t.Fatalf("second Execute() finished before the first committed: %+v", second)
	case <-time.After(500 * time.Millisecond):
	}

	release()

	first := <-firstDone
	second := <-secondDone

	if first.err != nil {
		t.Fatalf("first Execute() error = %v", first.err)
	}
	if second.err != nil {
		t.Fatalf("second Execute() error = %v", second.err)
	}
	if first.replayed {
		t.Fatal("first Execute() replayed = true, want false")
	}
	if !second.replayed {
		t.Fatal("second Execute() replayed = false, want true")
	}
	if calls.Load() != 1 {
		t.Fatalf("business calls = %d, want 1", calls.Load())
	}
	if first.value != second.value {
		t.Fatalf("second Execute() = %+v, want %+v", second.value, first.value)
	}
}

func TestClaimReclaimsExpiredRecord(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	key := testKey(t)

	_, err := testPool.Exec(
		ctx,
		`INSERT INTO ledger.idempotency (key, status, result, error, expires_at)
		 VALUES ($1, $2, $3::jsonb, '', $4)`,
		key,
		string(idempotency.StatusSuccess),
		`{"id":1,"name":"expired"}`,
		time.Now().Add(-time.Hour).UTC(),
	)
	if err != nil {
		t.Fatalf("inserting expired record: %v", err)
	}

	claimed, err := testStore.Claim(ctx, key, time.Now().Add(time.Hour).UTC())
	if err != nil {
		t.Fatalf("Claim() error = %v", err)
	}
	if !claimed {
		t.Fatal("Claim() = false, want true for an expired record")
	}

	record, found, err := testStore.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !found {
		t.Fatal("Get() found = false, want true")
	}
	if record.Status != idempotency.StatusInProgress {
		t.Fatalf("record status = %q, want %q", record.Status, idempotency.StatusInProgress)
	}
}

func testKey(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("%s:%d", t.Name(), time.Now().UnixNano())
}
