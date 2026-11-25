package grpc

import (
	"testing"
	"time"

	"github.com/dyingvoid/ledger/services/accounting/internal/domain"
)

func TestToProtoAccountInfo(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, time.November, 25, 12, 0, 0, 0, time.UTC)
	id, err := domain.ParseID("42")
	if err != nil {
		t.Fatalf("ParseID() error = %v", err)
	}

	account := toProtoAccountInfo(domain.AccountInfo{
		ID:          id,
		UserID:      7,
		Name:        "Daily",
		Description: "Spending",
		Type:        2100,
		Currency:    "USD",
		CreatedAt:   createdAt,
	})

	if account.GetId() != id.String() {
		t.Fatalf("ID = %q, want %q", account.GetId(), id)
	}
	if account.GetUserId() != 7 {
		t.Fatalf("UserID = %d, want 7", account.GetUserId())
	}
	if account.GetName() != "Daily" {
		t.Fatalf("Name = %q, want %q", account.GetName(), "Daily")
	}
	if account.GetDescription() != "Spending" {
		t.Fatalf("Description = %q, want %q", account.GetDescription(), "Spending")
	}
	if account.GetType() != 2100 {
		t.Fatalf("Type = %d, want 2100", account.GetType())
	}
	if account.GetCurrency() != "USD" {
		t.Fatalf("Currency = %q, want USD", account.GetCurrency())
	}
	if !account.GetCreatedAt().AsTime().Equal(createdAt) {
		t.Fatalf("CreatedAt = %s, want %s", account.GetCreatedAt().AsTime(), createdAt)
	}
}

func TestToProtoAccountLedger(t *testing.T) {
	t.Parallel()

	id, err := domain.ParseID("42")
	if err != nil {
		t.Fatalf("ParseID() error = %v", err)
	}

	account := toProtoAccountLedger(domain.AccountLedger{
		ID:     id,
		UserID: 7,
		Ledger: 1,
		Type:   2100,
	})

	if account.GetId() != id.String() {
		t.Fatalf("ID = %q, want %q", account.GetId(), id)
	}
	if account.GetUserId() != 7 {
		t.Fatalf("UserID = %d, want 7", account.GetUserId())
	}
	if account.GetLedger() != 1 {
		t.Fatalf("Ledger = %d, want 1", account.GetLedger())
	}
	if account.GetType() != 2100 {
		t.Fatalf("Type = %d, want 2100", account.GetType())
	}
}
