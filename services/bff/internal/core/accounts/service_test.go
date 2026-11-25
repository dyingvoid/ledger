package accounts

import (
	"context"
	"errors"
	"testing"

	"github.com/dyingvoid/ledger/services/bff/internal/domain"
)

type fakeWorkflow struct {
	operationID string
	input       domain.AccountOpeningInput
	status      domain.AccountOpeningStatus
	err         error
}

func (f *fakeWorkflow) StartAccountOpening(
	_ context.Context,
	operationID string,
	input domain.AccountOpeningInput,
) (domain.AccountOpeningStatus, error) {
	f.operationID = operationID
	f.input = input
	return f.status, f.err
}

func (f *fakeWorkflow) GetAccountOpeningStatus(
	_ context.Context,
	operationID string,
) (domain.AccountOpeningStatus, error) {
	f.operationID = operationID
	return f.status, f.err
}

type fakeIDGenerator struct {
	id string
}

func (f *fakeIDGenerator) NewID() string {
	return f.id
}

func TestCreateAccount(t *testing.T) {
	t.Parallel()

	workflow := &fakeWorkflow{status: domain.AccountOpeningStatusInProgress}
	service := NewService(workflow, &fakeIDGenerator{id: "123456789"})

	operation, err := service.CreateAccount(
		context.Background(),
		"create-main-account",
		domain.CreateAccountParams{
			UserID:      42,
			Name:        " Main account ",
			Description: " Client deposits ",
			Type:        2100,
			Currency:    " USD ",
		},
	)
	if err != nil {
		t.Fatalf("CreateAccount() error = %v", err)
	}
	if operation.ID != domain.NewAccountOperationID("create-main-account") {
		t.Fatalf("operation id = %q", operation.ID)
	}
	if workflow.input.AccountingID != "123456789" {
		t.Fatalf("accounting id = %q, want 123456789", workflow.input.AccountingID)
	}
	if workflow.input.Name != "Main account" {
		t.Fatalf("account name = %q, want Main account", workflow.input.Name)
	}
	if workflow.input.Currency != "USD" {
		t.Fatalf("account currency = %q, want USD", workflow.input.Currency)
	}
}

func TestCreateAccountRequiresIdempotencyKey(t *testing.T) {
	t.Parallel()

	service := NewService(
		&fakeWorkflow{status: domain.AccountOpeningStatusInProgress},
		&fakeIDGenerator{id: "123456789"},
	)
	_, err := service.CreateAccount(context.Background(), " ", domain.CreateAccountParams{
		UserID:   42,
		Name:     "Main account",
		Type:     2100,
		Currency: "USD",
	})
	if !errors.Is(err, domain.ErrIdempotencyKeyRequired) {
		t.Fatalf("CreateAccount() error = %v, want %v", err, domain.ErrIdempotencyKeyRequired)
	}
}

func TestGetAccountOperation(t *testing.T) {
	t.Parallel()

	operationID := domain.NewAccountOperationID("create-main-account")
	workflow := &fakeWorkflow{status: domain.AccountOpeningStatusSuccess}
	service := NewService(workflow, &fakeIDGenerator{})

	operation, err := service.GetAccountOperation(context.Background(), operationID)
	if err != nil {
		t.Fatalf("GetAccountOperation() error = %v", err)
	}
	if operation.Status != domain.AccountOpeningStatusSuccess {
		t.Fatalf("status = %q, want %q", operation.Status, domain.AccountOpeningStatusSuccess)
	}
}
