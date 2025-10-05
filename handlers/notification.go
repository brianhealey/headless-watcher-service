package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/brianhealey/sensecap-server/database"
	"github.com/brianhealey/sensecap-server/models"
)

// NotificationHandler handles /v1/notification/event POST requests
func NotificationHandler(w http.ResponseWriter, r *http.Request) {
	// Read device EUI from header
	deviceEUI := r.Header.Get("API-OBITER-DEVICE-EUI")
	authToken := r.Header.Get("Authorization")

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("ERROR: Failed to read request body: %v", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse JSON request
	var req models.NotificationEventRequest
	if err := json.Unmarshal(body, &req); err != nil {
		log.Printf("ERROR: Failed to parse JSON: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Log the request
	logNotificationRequest(r, deviceEUI, authToken, &req, body)

	// Save event to database
	saveNotificationToDatabase(deviceEUI, &req)

	// Return success response (code must be 200)
	response := models.NotificationResponse{
		Code: 200,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func saveNotificationToDatabase(deviceEUI string, req *models.NotificationEventRequest) {
	// Convert inference and sensor data to JSON strings
	var inferenceJSON, sensorJSON string

	if req.Events.Data != nil {
		if req.Events.Data.Inference != nil {
			if jsonBytes, err := json.Marshal(req.Events.Data.Inference); err == nil {
				inferenceJSON = string(jsonBytes)
			}
		}
		if req.Events.Data.Sensor != nil {
			if jsonBytes, err := json.Marshal(req.Events.Data.Sensor); err == nil {
				sensorJSON = string(jsonBytes)
			}
		}
	}

	// Create notification event
	event := &database.NotificationEvent{
		RequestID:     req.RequestID,
		DeviceEUI:     deviceEUI,
		Timestamp:     getTimestamp(req.Events.Timestamp),
		Text:          getString(req.Events.Text),
		Img:           getString(req.Events.Img),
		InferenceData: inferenceJSON,
		SensorData:    sensorJSON,
	}

	// Save to database
	if err := database.SaveNotificationEvent(event); err != nil {
		log.Printf("WARNING: Failed to save notification event to database: %v", err)
	} else {
		log.Printf("Notification event saved to database: ID=%d", event.ID)
	}
}

func getTimestamp(ts *int64) int64 {
	if ts == nil {
		return 0
	}
	return *ts
}

func getString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func logNotificationRequest(r *http.Request, deviceEUI, authToken string, req *models.NotificationEventRequest, rawBody []byte) {
	log.Println("================================================================================")
	log.Println("NOTIFICATION EVENT RECEIVED")
	log.Println("================================================================================")
	log.Printf("Timestamp:   %s", time.Now().Format(time.RFC3339))
	log.Printf("Action:      %s %s", r.Method, r.URL.Path)
	if r.URL.RawQuery != "" {
		log.Printf("Query:       %s", r.URL.RawQuery)
	}
	log.Printf("Remote Addr: %s", r.RemoteAddr)
	log.Printf("Request ID:  %s", req.RequestID)
	log.Printf("Device EUI:  %s (body)", req.DeviceEUI)

	// Log all headers
	log.Println("--------------------------------------------------------------------------------")
	log.Println("REQUEST HEADERS")
	log.Println("--------------------------------------------------------------------------------")
	for name, values := range r.Header {
		for _, value := range values {
			log.Printf("  %s: %s", name, value)
		}
	}

	// Log event data
	log.Println("--------------------------------------------------------------------------------")
	log.Println("EVENT DATA")
	log.Println("--------------------------------------------------------------------------------")
	if req.Events.Timestamp != nil {
		ts := time.Unix(*req.Events.Timestamp/1000, (*req.Events.Timestamp%1000)*1000000)
		log.Printf("Event Time:  %s (%d ms)", ts.Format(time.RFC3339), *req.Events.Timestamp)
	}

	if req.Events.Text != nil {
		log.Printf("Text:        %s", *req.Events.Text)
	}

	if req.Events.Img != nil {
		imgLen := len(*req.Events.Img)
		log.Printf("Image:       %d bytes (base64)", imgLen)
	}

	// Log inference data
	if req.Events.Data != nil && req.Events.Data.Inference != nil {
		inference := req.Events.Data.Inference
		log.Println("--------------------------------------------------------------------------------")
		log.Println("INFERENCE DATA")
		log.Println("--------------------------------------------------------------------------------")

		// Bounding boxes (object detection)
		if len(inference.Boxes) > 0 {
			log.Printf("Detected %d objects (bounding boxes):", len(inference.Boxes))
			for i, box := range inference.Boxes {
				x, y, w, h, score, target := box[0], box[1], box[2], box[3], box[4], box[5]
				className := "Unknown"
				if target < len(inference.ClassesName) {
					className = inference.ClassesName[target]
				}
				log.Printf("  [%d] %s: confidence=%d%%, bbox=(%d,%d,%d,%d)",
					i, className, score, x, y, w, h)
			}
		}

		// Classifications
		if len(inference.Classes) > 0 {
			log.Printf("Classification results (%d classes):", len(inference.Classes))
			for i, cls := range inference.Classes {
				score, target := cls[0], cls[1]
				className := "Unknown"
				if target < len(inference.ClassesName) {
					className = inference.ClassesName[target]
				}
				log.Printf("  [%d] %s: confidence=%d%%", i, className, score)
			}
		}

		// Class names
		if len(inference.ClassesName) > 0 {
			log.Printf("Available classes: %v", inference.ClassesName)
		}
	}

	// Log sensor data
	if req.Events.Data != nil && req.Events.Data.Sensor != nil {
		sensor := req.Events.Data.Sensor
		log.Println("--------------------------------------------------------------------------------")
		log.Println("SENSOR DATA")
		log.Println("--------------------------------------------------------------------------------")

		if sensor.Temperature != nil {
			log.Printf("Temperature: %.1f°C", *sensor.Temperature)
		}
		if sensor.Humidity != nil {
			log.Printf("Humidity:    %d%%", *sensor.Humidity)
		}
		if sensor.CO2 != nil {
			log.Printf("CO2:         %d ppm", *sensor.CO2)
		}
	}

	// Log raw JSON for debugging
	log.Println("--------------------------------------------------------------------------------")
	log.Println("RAW JSON REQUEST")
	log.Println("--------------------------------------------------------------------------------")

	// Pretty print JSON
	var prettyJSON map[string]interface{}
	if err := json.Unmarshal(rawBody, &prettyJSON); err == nil {
		if formatted, err := json.MarshalIndent(prettyJSON, "", "  "); err == nil {
			fmt.Println(string(formatted))
		}
	}

	log.Println("================================================================================")
	log.Println()
}
