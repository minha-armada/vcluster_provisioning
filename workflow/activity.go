package workflow

import (
	"context"
)

func CreateVclusterActivity(ctx context.Context, input VclusterInput) error {
	return triggerGithubAction(ctx, "create", input)
}

func OnboardAzureArcActivity(ctx context.Context, input VclusterInput) error {
	return triggerGithubAction(ctx, "arc-integration", input)
}
