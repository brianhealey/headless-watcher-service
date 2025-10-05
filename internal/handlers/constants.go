package handlers

import "time"

// Task Flow Module Types
const (
	TFModuleTypeAICamera        = "ai camera"
	TFModuleTypeImageAnalyzer   = "image analyzer"
	TFModuleTypeLocalAlarm      = "local alarm"
	TFModuleTypeSenseCraftAlarm = "sensecraft alarm"
	TFModuleTypeAlarmTrigger    = "alarm trigger"
)

// AI Camera Modes
const (
	TFModuleAICameraModesInference = 0 // Continuous inference mode
)

// AI Camera Detection Modes
const (
	TFModuleAICameraModeAppear = 1 // Appear/disappear detection
)

// AI Camera Detection Types
const (
	TFModuleAICameraTypePreset = 2 // Preset condition type
)

// AI Camera Output Types
const (
	TFModuleAICameraOutputSmall = 0 // Small image only
	TFModuleAICameraOutputBoth  = 1 // Small AND large image (large sent to backend)
)

// AI Camera Shutter Types
const (
	TFModuleAICameraShutterTriggerConstantly = 0 // Continuous triggering
)

// AI Camera Conditions Combo
const (
	TFModuleAICameraConditionsComboAND = 0 // AND combination of conditions
	TFModuleAICameraConditionsComboOR  = 1 // OR combination of conditions
)

// Image Analyzer Types
const (
	TFModuleImgAnalyzerTypeRecognize  = 0 // Recognition mode (just analyze)
	TFModuleImgAnalyzerTypeMonitoring = 1 // Monitoring mode (returns state for alarm triggering)
)

// Model Types for AI Camera
const (
	ModelTypeCloud   = 0 // Cloud model (requires download)
	ModelTypePerson  = 1 // Built-in person detection model
	ModelTypePet     = 2 // Built-in pet detection model (dog, cat)
	ModelTypeGesture = 3 // Built-in gesture detection model (rock, paper, scissors)
)

// Voice Interaction Modes
const (
	VIModeChat     = 0 // Conversational chat mode
	VIModeTask     = 1 // Task execution mode
	VIModeTaskAuto = 2 // Automatic task mode
)

// Default Durations
const (
	DefaultSilenceDuration         = 5 * time.Second  // Silence between AI camera triggers
	DefaultAlarmDuration           = 5 * time.Second  // Local alarm duration
	DefaultNotificationSilence     = 30 * time.Second // Silence between notifications
)

// HTTP Response Codes
const (
	ResponseCodeSuccess = 200
	ResponseCodeUnauthorized = 401
	ResponseCodeNotFound = 404
	ResponseCodeInternalError = 500
)

// Multipart Boundary
const (
	MultipartBoundary = "---sensecraftboundary---"
)

// Audio Format
const (
	AudioFormatWAV = "wav"
)

// COCO Dataset Classes (80 classes supported by default models)
var COCOClasses = []string{
	"person", "bicycle", "car", "motorcycle", "airplane", "bus", "train", "truck", "boat",
	"traffic light", "fire hydrant", "stop sign", "parking meter", "bench",
	"bird", "cat", "dog", "horse", "sheep", "cow", "elephant", "bear", "zebra", "giraffe",
	"backpack", "umbrella", "handbag", "tie", "suitcase", "frisbee", "skis", "snowboard",
	"sports ball", "kite", "baseball bat", "baseball glove", "skateboard", "surfboard",
	"tennis racket", "bottle", "wine glass", "cup", "fork", "knife", "spoon", "bowl",
	"banana", "apple", "sandwich", "orange", "broccoli", "carrot", "hot dog", "pizza",
	"donut", "cake", "chair", "couch", "potted plant", "bed", "dining table", "toilet",
	"tv", "laptop", "mouse", "remote", "keyboard", "cell phone", "microwave", "oven",
	"toaster", "sink", "refrigerator", "book", "clock", "vase", "scissors", "teddy bear",
	"hair drier", "toothbrush",
}
