// In a new system_monitor package/service
package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"time"
)

func main() {
	http.HandleFunc("/", dashboardHandler)
	http.HandleFunc("/api/nodes", nodesAPIHandler)
	http.HandleFunc("/api/storage", storageAPIHandler)
	http.HandleFunc("/api/tasks", tasksAPIHandler)

	fmt.Println("System monitor starting on :8500")
	http.ListenAndServe(":8500", nil)
}

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	// Serve an HTML dashboard
	tmpl := template.Must(template.ParseFiles("dashboard.html"))
	tmpl.Execute(w, nil)
}

func nodesAPIHandler(w http.ResponseWriter, r *http.Request) {
	// Query the discovery service for all nodes
	resp, err := http.Get("https://localhost:8000/nodes")
	if err != nil {
		http.Error(w, "Failed to get nodes", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Forward the response
	w.Header().Set("Content-Type", "application/json")
	json.NewDecoder(resp.Body).Decode(json.NewEncoder(w))
}

func storageAPIHandler(w http.ResponseWriter, r *http.Request) {
	// Query for storage capacity info
	resp, err := http.Get("https://localhost:8000/capacity")
	if err != nil {
		http.Error(w, "Failed to get storage info", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Forward the response
	w.Header().Set("Content-Type", "application/json")
	json.NewDecoder(resp.Body).Decode(json.NewEncoder(w))
}

func tasksAPIHandler(w http.ResponseWriter, r *http.Request) {
	// Would need to implement a task tracking API in construction nodes
	// For now, just return example data
	tasks := []map[string]interface{}{
		{
			"task_id":   "example-1",
			"status":    "completed",
			"started":   time.Now().Add(-10 * time.Minute),
			"completed": time.Now().Add(-5 * time.Minute),
		},
		{
			"task_id": "example-2",
			"status":  "processing",
			"started": time.Now().Add(-2 * time.Minute),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}
