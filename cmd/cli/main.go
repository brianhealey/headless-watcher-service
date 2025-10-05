package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/brianhealey/sensecap-server/internal/watcher"
)

func main() {
	log.SetFlags(0)

	fmt.Println("SenseCAP Watcher Configuration Tool")
	fmt.Println("====================================")

	// Initialize BLE handler
	ble, err := watcher.NewBLEHandler()
	if err != nil {
		log.Fatalf("Failed to initialize BLE: %v", err)
	}

	// Ensure cleanup on exit
	defer func() {
		if err := ble.Disconnect(); err != nil {
			log.Printf("Error during disconnect: %v", err)
		}
	}()

	// Create and run menu
	menu := NewMenu(ble)
	if err := menu.Run(); err != nil {
		log.Printf("Menu error: %v", err)
		os.Exit(1)
	}
}

// Menu handles the interactive CLI menu
type Menu struct {
	ble    *watcher.BLEHandler
	reader *bufio.Reader
}

// NewMenu creates a new menu
func NewMenu(ble *watcher.BLEHandler) *Menu {
	return &Menu{
		ble:    ble,
		reader: bufio.NewReader(os.Stdin),
	}
}

// Run starts the main menu loop
func (m *Menu) Run() error {
	for {
		m.printMainMenu()
		choice := m.readInput("Select an option: ")

		switch choice {
		case "1":
			if err := m.scanAndConnect(); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case "2":
			if err := m.viewDeviceInfo(); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case "3":
			if err := m.configureWiFi(); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case "4":
			if err := m.scanWiFiNetworks(); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case "5":
			if err := m.configureLocalServices(); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case "6":
			if err := m.configureDeviceSettings(); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case "7":
			if err := m.configureCloudService(); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case "8":
			if err := m.viewTaskFlowStatus(); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case "9":
			m.ble.Disconnect()
			fmt.Println("Goodbye!")
			return nil
		default:
			fmt.Println("Invalid option")
		}

		fmt.Println()
	}
}

func (m *Menu) printMainMenu() {
	fmt.Println("\n========================================")
	fmt.Println("  SenseCAP Watcher Configuration Tool")
	fmt.Println("========================================")
	if m.ble.IsConnected() {
		fmt.Println("Status: Connected ✓")
	} else {
		fmt.Println("Status: Not Connected")
	}
	fmt.Println("----------------------------------------")
	fmt.Println("1. Scan and Connect to Device")
	fmt.Println("2. View Device Information")
	fmt.Println("3. Configure WiFi")
	fmt.Println("4. Scan WiFi Networks")
	fmt.Println("5. Configure Local Services")
	fmt.Println("6. Configure Device Settings")
	fmt.Println("7. Configure Cloud Service")
	fmt.Println("8. View Task Flow Status")
	fmt.Println("9. Exit")
	fmt.Println("----------------------------------------")
}

func (m *Menu) scanAndConnect() error {
	watchers, err := m.ble.ScanForWatchers(5 * time.Second)
	if err != nil {
		return err
	}

	if len(watchers) == 0 {
		fmt.Println("No Watcher devices found")
		return nil
	}

	fmt.Println("\nFound devices:")
	for i, w := range watchers {
		fmt.Printf("%d. %s\n", i+1, w.Name)
		fmt.Printf("   Address: %s, RSSI: %d dBm\n", w.Address, w.RSSI)
	}

	choice := m.readInput("\nSelect device (1-%d): ", len(watchers))
	idx, err := strconv.Atoi(choice)
	if err != nil || idx < 1 || idx > len(watchers) {
		return fmt.Errorf("invalid selection")
	}

	return m.ble.Connect(watchers[idx-1])
}

func (m *Menu) viewDeviceInfo() error {
	if !m.ble.IsConnected() {
		return fmt.Errorf("not connected to device")
	}

	fmt.Println("Querying device info...")
	resp, err := m.ble.SendCommand(watcher.BuildDeviceInfoQuery())
	if err != nil {
		return err
	}

	if resp.Code != 0 {
		return fmt.Errorf("command failed with code: %d", resp.Code)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return err
	}

	fmt.Println("\n=== Device Information ===")
	fmt.Printf("EUI: %v\n", data["eui"])
	fmt.Printf("BLE MAC: %v\n", data["blemac"])
	fmt.Printf("ESP32 Version: %v\n", data["esp32softwareversion"])
	if himax, ok := data["himaxsoftwareversion"]; ok {
		fmt.Printf("Himax Version: %v\n", himax)
	}
	fmt.Printf("Battery: %v%% (%v mV)\n", data["batterypercent"], data["voltage"])
	fmt.Printf("Brightness: %v%%\n", data["brightness"])
	fmt.Printf("Sound: %v%%\n", data["sound"])
	fmt.Printf("RGB Switch: %v\n", data["rgbswitch"])
	fmt.Printf("Timezone: %v\n", data["timezone"])
	fmt.Printf("Timestamp: %v\n", data["timestamp"])

	return nil
}

func (m *Menu) configureWiFi() error {
	if !m.ble.IsConnected() {
		return fmt.Errorf("not connected to device")
	}

	fmt.Println("\n=== WiFi Configuration ===")
	ssid := m.readInput("Enter SSID: ")
	if ssid == "" {
		return fmt.Errorf("SSID cannot be empty")
	}

	password := m.readInput("Enter Password (leave empty for open network): ")

	cmd, err := watcher.BuildWiFiSetCommand(ssid, password)
	if err != nil {
		return err
	}

	fmt.Println("Configuring WiFi...")
	resp, err := m.ble.SendCommand(cmd)
	if err != nil {
		return err
	}

	if resp.Code == 0 {
		fmt.Println("✓ WiFi configured successfully")
	} else {
		fmt.Printf("WiFi configuration failed with code: %d\n", resp.Code)
	}

	return nil
}

func (m *Menu) scanWiFiNetworks() error {
	if !m.ble.IsConnected() {
		return fmt.Errorf("not connected to device")
	}

	fmt.Println("Scanning for WiFi networks (this may take a few seconds)...")
	resp, err := m.ble.SendCommand(watcher.BuildWiFiTableQuery())
	if err != nil {
		return err
	}

	var data map[string][]map[string]interface{}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return fmt.Errorf("failed to parse WiFi data: %w", err)
	}

	fmt.Println("\n=== Connected WiFi ===")
	if connected, ok := data["connected_wifi"]; ok && len(connected) > 0 {
		for _, net := range connected {
			fmt.Printf("- %s (RSSI: %s, Security: %s)\n", net["ssid"], net["rssi"], net["encryption"])
		}
	} else {
		fmt.Println("No connected networks")
	}

	fmt.Println("\n=== Available WiFi Networks ===")
	if scanned, ok := data["scanned_wifi"]; ok && len(scanned) > 0 {
		for _, net := range scanned {
			fmt.Printf("- %s (RSSI: %s, Security: %s)\n", net["ssid"], net["rssi"], net["encryption"])
		}
	} else {
		fmt.Println("No networks found")
	}

	return nil
}

func (m *Menu) configureLocalServices() error {
	if !m.ble.IsConnected() {
		return fmt.Errorf("not connected to device")
	}

	fmt.Println("\n=== Local Services Configuration ===")
	fmt.Println("Configure which service? ")
	fmt.Println("1. Audio Task Composer")
	fmt.Println("2. Image Analyzer")
	fmt.Println("3. Training")
	fmt.Println("4. Notification Proxy")
	fmt.Println("5. View Current Configuration")
	fmt.Println("6. Back")

	choice := m.readInput("Select: ")

	if choice == "5" {
		return m.viewLocalServices()
	}

	if choice == "6" {
		return nil
	}

	enabled := m.readInput("Enable service? (y/n): ")
	switchVal := 0
	if strings.ToLower(enabled) == "y" {
		switchVal = 1
	}

	url := m.readInput("Enter service URL: ")
	token := m.readInput("Enter token (optional): ")

	serviceConfig := watcher.LocalServiceConfig{
		Switch: switchVal,
		URL:    url,
		Token:  token,
	}

	var services watcher.LocalServiceData
	switch choice {
	case "1":
		services.AudioTaskComposer = &serviceConfig
	case "2":
		services.ImageAnalyzer = &serviceConfig
	case "3":
		services.Training = &serviceConfig
	case "4":
		services.NotificationProxy = &serviceConfig
	default:
		return fmt.Errorf("invalid selection")
	}

	cmd, err := watcher.BuildLocalServiceSetCommand(services)
	if err != nil {
		return err
	}

	fmt.Println("Configuring local service...")
	resp, err := m.ble.SendCommand(cmd)
	if err != nil {
		return err
	}

	if resp.Code == 0 {
		fmt.Println("✓ Local service configured successfully")
	} else {
		fmt.Printf("Configuration failed with code: %d\n", resp.Code)
	}

	return nil
}

func (m *Menu) viewLocalServices() error {
	fmt.Println("Querying local services...")
	resp, err := m.ble.SendCommand(watcher.BuildLocalServiceQuery())
	if err != nil {
		return err
	}

	var data map[string]map[string]interface{}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return err
	}

	fmt.Println("\n=== Local Services ===")
	for service, config := range data {
		fmt.Printf("\n%s:\n", service)
		fmt.Printf("  Enabled: %v\n", config["switch"])
		fmt.Printf("  URL: %v\n", config["url"])
	}

	return nil
}

func (m *Menu) configureDeviceSettings() error {
	if !m.ble.IsConnected() {
		return fmt.Errorf("not connected to device")
	}

	fmt.Println("\n=== Device Settings ===")
	fmt.Println("1. Set Brightness")
	fmt.Println("2. Set Sound Volume")
	fmt.Println("3. Toggle RGB LED")
	fmt.Println("4. Set Screen Timeout")
	fmt.Println("5. Set Timezone")
	fmt.Println("6. Reboot Device")
	fmt.Println("7. Factory Reset")
	fmt.Println("8. Back")

	choice := m.readInput("Select: ")

	var config watcher.DeviceConfigData

	switch choice {
	case "1":
		val := m.readInputInt("Enter brightness (0-100): ")
		config.Brightness = &val
	case "2":
		val := m.readInputInt("Enter volume (0-100): ")
		config.Sound = &val
	case "3":
		enabled := m.readInput("Enable RGB LED? (y/n): ")
		val := 0
		if strings.ToLower(enabled) == "y" {
			val = 1
		}
		config.RGBSwitch = &val
	case "4":
		fmt.Println("0=15s, 1=30s, 2=1min, 3=2min, 4=5min, 5=10min, 6=never")
		val := m.readInputInt("Enter screen timeout: ")
		config.ScreenOffTime = &val
	case "5":
		val := m.readInputInt("Enter timezone offset (hours from UTC): ")
		config.Timezone = &val
	case "6":
		confirm := m.readInput("Reboot device? (y/n): ")
		if strings.ToLower(confirm) == "y" {
			val := 1
			config.Reboot = &val
		} else {
			return nil
		}
	case "7":
		confirm := m.readInput("Factory reset device? This will erase all data! (y/n): ")
		if strings.ToLower(confirm) == "y" {
			val := 1
			config.Reset = &val
		} else {
			return nil
		}
	case "8":
		return nil
	default:
		return fmt.Errorf("invalid selection")
	}

	cmd, err := watcher.BuildDeviceConfigCommand(config)
	if err != nil {
		return err
	}

	fmt.Println("Applying settings...")
	resp, err := m.ble.SendCommand(cmd)
	if err != nil {
		return err
	}

	if resp.Code == 0 {
		fmt.Println("✓ Settings applied successfully")
	} else {
		fmt.Printf("Configuration failed with code: %d\n", resp.Code)
	}

	return nil
}

func (m *Menu) configureCloudService() error {
	if !m.ble.IsConnected() {
		return fmt.Errorf("not connected to device")
	}

	fmt.Println("\n=== Cloud Service Configuration ===")
	enabled := m.readInput("Enable cloud service? (y/n): ")

	cmd, err := watcher.BuildCloudServiceSetCommand(strings.ToLower(enabled) == "y")
	if err != nil {
		return err
	}

	fmt.Println("Configuring cloud service...")
	resp, err := m.ble.SendCommand(cmd)
	if err != nil {
		return err
	}

	if resp.Code == 0 {
		fmt.Println("✓ Cloud service configured successfully")
	} else {
		fmt.Printf("Configuration failed with code: %d\n", resp.Code)
	}

	return nil
}

func (m *Menu) viewTaskFlowStatus() error {
	if !m.ble.IsConnected() {
		return fmt.Errorf("not connected to device")
	}

	fmt.Println("Querying task flow status...")
	resp, err := m.ble.SendCommand(watcher.BuildTaskFlowQuery())
	if err != nil {
		return err
	}

	var data map[string]interface{}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return err
	}

	fmt.Println("\n=== Task Flow Status ===")
	fmt.Printf("Status: %v\n", data["status"])
	fmt.Printf("Task ID: %v\n", data["tlid"])
	fmt.Printf("Current Task: %v\n", data["ctd"])
	fmt.Printf("Module: %v\n", data["module"])
	fmt.Printf("Module Error Code: %v\n", data["module_err_code"])
	fmt.Printf("Progress: %v%%\n", data["percent"])

	return nil
}

func (m *Menu) readInput(prompt string, args ...interface{}) string {
	fmt.Printf(prompt, args...)
	text, _ := m.reader.ReadString('\n')
	return strings.TrimSpace(text)
}

func (m *Menu) readInputInt(prompt string) int {
	text := m.readInput(prompt)
	val, err := strconv.Atoi(text)
	if err != nil {
		return 0
	}
	return val
}
