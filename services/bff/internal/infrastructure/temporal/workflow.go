package temporal

import (
	"github.com/dyingvoid/ledger/services/bff/internal/domain"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	AccountOpeningWorkflowName    = "AccountOpeningWorkflow"
	CreateAccountInfoActivityName = "CreateAccountInfo"
	OpenLedgerAccountActivityName = "OpenLedgerAccount"
)

func AccountOpeningWorkflow(ctx workflow.Context, input domain.AccountOpeningInput) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout:    input.ActivityStartToCloseTimeout,
		ScheduleToCloseTimeout: input.ActivityScheduleToCloseTimeout,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: input.ActivityMaximumAttempts,
		},
	})

	if err := workflow.ExecuteActivity(
		ctx,
		CreateAccountInfoActivityName,
		input,
	).Get(ctx, nil); err != nil {
		return err
	}

	return workflow.ExecuteActivity(
		ctx,
		OpenLedgerAccountActivityName,
		input.AccountingID,
	).Get(ctx, nil)
}
