package watcher

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"tinygo.org/x/bluetooth"
)

// GATT UUIDs from firmware
// Service: 49535343-FE7D-4AE5-8FA9-9FAFD205E455
// Write:   49535343-8841-43F4-A8D4-ECBE34729BB3
// Read:    49535343-1E4D-4BD9-BA61-23C647249616
var (
	serviceUUID   = bluetooth.NewUUID([16]byte{0x49, 0x53, 0x53, 0x43, 0xFE, 0x7D, 0x4A, 0xE5, 0x8F, 0xA9, 0x9F, 0xAF, 0xD2, 0x05, 0xE4, 0x55})
	writeCharUUID = bluetooth.NewUUID([16]byte{0x49, 0x53, 0x53, 0x43, 0x88, 0x41, 0x43, 0xF4, 0xA8, 0xD4, 0xEC, 0xBE, 0x34, 0x72, 0x9B, 0xB3})
	readCharUUID  = bluetooth.NewUUID([16]byte{0x49, 0x53, 0x53, 0x43, 0x1E, 0x4D, 0x4B, 0xD9, 0xBA, 0x61, 0x23, 0xC6, 0x47, 0x24, 0x96, 0x16})
)

// BLEHandler manages BLE communication with Watcher devices
type BLEHandler struct {
	adapter         *bluetooth.Adapter
	device          *bluetooth.Device
	writeChar       bluetooth.DeviceCharacteristic
	readChar        bluetooth.DeviceCharacteristic
	responseBuf     strings.Builder
	responseMutex   sync.Mutex
	responseReady   chan struct{}
	connected       bool
	responseTimeout time.Duration
}

// NewBLEHandler creates a new BLE handler
func NewBLEHandler() (*BLEHandler, error) {
	adapter := bluetooth.DefaultAdapter
	err := adapter.Enable()
	if err != nil {
		return nil, fmt.Errorf("failed to enable BLE adapter: %w", err)
	}

	return &BLEHandler{
		adapter:         adapter,
		responseReady:   make(chan struct{}, 1),
		responseTimeout: 30 * time.Second,
	}, nil
}

// ScanForWatchers scans for SenseCAP Watcher devices
func (h *BLEHandler) ScanForWatchers(duration time.Duration) ([]WatcherDevice, error) {
	fmt.Printf("Scanning for Watcher devices for %v...\n", duration)

	// Map to deduplicate devices by address (keep strongest RSSI)
	watcherMap := make(map[string]WatcherDevice)
	var mutex sync.Mutex
	scanDone := make(chan error, 1)

	// Start scan in goroutine
	go func() {
		err := h.adapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
			// Filter for devices with names ending in -WACH
			name := result.LocalName()
			if name != "" && strings.HasSuffix(name, "-WACH") {
				addr := result.Address.String()

				mutex.Lock()
				// Keep the entry with strongest RSSI
				if existing, exists := watcherMap[addr]; !exists || result.RSSI > existing.RSSI {
					watcherMap[addr] = WatcherDevice{
						Name:    name,
						Address: addr,
						RSSI:    result.RSSI,
						device:  result,
					}
					if !exists {
						fmt.Printf("  ✓ Found: %s (RSSI: %d dBm)\n", name, result.RSSI)
					}
				}
				mutex.Unlock()
			}
		})
		scanDone <- err
	}()

	// Wait for either scan to complete, error, or timeout
	select {
	case err := <-scanDone:
		if err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
	case <-time.After(duration):
		// Timeout is normal
	}

	// Stop the scan
	if err := h.adapter.StopScan(); err != nil {
		fmt.Printf("Warning: error stopping scan: %v\n", err)
	}

	// Wait a bit for any pending callbacks
	time.Sleep(100 * time.Millisecond)

	// Convert map to slice
	watchers := make([]WatcherDevice, 0, len(watcherMap))
	for _, w := range watcherMap {
		watchers = append(watchers, w)
	}

	return watchers, nil
}

// Connect connects to a Watcher device
func (h *BLEHandler) Connect(watcher WatcherDevice) error {
	fmt.Printf("Connecting to %s...\n", watcher.Name)

	device, err := h.adapter.Connect(watcher.device.Address, bluetooth.ConnectionParams{})
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}

	h.device = &device

	// Give the device a moment to be ready
	time.Sleep(500 * time.Millisecond)

	// Discover services
	services, err := device.DiscoverServices([]bluetooth.UUID{serviceUUID})
	if err != nil {
		return fmt.Errorf("service discovery failed: %w", err)
	}

	if len(services) == 0 {
		return fmt.Errorf("watcher service not found")
	}

	// Discover characteristics
	chars, err := services[0].DiscoverCharacteristics([]bluetooth.UUID{writeCharUUID, readCharUUID})
	if err != nil {
		return fmt.Errorf("characteristic discovery failed: %w", err)
	}

	// Find write and read characteristics
	for _, char := range chars {
		if char.UUID() == writeCharUUID {
			h.writeChar = char
		} else if char.UUID() == readCharUUID {
			h.readChar = char
		}
	}

	var zeroUUID bluetooth.UUID
	if h.writeChar.UUID() == zeroUUID || h.readChar.UUID() == zeroUUID {
		return errors.New("required characteristics not found")
	}

	// Enable notifications on read characteristic
	err = h.readChar.EnableNotifications(func(buf []byte) {
		h.handleNotification(buf)
	})
	if err != nil {
		return fmt.Errorf("failed to enable notifications: %w", err)
	}

	h.connected = true
	fmt.Printf("Connected to %s\n", watcher.Name)
	return nil
}

// Disconnect disconnects from the device
func (h *BLEHandler) Disconnect() error {
	if h.device != nil && h.connected {
		err := h.device.Disconnect()
		h.connected = false
		h.device = nil
		if err != nil {
			return err
		}
		fmt.Println("Disconnected from device")
	}
	return nil
}

// handleNotification processes incoming notifications from the read characteristic
func (h *BLEHandler) handleNotification(data []byte) {
	h.responseMutex.Lock()
	defer h.responseMutex.Unlock()

	h.responseBuf.Write(data)

	currentBuf := h.responseBuf.String()

	// Check if response is complete (ends with \r\nok\r\n)
	if strings.Contains(currentBuf, "\r\nok\r\n") {
		// Signal that response is ready
		select {
		case h.responseReady <- struct{}{}:
		default:
		}
	}
}

// SendCommand sends an AT command and waits for response
func (h *BLEHandler) SendCommand(command string) (*ATResponse, error) {
	if !h.connected {
		return nil, errors.New("not connected to device")
	}

	// Clear response buffer
	h.responseMutex.Lock()
	h.responseBuf.Reset()
	h.responseMutex.Unlock()

	// Drain any pending response signals
	select {
	case <-h.responseReady:
	default:
	}

	// Add terminator if not present
	if !strings.HasSuffix(command, "\r\n") {
		command += "\r\n"
	}

	// Send command
	_, err := h.writeChar.Write([]byte(command))
	if err != nil {
		return nil, fmt.Errorf("write failed: %w", err)
	}

	// Wait for response with timeout
	select {
	case <-h.responseReady:
		h.responseMutex.Lock()
		response := h.responseBuf.String()
		h.responseMutex.Unlock()

		// Remove \r\nok\r\n suffix
		response = strings.TrimSuffix(response, "\r\nok\r\n")

		// Try to parse as standard AT response
		var atResp ATResponse
		err := json.Unmarshal([]byte(response), &atResp)
		if err != nil {
			return nil, fmt.Errorf("failed to parse response: %w\nRaw: %s", err, response)
		}

		// Special case: some responses (like wifitable) don't have name/code wrapper
		// In this case, the entire response IS the data
		if atResp.Name == "" && len(atResp.Data) == 0 {
			// Re-parse: the response itself is the data
			atResp.Data = json.RawMessage(response)
			atResp.Code = 0 // Assume success if we got valid JSON
		}

		return &atResp, nil

	case <-time.After(h.responseTimeout):
		return nil, errors.New("command timed out")
	}
}

// IsConnected returns whether currently connected to a device
func (h *BLEHandler) IsConnected() bool {
	return h.connected
}
