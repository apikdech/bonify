package workflow

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// FXSyncWorkflow orchestrates the FX rate synchronization process
func FXSyncWorkflow(ctx workflow.Context) error {
	// Define activity options with retry policy
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    2 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumAttempts:    3,
		},
	}

	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	// Fetch FX rates from Frankfurter API and store in DB
	if err := workflow.ExecuteActivity(ctx, "FetchFXRatesActivity").Get(ctx, nil); err != nil {
		return err
	}

	return nil
}
