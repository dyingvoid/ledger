package temporal

import (
	"testing"

	"github.com/dyingvoid/ledger/services/bff/internal/domain"
	enumspb "go.temporal.io/api/enums/v1"
)

func TestWorkflowStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status enumspb.WorkflowExecutionStatus
		want   domain.AccountOpeningStatus
	}{
		{status: enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING, want: domain.AccountOpeningStatusInProgress},
		{status: enumspb.WORKFLOW_EXECUTION_STATUS_PAUSED, want: domain.AccountOpeningStatusInProgress},
		{status: enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED, want: domain.AccountOpeningStatusSuccess},
		{status: enumspb.WORKFLOW_EXECUTION_STATUS_FAILED, want: domain.AccountOpeningStatusFailure},
		{status: enumspb.WORKFLOW_EXECUTION_STATUS_CANCELED, want: domain.AccountOpeningStatusFailure},
		{status: enumspb.WORKFLOW_EXECUTION_STATUS_TERMINATED, want: domain.AccountOpeningStatusFailure},
		{status: enumspb.WORKFLOW_EXECUTION_STATUS_TIMED_OUT, want: domain.AccountOpeningStatusFailure},
	}

	for _, tt := range tests {
		got, err := workflowStatus(tt.status)
		if err != nil {
			t.Fatalf("workflowStatus(%s) error = %v", tt.status, err)
		}
		if got != tt.want {
			t.Fatalf("workflowStatus(%s) = %q, want %q", tt.status, got, tt.want)
		}
	}
}
