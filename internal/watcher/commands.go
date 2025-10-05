package watcher

import (
	"encoding/json"
	"fmt"
)

// AT Command Builders
// These functions build properly formatted AT commands for the SenseCAP Watcher device

// BuildDeviceInfoQuery builds AT+deviceinfo? command
func BuildDeviceInfoQuery() string {
	return "AT+deviceinfo?"
}

// BuildWiFiQuery builds AT+wifi? command
func BuildWiFiQuery() string {
	return "AT+wifi?"
}

// BuildWiFiTableQuery builds AT+wifitable? command
func BuildWiFiTableQuery() string {
	return "AT+wifitable?"
}

// BuildLocalServiceQuery builds AT+localservice? command
func BuildLocalServiceQuery() string {
	return "AT+localservice?"
}

// BuildCloudServiceQuery builds AT+cloudservice? command
func BuildCloudServiceQuery() string {
	return "AT+cloudservice?"
}

// BuildWiFiSetCommand builds AT+wifi= command
func BuildWiFiSetCommand(ssid, password string) (string, error) {
	data := map[string]string{
		"ssid": ssid,
	}

	if password != "" {
		data["password"] = password
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("AT+wifi=%s", string(jsonData)), nil
}

// BuildDeviceConfigCommand builds AT+devicecfg= command
func BuildDeviceConfigCommand(config DeviceConfigData) (string, error) {
	payload := map[string]interface{}{
		"data": config,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("AT+devicecfg=%s", string(jsonData)), nil
}

// BuildLocalServiceSetCommand builds AT+localservice= command
func BuildLocalServiceSetCommand(services LocalServiceData) (string, error) {
	payload := map[string]interface{}{
		"data": services,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("AT+localservice=%s", string(jsonData)), nil
}

// BuildCloudServiceSetCommand builds AT+cloudservice= command
func BuildCloudServiceSetCommand(enable bool) (string, error) {
	remoteControl := 0
	if enable {
		remoteControl = 1
	}

	payload := map[string]interface{}{
		"data": map[string]int{
			"remotecontrol": remoteControl,
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("AT+cloudservice=%s", string(jsonData)), nil
}

// BuildTaskFlowQuery builds AT+taskflow? command
func BuildTaskFlowQuery() string {
	return "AT+taskflow?"
}

// BuildTaskFlowInfoQuery builds AT+taskflowinfo? command
func BuildTaskFlowInfoQuery() string {
	return "AT+taskflowinfo?"
}
