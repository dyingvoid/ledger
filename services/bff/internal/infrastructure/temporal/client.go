package temporal

import (
	"context"
	"errors"
	"fmt"

	"github.com/dyingvoid/ledger/services/bff/internal/domain"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	temporalclient "go.temporal.io/sdk/client"
)

type Client struct {
	client temporalclient.Client
	cfg    Config
}

func NewClient(cfg Config) (*Client, error) {
	client, err := temporalclient.Dial(temporalclient.Options{
		HostPort:  cfg.HostPort,
		Namespace: cfg.Namespace,
	})
	if err != nil {
		return nil, fmt.Errorf("connecting to temporal at %s: %w", cfg.HostPort, err)
	}
	return &Client{client: client, cfg: cfg}, nil
}

func (c *Client) Close() {
	c.client.Close()
}

func (c *Client) StartAccountOpening(
	ctx context.Context,
	operationID string,
	input domain.AccountOpeningInput,
) (domain.AccountOpeningStatus, error) {
	input.ActivityStartToCloseTimeout = c.cfg.ActivityStartToCloseTimeout
	input.ActivityScheduleToCloseTimeout = c.cfg.ActivityScheduleToCloseTimeout
	input.ActivityMaximumAttempts = c.cfg.ActivityMaximumAttempts

	_, err := c.client.ExecuteWorkflow(ctx, temporalclient.StartWorkflowOptions{
		ID:                                       accountOpeningWorkflowID(operationID),
		TaskQueue:                                c.cfg.TaskQueue,
		WorkflowExecutionTimeout:                 c.cfg.WorkflowExecutionTimeout,
		WorkflowIDConflictPolicy:                 enumspb.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
		WorkflowIDReusePolicy:                    enumspb.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
		WorkflowExecutionErrorWhenAlreadyStarted: false,
	}, AccountOpeningWorkflowName, input)
	if err != nil {
		return domain.AccountOpeningStatusUnknown, fmt.Errorf("executing workflow: %w", err)
	}
	return c.GetAccountOpeningStatus(ctx, operationID)
}

func (c *Client) GetAccountOpeningStatus(
	ctx context.Context,
	operationID string,
) (domain.AccountOpeningStatus, error) {
	description, err := c.client.DescribeWorkflowExecution(
		ctx,
		accountOpeningWorkflowID(operationID),
		"",
	)
	if err != nil {
		if _, isNotFound := errors.AsType[*serviceerror.NotFound](err); isNotFound {
			return domain.AccountOpeningStatusUnknown, domain.ErrAccountOperationNotFound
		}
		return domain.AccountOpeningStatusUnknown, fmt.Errorf("describing workflow: %w", err)
	}

	info := description.GetWorkflowExecutionInfo()
	if info == nil {
		return domain.AccountOpeningStatusUnknown, errors.New("workflow execution info is missing")
	}
	return workflowStatus(info.GetStatus())
}

func workflowStatus(status enumspb.WorkflowExecutionStatus) (domain.AccountOpeningStatus, error) {
	switch status {
	case enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING,
		enumspb.WORKFLOW_EXECUTION_STATUS_PAUSED,
		enumspb.WORKFLOW_EXECUTION_STATUS_CONTINUED_AS_NEW:
		return domain.AccountOpeningStatusInProgress, nil
	case enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED:
		return domain.AccountOpeningStatusSuccess, nil
	case enumspb.WORKFLOW_EXECUTION_STATUS_FAILED,
		enumspb.WORKFLOW_EXECUTION_STATUS_CANCELED,
		enumspb.WORKFLOW_EXECUTION_STATUS_TERMINATED,
		enumspb.WORKFLOW_EXECUTION_STATUS_TIMED_OUT:
		return domain.AccountOpeningStatusFailure, nil
	case enumspb.WORKFLOW_EXECUTION_STATUS_UNSPECIFIED:
		return domain.AccountOpeningStatusUnknown, fmt.Errorf("unexpected workflow status %s", status)
	}
	return domain.AccountOpeningStatusUnknown, fmt.Errorf("unknown workflow status %d", status)
}

func accountOpeningWorkflowID(operationID string) string {
	return "account-opening/" + operationID
}
