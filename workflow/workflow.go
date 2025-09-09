package workflow

import (
	"time"

	"go.temporal.io/sdk/workflow"
)

type VclusterInput struct {
	VclusterName  string
	Namespace     string
	CPU           string
	Memory        string
	Storage       string
	WorkflowID    string
	SignalName    string
	SignalPayload string
}

func CreateVclusterWorkflow(ctx workflow.Context, input VclusterInput) (string, error) {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	logger := workflow.GetLogger(ctx)

	//vcluster create
	if err := workflow.ExecuteActivity(ctx, CreateVclusterActivity, input).Get(ctx, nil); err != nil {
		logger.Error("CreateVclusterActivity failed", "error", err)
		return "", err
	}

	// wait for Github Signal
	signalChan := workflow.GetSignalChannel(ctx, "vcluster-created")
	timeout := workflow.NewTimer(ctx, 15*time.Minute)
	logger.Info("Waiting for GitHub Action to signal vcluster creation complete")

	var signalPayload string
	received := false

	selector := workflow.NewSelector(ctx)
	selector.AddReceive(signalChan, func(c workflow.ReceiveChannel, more bool) {
		c.Receive(ctx, &signalPayload)
		received = true
	})
	selector.AddFuture(timeout, func(f workflow.Future) {
		logger.Error("Timeout waiting for vCluster signal from GitHub")
		// You could set a flag or return here if you want to fail the workflow on timeout.
	})

	for !received {
		selector.Select(ctx)
	}

	if !received {
		return "", workflow.NewContinueAsNewError(ctx, CreateVclusterWorkflow, input) // or return error
	}

	logger.Info("Received signal from GitHub:", "payload", signalPayload)

	//trigger azure arc onboarding
	if err := workflow.ExecuteActivity(ctx, OnboardAzureArcActivity, input).Get(ctx, nil); err != nil {
		logger.Error("OnboardAzureArcActivity failed", "error", err)
		return "", err
	}

	return "vCluster creation and Azure Arc onboarding completed", nil
}
