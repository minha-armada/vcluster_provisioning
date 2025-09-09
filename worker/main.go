package main

import (
	"log"
	"os"

	"task1/workflow"

	"github.com/joho/godotenv"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func main() {
	// Load env vars from .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found (or failed to load)")
	}

	// Check required env vars
	required := []string{"GH_TOKEN", "REPO_OWNER", "REPO_NAME", "VCLUSTER_WORKFLOW_FILE"}
	for _, key := range required {
		if os.Getenv(key) == "" {
			log.Fatalf("Missing required environment variable: %s", key)
		}
	}

	// Temporal client setup
	c, err := client.NewLazyClient(client.Options{})
	if err != nil {
		log.Fatalln("Unable to create client", err)
	}
	defer c.Close()

	w := worker.New(c, "VclusterTaskQueue", worker.Options{})
	w.RegisterWorkflow(workflow.CreateVclusterWorkflow)
	w.RegisterActivity(workflow.CreateVclusterActivity)
	w.RegisterActivity(workflow.OnboardAzureArcActivity)

	err = w.Run(worker.InterruptCh())
	if err != nil {
		log.Fatalln("Unable to start worker", err)
	}
}
