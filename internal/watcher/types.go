package watcher

import (
	"encoding/json"

	"tinygo.org/x/bluetooth"
)

// WatcherDevice represents a discovered SenseCAP Watcher device
type WatcherDevice struct {
	Name    string
	Address string
	RSSI    int16
	device  bluetooth.ScanResult
}

// ATResponse represents a parsed AT command response
type ATResponse struct {
	Name string          `json:"name"`
	Code int             `json:"code"`
	Data json.RawMessage `json:"data,omitempty"`
}

// DeviceConfigData represents device configuration parameters
type DeviceConfigData struct {
	Timezone        *int   `json:"timezone,omitempty"`
	Daylight        *int   `json:"daylight,omitempty"`
	Timestamp       string `json:"timestamp,omitempty"`
	Brightness      *int   `json:"brightness,omitempty"`
	RGBSwitch       *int   `json:"rgbswitch,omitempty"`
	Sound           *int   `json:"sound,omitempty"`
	ScreenOffTime   *int   `json:"screenofftime,omitempty"`
	ScreenOffSwitch *int   `json:"screenoffswitch,omitempty"`
	Reset           *int   `json:"reset,omitempty"`
	ResetShutdown   *int   `json:"resetshutdown,omitempty"`
	Reboot          *int   `json:"reboot,omitempty"`
	Shutdown        *int   `json:"shutdown,omitempty"`
}

// LocalServiceConfig represents a local service configuration
type LocalServiceConfig struct {
	Switch int    `json:"switch"`
	URL    string `json:"url"`
	Token  string `json:"token"`
}

// LocalServiceData represents all local service configurations
type LocalServiceData struct {
	AudioTaskComposer *LocalServiceConfig `json:"audio_task_composer,omitempty"`
	ImageAnalyzer     *LocalServiceConfig `json:"image_analyzer,omitempty"`
	Training          *LocalServiceConfig `json:"training,omitempty"`
	NotificationProxy *LocalServiceConfig `json:"notification_proxy,omitempty"`
}
