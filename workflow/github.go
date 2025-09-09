package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"go.temporal.io/sdk/activity"
)

func triggerGithubAction(ctx context.Context, actionName string, input VclusterInput) error {
	logger := activity.GetLogger(ctx)

	token := os.Getenv("GH_TOKEN")
	repoOwner := os.Getenv("REPO_OWNER")
	repoName := os.Getenv("REPO_NAME")

	// Add a new environment variable for the Azure Arc workflow file
	var workflowFile string
	switch actionName {
	case "create":
		workflowFile = os.Getenv("VCLUSTER_WORKFLOW_FILE")
	case "arc-integration":
		workflowFile = os.Getenv("ARC_WORKFLOW_FILE") // Use a new env var
	}

	if token == "" || repoOwner == "" || repoName == "" || workflowFile == "" {
		return fmt.Errorf("missing required environment variables")
	}

	apiURL := fmt.Sprintf(
		"https://api.github.com/repos/%s/%s/actions/workflows/%s/dispatches",
		repoOwner, repoName, workflowFile,
	)

	inputs := map[string]string{
		"cluster_name":   input.VclusterName,
		"namespace_name": fmt.Sprintf("%s-ns", input.VclusterName),
		"cpu":            input.CPU,
		"memory":         input.Memory,
		"storage":        input.Storage,
		"workflow_id":    input.WorkflowID,
		"signal_name":    "vcluster-created",
		"signal_payload": "done",
	}

	switch actionName {
	case "create":
		inputs = map[string]string{
			"cluster_name":   input.VclusterName,
			"namespace_name": fmt.Sprintf("%s-ns", input.VclusterName),
			"cpu":            input.CPU,
			"memory":         input.Memory,
			"storage":        input.Storage,
			"workflow_id":    input.WorkflowID,
			"signal_name":    "vcluster-created",
			"signal_payload": "done",
		}
	case "arc-integration":
		inputs = map[string]string{
			"cluster_name":   input.VclusterName, // Corrected key name
			"namespace_name": fmt.Sprintf("vcluster-%s-ns", input.VclusterName),
		}
	}

	payloadObj := map[string]interface{}{
		"ref":    "main",
		"inputs": inputs,
	}

	payloadBytes, err := json.Marshal(payloadObj)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	logger.Info("Triggering GitHub Action", "url", apiURL, "payload", string(payloadBytes))

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(string(payloadBytes)))
	if err != nil {
		return fmt.Errorf("failed to create GitHub request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("GitHub API call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("GitHub API error: %s", resp.Status)
	}

	logger.Info("GitHub Action triggered", "action", actionName)
	return nil
}
