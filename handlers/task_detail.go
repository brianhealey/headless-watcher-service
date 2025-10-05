package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

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

	// Build response with data.tl.task_flow format that firmware expects
	var response map[string]interface{}
	if len(taskFlows) > 0 {
		// Convert to Node-RED style task flow
		taskFlowData := convertToNodeREDFormat(taskFlows[0])

		response = map[string]interface{}{
			"code": 200,
			"data": map[string]interface{}{
				"tl": taskFlowData, // Contains type, tlid, ctd, tn, task_flow fields
			},
		}
	} else {
		// No tasks - return empty response to stop task flow
		response = map[string]interface{}{
			"code": 200,
			"data": map[string]interface{}{
				"tl": map[string]interface{}{},
			},
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// selectModelType determines which local model to use based on target object
func selectModelType(targetObject string) int {
	// Normalize to lowercase for comparison
	obj := strings.ToLower(targetObject)

	// Model type 1: Person detection
	if obj == "person" {
		return 1
	}

	// Model type 2: Pet/Animal detection (dog, cat)
	if obj == "dog" || obj == "cat" {
		return 2
	}

	// Model type 3: Gesture detection (rock, paper, scissors)
	if obj == "rock" || obj == "paper" || obj == "scissors" {
		return 3
	}

	// Model type 0: Cloud model (download required) for everything else
	log.Printf("WARNING: Target object '%s' not supported by local models, falling back to cloud model", targetObject)
	return 0
}

// convertToNodeREDFormat converts our simple TaskFlow to the firmware's Node-RED style format
func convertToNodeREDFormat(task *database.TaskFlow) map[string]interface{} {
	// Use task ID as tlid and created timestamp as ctd
	tlid := task.ID
	ctd := task.CreatedAt.UnixMilli()

	// Use the LLM-selected model type stored in database
	modelType := task.ModelType
	log.Printf("Using stored model type: %d for task '%s'", modelType, task.Headline)

	// Node 1: AI camera with detection conditions
	aiCameraNode := map[string]interface{}{
		"id":    1,
		"type":  "ai camera",
		"index": 0,
		"params": map[string]interface{}{
			"modes":      0,         // TF_MODULE_AI_CAMERA_MODES_INFERENCE
			"model_type": modelType, // 0=cloud, 1=person, 2=pet, 3=gesture
			"conditions": []map[string]interface{}{
				{
					"class": task.TargetObjects[0],
					"mode":  1,   // 1 = appear/disappear detection
					"type":  2,   // Type 2 from preset
					"num":   0,
				},
			},
			"conditions_combo": 0, // AND
			"silent_period": map[string]interface{}{
				"silence_duration": 5, // 5 seconds between triggers
			},
			"output_type": 1, // 1 = small img AND large img (large img sent to backend for LLaVA)
			"shutter":     0, // TF_MODULE_AI_CAMERA_SHUTTER_TRIGGER_CONSTANTLY
		},
		"wires": [][]int{{2}}, // Connect to node 2 (alarm trigger)
	}

	// Node 2: Image analyzer - sends large image to LLaVA for verification
	imageAnalyzerNode := map[string]interface{}{
		"id":    2,
		"type":  "image analyzer",
		"index": 1,
		"params": map[string]interface{}{
			"body": map[string]interface{}{
				"prompt":    task.TriggerCondition, // LLM prompt for verification
				"type":      1,                      // TF_MODULE_IMG_ANALYZER_TYPE_MONITORING (returns state for alarm triggering)
				"audio_txt": "",                     // No audio text
			},
		},
		"wires": [][]int{{3, 4}}, // Connect to both local alarm (3) and sensecraft alarm (4)
	}

	// Node 3: Local alarm - beep/LED/display on device
	localAlarmNode := map[string]interface{}{
		"id":    3,
		"type":  "local alarm",
		"index": 2,
		"params": map[string]interface{}{
			"sound":    1,  // Enable sound
			"rgb":      1,  // Enable RGB LED
			"img":      0,  // Don't show image
			"text":     0,  // Don't show text
			"duration": 5,  // 5 seconds
		},
		"wires": [][]int{}, // Terminal node
	}

	// Node 4: SenseCraft alarm - sends HTTP notification to our server
	sensecraftAlarmNode := map[string]interface{}{
		"id":    4,
		"type":  "sensecraft alarm",
		"index": 3,
		"params": map[string]interface{}{
			"silence_duration": 30, // 30 seconds between notifications
		},
		"wires": [][]int{}, // Terminal node
	}

	// Build complete task flow structure
	taskFlowData := map[string]interface{}{
		"type":      0,          // Task flow type
		"tlid":      tlid,       // Task list ID
		"ctd":       ctd,        // Created date timestamp
		"tn":        task.Headline, // Task name
		"task_flow": []map[string]interface{}{
			aiCameraNode,
			imageAnalyzerNode,
			localAlarmNode,
			sensecraftAlarmNode,
		},
	}

	return taskFlowData
}
