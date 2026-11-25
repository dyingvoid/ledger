package temporal

import (
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

type Worker struct {
	worker worker.Worker
}

func NewWorker(client *Client, activities *Activities) *Worker {
	worker := worker.New(client.client, client.cfg.TaskQueue, worker.Options{})
	worker.RegisterWorkflowWithOptions(
		AccountOpeningWorkflow,
		workflow.RegisterOptions{Name: AccountOpeningWorkflowName},
	)
	worker.RegisterActivityWithOptions(
		activities.CreateAccountInfo,
		activity.RegisterOptions{Name: CreateAccountInfoActivityName},
	)
	worker.RegisterActivityWithOptions(
		activities.OpenLedgerAccount,
		activity.RegisterOptions{Name: OpenLedgerAccountActivityName},
	)
	return &Worker{worker: worker}
}

func (w *Worker) Start() error {
	return w.worker.Start()
}

func (w *Worker) Stop() {
	w.worker.Stop()
}
