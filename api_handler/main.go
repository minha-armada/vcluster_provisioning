package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"task1/workflow"

	"go.temporal.io/sdk/client"
)

var temporalClient client.Client

func main() {
	var err error
	temporalClient, err = client.NewLazyClient(client.Options{})
	if err != nil {
		log.Fatalf("Unable to create Temporal client: %v", err)
	}
	defer temporalClient.Close()

	http.HandleFunc("/", serveForm)
	http.HandleFunc("/submit", handleSubmit)
	http.HandleFunc("/trigger-signal", handleSignal)

	log.Println("Server started at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))

}

// serveForm serves the HTML form located in ./ui/ui.html
func serveForm(w http.ResponseWriter, r *http.Request) {
	// Load the form HTML file
	tmplPath := filepath.Join("ui", "ui.html")

	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		http.Error(w, "Error loading form", http.StatusInternalServerError)
		log.Printf("Template parsing error: %v", err)
		return
	}

	err = tmpl.Execute(w, nil)
	if err != nil {
		http.Error(w, "Error rendering form", http.StatusInternalServerError)
		log.Printf("Template execution error: %v", err)
	}
}

// handleSubmit handles POST requests to trigger Temporal workflow
func handleSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	vclusterName := r.FormValue("vclusterName")
	cpu := r.FormValue("cpu")
	memory := r.FormValue("memory")
	storage := r.FormValue("storage")

	if vclusterName == "" || cpu == "" || memory == "" || storage == "" {
		http.Error(w, "Missing required parameters", http.StatusBadRequest)
		return
	}

	namespace := fmt.Sprintf("%s-ns", vclusterName)
	workflowID := fmt.Sprintf("vcluster-workflow-%s", vclusterName)

	workflowInput := workflow.VclusterInput{
		VclusterName: vclusterName,
		Namespace:    namespace,
		CPU:          cpu,
		Memory:       memory,
		Storage:      storage,
		WorkflowID:   workflowID,
	}

	workflowOptions := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: "VclusterTaskQueue",
	}

	we, err := temporalClient.ExecuteWorkflow(context.Background(), workflowOptions, workflow.CreateVclusterWorkflow, workflowInput)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to start workflow: %v", err), http.StatusInternalServerError)
		return
	}

	// Response shown after submission
	fmt.Fprintf(w, "Workflow started successfully! RunID: %s", we.GetRunID())
}

func handleSignal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		WorkflowID  string `json:"workflowId"`
		SignalName  string `json:"signalName"`
		SignalInput string `json:"signalInput"`
	}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Failed to decode JSON", http.StatusBadRequest)
		return
	}

	err = temporalClient.SignalWorkflow(context.Background(), req.WorkflowID, "", req.SignalName, req.SignalInput)
	if err != nil {
		http.Error(w, "Failed to send signal to workflow: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Signal sent to workflow successfully"))
}
