package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/brianhealey/sensecap-server/database"
)

// TaskDetailHandler handles /v2/watcher/talk/view_task_detail POST requests
func TaskDetailHandler(w http.ResponseWriter, r *http.Request) {
	// Read device EUI from header
	deviceEUI := r.Header.Get("API-OBITER-DEVICE-EUI")

	log.Printf("Task detail request from device: %s", deviceEUI)

	// Get all task flows for this device
	taskFlows, err := database.GetTaskFlowsByDevice(deviceEUI)
	if err != nil {
		log.Printf("ERROR: Failed to retrieve task flows: %v", err)
		http.Error(w, "Failed to retrieve task flows", http.StatusInternalServerError)
		return
	}

	log.Printf("Found %d task flows for device %s", len(taskFlows), deviceEUI)

	// Build response
	response := map[string]interface{}{
		"code": 200,
		"data": map[string]interface{}{
			"tasks": taskFlows,
			"count": len(taskFlows),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
