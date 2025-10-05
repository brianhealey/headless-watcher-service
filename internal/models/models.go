package models

// NotificationEventRequest represents the alarm/notification event from the device
type NotificationEventRequest struct {
	RequestID string `json:"requestId"`
	DeviceEUI string `json:"deviceEui"`
	Events    Events `json:"events"`
}

// Events contains the event data
type Events struct {
	Timestamp *int64         `json:"timestamp,omitempty"` // Unix timestamp in milliseconds
	Text      *string        `json:"text,omitempty"`
	Img       *string        `json:"img,omitempty"` // Base64-encoded JPEG
	Data      *EventData     `json:"data,omitempty"`
}

// EventData contains inference and sensor data
type EventData struct {
	Inference *InferenceData `json:"inference,omitempty"`
	Sensor    *SensorData    `json:"sensor,omitempty"`
}

// InferenceData contains AI inference results
type InferenceData struct {
	Boxes       []BoundingBox     `json:"boxes,omitempty"`       // Object detection results
	Classes     []Classification  `json:"classes,omitempty"`     // Classification results
	ClassesName []string          `json:"classes_name,omitempty"` // Class names indexed by ID
}

// BoundingBox represents object detection box
// Format: [x, y, width, height, confidence_score, class_id]
type BoundingBox [6]int

// Classification represents classification result
// Format: [confidence_score, class_id]
// Note: C struct has target first, but JSON puts score first
type Classification [2]int

// SensorData contains sensor readings
type SensorData struct {
	Temperature *float64 `json:"temperature,omitempty"` // Celsius
	Humidity    *int     `json:"humidity,omitempty"`    // Percentage (0-100)
	CO2         *int     `json:"CO2,omitempty"`         // PPM
}

// NotificationResponse is the response for notification endpoint
type NotificationResponse struct {
	Code int `json:"code"`
}

// ImageAnalyzerRequest represents the image analysis request
type ImageAnalyzerRequest struct {
	Img      string `json:"img"`       // Base64-encoded JPEG (large resolution)
	Prompt   string `json:"prompt"`    // AI prompt/instruction
	AudioTxt string `json:"audio_txt"` // Audio transcription text
	Type     int    `json:"type"`      // Analysis type: 0=RECOGNIZE, 1=MONITORING
}

// ImageAnalyzerResponse is the response for image analyzer endpoint
type ImageAnalyzerResponse struct {
	Code int                      `json:"code"`
	Data ImageAnalyzerResponseData `json:"data"`
}

// ImageAnalyzerResponseData contains the analysis results
type ImageAnalyzerResponseData struct {
	State int     `json:"state"`          // 0=no event, 1=event detected
	Type  int     `json:"type"`           // Echo of request type or updated type
	Audio *string `json:"audio,omitempty"` // Base64-encoded audio response (optional)
	Img   *string `json:"img,omitempty"`   // Base64-encoded image (optional)
}
