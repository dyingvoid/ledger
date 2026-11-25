package temporal

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/dyingvoid/ledger/services/bff/internal/domain"
	"go.temporal.io/sdk/testsuite"
)

type fakeAccountGateway struct {
	mu           sync.Mutex
	createCalls  int
	openCalls    int
	accountingID string
	createErr    error
	openErr      error
}

func (f *fakeAccountGateway) CreateAccountInfo(
	_ context.Context,
	accountingID string,
	_ domain.CreateAccountParams,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.createCalls++
	f.accountingID = accountingID
	return f.createErr
}

func (f *fakeAccountGateway) OpenLedgerAccount(
	_ context.Context,
	accountingID string,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.openCalls++
	f.accountingID = accountingID
	return f.openErr
}

type fakeActivities struct {
	gateway *fakeAccountGateway
}

func (f *fakeActivities) CreateAccountInfo(
	ctx context.Context,
	input domain.AccountOpeningInput,
) error {
	return f.gateway.CreateAccountInfo(ctx, input.AccountingID, domain.CreateAccountParams{})
}

func (f *fakeActivities) OpenLedgerAccount(ctx context.Context, accountingID string) error {
	return f.gateway.OpenLedgerAccount(ctx, accountingID)
}

func TestAccountOpeningWorkflow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		createErr  error
		openErr    error
		wantErr    bool
		wantCreate int
		wantOpen   int
	}{
		{
			name:       "success",
			wantCreate: 1,
			wantOpen:   1,
		},
		{
			name:       "create account info failure",
			createErr:  errors.New("create failed"),
			wantErr:    true,
			wantCreate: 1,
		},
		{
			name:       "open ledger account failure",
			openErr:    errors.New("open failed"),
			wantErr:    true,
			wantCreate: 1,
			wantOpen:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gateway := &fakeAccountGateway{createErr: tt.createErr, openErr: tt.openErr}
			activities := &fakeActivities{gateway: gateway}
			env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()
			env.RegisterWorkflow(AccountOpeningWorkflow)
			env.RegisterActivity(activities)

			env.ExecuteWorkflow(AccountOpeningWorkflowName, domain.AccountOpeningInput{
				AccountingID:                   "123456789",
				UserID:                         42,
				Name:                           "Main account",
				Type:                           2100,
				Currency:                       "USD",
				ActivityStartToCloseTimeout:    time.Second,
				ActivityScheduleToCloseTimeout: time.Minute,
				ActivityMaximumAttempts:        1,
			})

			if got := env.GetWorkflowError(); (got != nil) != tt.wantErr {
				t.Fatalf("workflow error = %v, want error %t", got, tt.wantErr)
			}
			if gateway.createCalls != tt.wantCreate {
				t.Fatalf("create calls = %d, want %d", gateway.createCalls, tt.wantCreate)
			}
			if gateway.openCalls != tt.wantOpen {
				t.Fatalf("open calls = %d, want %d", gateway.openCalls, tt.wantOpen)
			}
			if gateway.createCalls > 0 && gateway.accountingID != "123456789" {
				t.Fatalf("accounting id = %q, want 123456789", gateway.accountingID)
			}
		})
	}
}
