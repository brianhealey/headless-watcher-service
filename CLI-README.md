# SenseCAP Watcher Configuration CLI

A command-line tool for configuring SenseCAP Watcher devices via Bluetooth Low Energy (BLE).

## Features

- **Device Discovery**: Scan for nearby SenseCAP Watcher devices
- **Device Information**: View device details, battery status, firmware versions
- **WiFi Configuration**: Connect the device to WiFi networks
- **WiFi Scanning**: View available WiFi networks from the device
- **Local Services**: Configure local server endpoints for offline operation
  - Audio Task Composer
  - Image Analyzer
  - Training Service
  - Notification Proxy
- **Device Settings**: Configure brightness, sound, RGB LED, screen timeout, timezone
- **Cloud Service**: Enable/disable cloud connectivity
- **Task Flow**: View current task flow status and progress
- **System Actions**: Reboot or factory reset the device

## Prerequisites

### macOS
- Bluetooth 4.0+ hardware
- macOS 10.13 or later
- Xcode command line tools (for building)

### Linux
- Bluetooth 4.0+ hardware
- BlueZ Bluetooth stack
- Required packages:
  ```bash
  sudo apt-get install bluetooth bluez libbluetooth-dev
  ```

### Windows
- Bluetooth 4.0+ hardware
- Windows 10 or later (with BLE support)

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/brianhealey/sensecap-server.git
cd sensecap-server

# Build the application from project root
go build -o cmd/cli/watcher-config ./cmd/cli

# Run the application
./cmd/cli/watcher-config
```

### Cross-Platform Build

```bash
# macOS (Intel)
GOOS=darwin GOARCH=amd64 go build -o watcher-config-macos ./cmd/cli

# macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o watcher-config-macos-arm ./cmd/cli

# Linux
GOOS=linux GOARCH=amd64 go build -o watcher-config-linux ./cmd/cli

# Windows
GOOS=windows GOARCH=amd64 go build -o watcher-config.exe ./cmd/cli
```

## Usage

### Starting the Application

```bash
./watcher-config
```

### Main Menu

Upon starting, you'll see the main menu:

```
========================================
  SenseCAP Watcher Configuration Tool
========================================
Status: Not Connected
----------------------------------------
1. Scan and Connect to Device
2. View Device Information
3. Configure WiFi
4. Scan WiFi Networks
5. Configure Local Services
6. Configure Device Settings
7. Configure Cloud Service
8. View Task Flow Status
9. Exit
----------------------------------------
```

### Workflow

1. **Scan and Connect**: Start by scanning for devices and selecting one to connect
2. **View Info**: Check device information to verify connection
3. **Configure**: Use the menu options to configure various settings
4. **Exit**: Gracefully disconnect and exit when done

## Configuration Examples

### WiFi Setup

```
Select option: 3
Enter SSID: MyHomeNetwork
Enter Password: mypassword123
Configuring WiFi...
✓ WiFi configured successfully
```

### Local Service Configuration

For local server integration (e.g., sensecap-server):

```
Select option: 5
Configure which service?
1. Audio Task Composer
2. Image Analyzer
3. Training
4. Notification Proxy

Select: 1
Enable service? (y/n): y
Enter service URL: http://192.168.1.100:8080/v2/watcher/talk/audio_stream
Enter token (optional):
Configuring local service...
✓ Local service configured successfully
```

### Device Settings

Adjust brightness, sound, and other settings:

```
Select option: 6
=== Device Settings ===
1. Set Brightness
2. Set Sound Volume
3. Toggle RGB LED
4. Set Screen Timeout
5. Set Timezone
6. Reboot Device
7. Factory Reset
8. Back

Select: 1
Enter brightness (0-100): 75
Applying settings...
✓ Settings applied successfully
```

## Common Use Cases

### Initial Device Setup

1. Scan and connect to device
2. Configure WiFi to connect device to network
3. Configure local services (if using local server)
4. Adjust device settings (brightness, sound, etc.)

### Switching to Local Server

1. Connect to device
2. Configure Local Services (option 5)
3. Enable Audio Task Composer with your local server URL
4. Enable Image Analyzer with your local server URL
5. Enable Notification Proxy for alerts
6. Optionally disable Cloud Service (option 7)

### Troubleshooting

1. View Device Information to check battery and firmware
2. Scan WiFi Networks to verify connectivity
3. View Task Flow Status to check current operation
4. Reboot device if needed

## Configuration Reference

### Screen Timeout Values
- `0` = 15 seconds
- `1` = 30 seconds
- `2` = 1 minute
- `3` = 2 minutes
- `4` = 5 minutes
- `5` = 10 minutes
- `6` = Never

### Local Service URLs

When using the sensecap-server project:

- **Audio Task Composer**: `http://<server-ip>:8080/v2/watcher/talk/audio_stream`
- **Image Analyzer**: `http://<server-ip>:8080/v1/watcher/vision`
- **Training**: `http://<server-ip>:8080/v1/training` (if implemented)
- **Notification Proxy**: `http://<server-ip>:8080/v1/notification/event`

Replace `<server-ip>` with your local server's IP address.

## Troubleshooting

### Cannot Find Devices

- Ensure the Watcher device is powered on
- Check that Bluetooth is enabled on your computer
- Verify the device is in pairing mode (check device documentation)
- Move closer to the device

### Connection Failed

- Make sure no other device is connected to the Watcher
- Try power cycling the Watcher device
- Restart the CLI application
- Check Bluetooth permissions on your system

### Linux Specific Issues

If you get permission errors:

```bash
# Add your user to the bluetooth group
sudo usermod -a -G bluetooth $USER

# Reload group membership (or log out/in)
newgrp bluetooth

# Run the application with capabilities
sudo setcap 'cap_net_raw,cap_net_admin+eip' ./watcher-config
```

### macOS Specific Issues

Grant Bluetooth permissions:
- System Preferences → Security & Privacy → Privacy → Bluetooth
- Add Terminal (or your terminal emulator) to the allowed apps

### Commands Timeout

- Device may be processing previous command
- Check device battery level
- Ensure WiFi configuration hasn't caused disconnect
- Try disconnecting and reconnecting

## Development

### Project Structure

```
cmd/cli/
├── main.go            # Application entry point and menu system
└── README.md          # This file

internal/watcher/      # BLE functionality package
├── ble.go            # BLE scanning and connection handling
├── commands.go       # AT command builders
└── types.go          # Shared type definitions
```

The CLI application uses the `internal/watcher` package for all Bluetooth Low Energy operations. This package is shared between the CLI tool and other parts of the sensecap-server project.

### Building from Source

```bash
# From project root
go build -o cmd/cli/watcher-config ./cmd/cli

# Or from anywhere
go build -o watcher-config github.com/brianhealey/sensecap-server/cmd/cli
```

### Adding New Commands

1. Add command builder to `internal/watcher/commands.go`:
```go
func BuildMyCommand(param string) (string, error) {
    return fmt.Sprintf("AT+mycommand=%s", param), nil
}
```

2. Add menu handler in `cmd/cli/main.go`:
```go
func (m *Menu) handleMyCommand() error {
    cmd, err := watcher.BuildMyCommand("value")
    if err != nil {
        return err
    }
    resp, err := m.ble.SendCommand(cmd)
    // Handle response
}
```

3. Add to main menu switch statement in `Run()` method

## License

See the main project LICENSE file.

## Related Documentation

- [BLUETOOTH_API.md](../../BLUETOOTH_API.md) - Complete Bluetooth API reference
- [Main README](../../README.md) - SenseCAP Server project documentation

## Support

For issues and questions:
- Check the [BLUETOOTH_API.md](../../BLUETOOTH_API.md) documentation
- Review troubleshooting section above
- Check device is powered and in range
