# SenseCAP Watcher Bluetooth API Documentation

## Overview

The SenseCAP Watcher device supports Bluetooth Low Energy (BLE) communication for device configuration and control. This document describes the Bluetooth interface, including connectivity, authentication, GATT profiles, and available commands.

**Source Files:**
- `/examples/factory_firmware/main/app/app_ble.c` - Bluetooth implementation
- `/examples/factory_firmware/main/app/app_ble.h` - Bluetooth interface
- `/examples/factory_firmware/main/app/at_cmd.c` - AT command handling

---

## Connectivity

### Device Discovery

The device advertises with the following characteristics:

- **Device Name Format:** `<SERIAL_NUMBER>-WACH`
  - Example: `0123456789abcdef01-WACH`
  - Serial number is 18 characters long
- **Advertisement Data:** Contains service UUIDs and device name
- **Advertising Mode:** General discoverable, undirected connectable
- **Connection Mode:** Supports single connection at a time

### Connection Parameters

- **MTU (Maximum Transmission Unit):** Negotiable, minimum 23 bytes
- **Default MTU:** 23 bytes
- **Maximum payload per packet:** MTU - 3 bytes (for BLE overhead)

---

## Authentication

### Pairing Mechanism

The device uses **"Just Works"** pairing (BLE Security Manager I/O Capability: No Input/No Output).

**Security Configuration:**
- **I/O Capability:** `BLE_SM_IO_CAP_NO_IO` (0x03)
- **Secure Connections:** Disabled (`sm_sc = 0`)
- **Pairing Requirements:**
  - No PIN required
  - No passkey display
  - No user confirmation needed
- **Bonding:** Supported
- **Bond Management:** Previous bonds are deleted if repeat pairing occurs

**Security State:**
```c
// From connection descriptor
.sec_state.encrypted     // Encryption status
.sec_state.authenticated // Authentication status
.sec_state.bonded        // Bonding status
```

---

## GATT Profiles

### Primary Service

**Service UUID:** `49535343-FE7D-4AE5-8FA9-9FAFD205E455`

This custom service contains two characteristics for bidirectional communication.

### Characteristics

#### 1. Write Characteristic (Device Input)

**UUID:** `49535343-8841-43F4-A8D4-ECBE34729BB3`

**Properties:**
- READ
- WRITE
- NOTIFY

**Purpose:** Clients write AT commands to this characteristic.

**Data Flow:**
1. Client writes command data
2. Device receives and queues message
3. AT command processor handles the command
4. Response sent via Read characteristic

#### 2. Read Characteristic (Device Output)

**UUID:** `49535343-1E4D-4BD9-BA61-23C647249616`

**Properties:**
- READ
- WRITE
- NOTIFY

**Purpose:** Device sends responses via indications on this characteristic.

**Data Flow:**
1. Device processes command
2. Generates JSON response
3. Sends indication(s) to subscribed client
4. Handles fragmentation for large responses

### GATT Access

**Write Operations:**
- Client writes to Write Characteristic
- Data is queued to `ble_msg_queue`
- AT command task processes the queue
- Maximum buffer size: 500KB
- Buffer grows dynamically in 100KB increments

**Read Operations:**
- Device sends indications via Read Characteristic
- Automatic fragmentation for payloads > MTU
- Retry mechanism with exponential backoff
- Maximum retries: 10

---

## Bluetooth Commands

All commands follow the AT command format and use JSON for data exchange.

### Command Format

**Query Command:**
```
AT+<command>?\r\n
```

**Set Command:**
```
AT+<command>={<JSON_DATA>}\r\n
```

**Response Format:**
```json
{
  "name": "<command_name>",
  "code": <error_code>,
  "data": { <response_data> }
}
\r\nok\r\n
```

### Command Processing

**Pattern:** `^AT\\+([a-zA-Z0-9]+)(\\?|=(\\{.*\\}))?\\r\\n$`

**Steps:**
1. Client writes command to Write Characteristic
2. Device accumulates data until `\r\n` received
3. Regex validates command format
4. Command dispatcher executes handler
5. Response sent via Read Characteristic indication

**Error Codes:**
```c
AT_CMD_SUCCESS = 0x0000          // Command succeeded
ERROR_UNKNOWN = 0x2011            // Unknown error
ERROR_INVALID_PARAM = 0x2012      // Invalid parameter
ERROR_CMD_FORMAT = 0x2020         // Command format error
ERROR_CMD_UNSUPPORTED = 0x2021    // Unsupported command
ERROR_CMD_COMMAND_NOT_FOUND = 0x2025  // Command not found
ERROR_NETWORK_FAIL = 0x2030       // Network connection failed
ERROR_FILE_OPEN_FAIL = 0x2050     // File operation failed
// See at_cmd.h for complete error code list
```

---

## Available Commands

### 1. Device Information Query

**Command:** `AT+deviceinfo?\r\n`

**Response:**
```json
{
  "name": "deviceinfo",
  "code": 0,
  "data": {
    "eui": "0123456789ABCDEF",        // Device EUI (16 hex chars)
    "blemac": "AABBCCDDEEFF",          // BLE MAC address (12 hex chars)
    "automatic": 1,                    // Auto time update (0/1)
    "rgbswitch": 1,                    // RGB LED enabled (0/1)
    "sound": 50,                       // Sound volume (0-100)
    "brightness": 80,                  // Screen brightness (0-100)
    "screenofftime": 3,                // Screen timeout setting (0-6)
    "screenoffswitch": 1,              // Screen auto-off enabled (0/1)
    "timestamp": "1704067200",         // Current timestamp (seconds)
    "timezone": -5,                    // Timezone offset (hours)
    "esp32softwareversion": "1.0.0",   // ESP32 firmware version
    "himaxsoftwareversion": "1.0.0",   // Himax firmware version
    "batterypercent": 85,              // Battery percentage (0-100)
    "voltage": 4200                    // Battery voltage (millivolts)
  }
}
```

### 2. Device Configuration

**Command:** `AT+devicecfg={<JSON>}\r\n`

**Parameters:**
```json
{
  "data": {
    "timezone": -5,           // Timezone offset in hours (required for time config)
    "daylight": 0,            // Daylight saving time offset (0/1)
    "timestamp": "1704067200",// UTC timestamp string
    "brightness": 80,         // Screen brightness (0-100)
    "rgbswitch": 1,          // RGB LED switch (0/1)
    "sound": 50,             // Sound volume (0-100)
    "screenofftime": 3,      // Screen timeout (0-6)
                             // 0=15s, 1=30s, 2=1min, 3=2min, 4=5min, 5=10min, 6=never
    "screenoffswitch": 1,    // Auto screen off (0/1)
    "reset": 0,              // Factory reset and reboot (0/1)
    "resetshutdown": 0,      // Factory reset and shutdown (0/1)
    "reboot": 0,             // Reboot device (0/1)
    "shutdown": 0            // Shutdown device (0/1)
  }
}
```

**Response:**
```json
{
  "name": "devicecfg",
  "code": 0
}
```

### 3. WiFi Connection

**Command:** `AT+wifi={<JSON>}\r\n`

**Parameters:**
```json
{
  "ssid": "MyNetwork",
  "password": "MyPassword"  // Optional, omit for open networks
}
```

**Response:**
```json
{
  "name": "wifi",
  "code": 0,  // 0=success, other=WiFi error reason code
  "data": {
    "ssid": "MyNetwork"
  }
}
```

**Notes:**
- Connection attempt timeout: ~15 seconds
- Device will automatically reconnect on disconnect

### 4. WiFi Status Query

**Command:** `AT+wifi?\r\n`

**Response:**
```json
{
  "name": "wifi",
  "code": 1,  // Network connection flag (0=disconnected, 1=connected)
  "data": {
    "ssid": "MyNetwork",
    "rssi": "-45",          // Signal strength in dBm
    "encryption": "WPA2"    // Security type
  }
}
```

### 5. WiFi Scan

**Command:** `AT+wifitable?\r\n`

**Response:**
```json
{
  "connected_wifi": [
    {
      "ssid": "CurrentNetwork",
      "rssi": "-45",
      "encryption": "WPA2"
    }
  ],
  "scanned_wifi": [
    {
      "ssid": "Network1",
      "rssi": "-60",
      "encryption": "WPA2"
    },
    {
      "ssid": "Network2",
      "rssi": "-75",
      "encryption": "WPA"
    }
  ]
}
```

**Notes:**
- Scan is triggered on command execution
- Results may take several seconds

### 6. Task Flow Configuration

**Command:** `AT+taskflow={<JSON>}\r\n`

**Parameters:**
```json
{
  "data": {
    // Task flow JSON configuration
    // Format is device-specific, see task flow documentation
  }
}
```

**Response:**
```json
{
  "name": "taskflow",
  "code": 0,
  "data": {}
}
```

### 7. Task Flow Status Query

**Command:** `AT+taskflow?\r\n`

**Response:**
```json
{
  "name": "taskflow",
  "code": 0,
  "data": {
    "status": 0,           // Engine status (0=idle, 1=starting, 2=running, etc.)
    "tlid": 12345,         // Task flow ID
    "ctd": 67890,          // Current task ID
    "module": "detection", // Active module name
    "module_err_code": 0,  // Module error code
    "percent": 100         // AI model download progress (0-100)
  }
}
```

### 8. Task Flow Info Query

**Command:** `AT+taskflowinfo?\r\n`

**Response:**
```json
{
  "name": "taskflowinfo",
  "code": 0,
  "data": {
    "taskflow": {
      // Complete task flow configuration object
    }
  }
}
```

### 9. Cloud Service Configuration

**Command:** `AT+cloudservice={<JSON>}\r\n`

**Parameters:**
```json
{
  "data": {
    "remotecontrol": 1  // Enable cloud service (0/1)
  }
}
```

**Response:**
```json
{
  "name": "cloudservice",
  "code": 0
}
```

### 10. Cloud Service Status Query

**Command:** `AT+cloudservice?\r\n`

**Response:**
```json
{
  "name": "cloudservice",
  "code": 0,
  "data": {
    "remotecontrol": 1  // Cloud service enabled (0/1)
  }
}
```

### 11. Local Service Configuration

**Command:** `AT+localservice={<JSON>}\r\n`

**Parameters:**
```json
{
  "data": {
    "audio_task_composer": {
      "switch": 1,                    // Enable service (0/1)
      "url": "http://192.168.1.100:8080/v2/watcher/talk/audio_stream",
      "token": "optional_auth_token"  // Optional authentication token
    },
    "image_analyzer": {
      "switch": 1,
      "url": "http://192.168.1.100:8080/v1/watcher/vision",
      "token": ""
    },
    "training": {
      "switch": 0,
      "url": "http://192.168.1.100:8080/v1/training",
      "token": ""
    },
    "notification_proxy": {
      "switch": 1,
      "url": "http://192.168.1.100:8080/v1/notification/event",
      "token": ""
    }
  }
}
```

**Response:**
```json
{
  "code": 0  // Success or error code
}
```

**Notes:**
- URLs must not contain spaces
- Token is optional (use empty string if not needed)
- Each service can be independently enabled/disabled

### 12. Local Service Query

**Command:** `AT+localservice?\r\n`

**Response:**
```json
{
  "name": "localservice",
  "code": 0,
  "data": {
    "audio_task_composer": {
      "switch": 1,
      "url": "http://192.168.1.100:8080/v2/watcher/talk/audio_stream",
      "token": ""
    },
    "image_analyzer": {
      "switch": 1,
      "url": "http://192.168.1.100:8080/v1/watcher/vision",
      "token": ""
    },
    "training": {
      "switch": 0,
      "url": "http://192.168.1.100:8080/v1/training",
      "token": ""
    },
    "notification_proxy": {
      "switch": 1,
      "url": "http://192.168.1.100:8080/v1/notification/event",
      "token": ""
    }
  }
}
```

### 13. Emoji/Image Download

**Command:** `AT+emoji={<JSON>}\r\n`

**Parameters:**
```json
{
  "filename": "emoji_set_1",
  "urls": [
    "http://example.com/emoji1.png",
    "http://example.com/emoji2.png",
    "http://example.com/emoji3.png"
  ]
}
```

**Response:**
```json
{
  "name": "emoji",
  "data": [
    0,    // Success for emoji1
    0,    // Success for emoji2
    8272  // Error code for emoji3
  ]
}
```

**Notes:**
- Maximum images per command: Check MAX_IMAGES constant
- Each URL result has individual error code
- Images are stored in device filesystem

### 14. Bind/Pairing

**Command:** `AT+bind={<JSON>}\r\n`

**Parameters:**
```json
{
  "code": 123456  // Binding code/index
}
```

**Response:**
```json
{
  "name": "bind",
  "code": 0
}
```

**Notes:**
- Used for device pairing/binding operations
- Exact binding mechanism is application-specific

---

## Configuration via Bluetooth

### Configurable Settings

The following settings can be configured via Bluetooth:

#### Time Configuration
- **Timezone:** Hours offset from UTC (-12 to +14)
- **Daylight Saving:** Enable/disable (0/1)
- **Timestamp:** Manual time setting (Unix timestamp)
- **Auto Update:** Automatic time sync (0/1)

#### Display Settings
- **Brightness:** 0-100%
- **Screen Off Time:** 0-6 (15s, 30s, 1m, 2m, 5m, 10m, never)
- **Screen Off Switch:** Enable auto screen-off (0/1)

#### Audio/Visual Settings
- **Sound Volume:** 0-100%
- **RGB LED:** Enable/disable (0/1)

#### Network Settings
- **WiFi SSID:** Network name
- **WiFi Password:** Network password
- **Cloud Service:** Enable/disable remote control

#### Local Server Settings
- **Audio Task Composer:** URL and enable/disable
- **Image Analyzer:** URL and enable/disable
- **Training Service:** URL and enable/disable
- **Notification Proxy:** URL and enable/disable

#### System Actions
- **Factory Reset:** With reboot or shutdown
- **Reboot:** Restart device
- **Shutdown:** Power off device

---

## Implementation Notes

### Message Buffering

**Buffer Configuration:**
- Initial size: 100KB
- Growth step: 100KB
- Maximum size: 500KB
- Location: PSRAM (external RAM)

**Buffer Behavior:**
- Accumulates data until `\r\n` received
- Grows dynamically as needed
- Resets on BLE disconnect
- Protected by mutex semaphore

### Response Transmission

**Indication Mechanism:**
```c
// Characteristics
Write Characteristic: Client → Device (commands)
Read Characteristic:  Device → Client (responses via indication)
```

**Fragmentation:**
- Automatic for responses > (MTU - 3)
- Retry with exponential backoff
- Maximum wait for buffer allocation: 10 seconds
- Maximum retries: 10

**Flow Control:**
- Event group for indication status
- Prevents concurrent indications
- Waits for acknowledgment before sending next fragment

### Event Handling

**BLE Events Monitored:**
- Connection established/failed
- Disconnection
- MTU update
- Encryption change
- Subscription status

**Application Events:**
- WiFi status changes
- Task flow status updates
- AI model OTA progress
- View data updates

---

## Usage Examples

### Example 1: Query Device Info

**Send:**
```
AT+deviceinfo?\r\n
```

**Receive:**
```json
{
  "name": "deviceinfo",
  "code": 0,
  "data": {
    "eui": "0123456789ABCDEF",
    "blemac": "AABBCCDDEEFF",
    "brightness": 80,
    "sound": 50,
    "timezone": -5
  }
}
\r\nok\r\n
```

### Example 2: Configure WiFi

**Send:**
```
AT+wifi={"ssid":"MyHomeWiFi","password":"SecurePass123"}\r\n
```

**Receive (on success):**
```json
{
  "name": "wifi",
  "code": 0,
  "data": {
    "ssid": "MyHomeWiFi"
  }
}
\r\nok\r\n
```

### Example 3: Configure Local Server

**Send:**
```
AT+localservice={"data":{"audio_task_composer":{"switch":1,"url":"http://192.168.1.100:8080/v2/watcher/talk/audio_stream","token":""}}}\r\n
```

**Receive:**
```json
{
  "code": 0
}
\r\nok\r\n
```

### Example 4: Adjust Brightness

**Send:**
```
AT+devicecfg={"data":{"brightness":70}}\r\n
```

**Receive:**
```json
{
  "name": "devicecfg",
  "code": 0
}
\r\nok\r\n
```

---

## Development Guidelines

### For Client Applications

1. **Connection:**
   - Scan for devices matching pattern `*-WACH`
   - Connect to device (no PIN required)
   - Discover GATT services and characteristics
   - Subscribe to Read characteristic for notifications

2. **Sending Commands:**
   - Ensure command ends with `\r\n`
   - Wait for response before sending next command
   - Handle MTU negotiation appropriately
   - Maximum command size should consider device buffer limits

3. **Receiving Responses:**
   - Subscribe to Read characteristic indications
   - Reassemble fragmented responses
   - Parse JSON response
   - Check `code` field for errors
   - Look for `\r\nok\r\n` suffix

4. **Error Handling:**
   - Check response `code` field
   - Implement timeout for commands (15-30 seconds recommended)
   - Handle connection loss gracefully
   - Retry logic for transient errors

### For Device Integration

1. **Initialization:**
   - Call `app_ble_init()` during startup
   - Register AT command handlers
   - Initialize message queue
   - Set up event handlers

2. **Adding New Commands:**
   - Define command handler function
   - Register with `add_command()`
   - Follow AT command pattern
   - Return appropriate error codes
   - Send JSON response via `send_at_response()`

3. **Advertising Control:**
   - `app_ble_adv_switch(true/false)` - Start/stop advertising
   - `app_ble_adv_pause()` / `app_ble_adv_resume()` - Paired pause/resume

4. **Data Transmission:**
   - `app_ble_send_indicate()` - Send data to client
   - Automatically handles fragmentation
   - Returns error if not connected

---

## Security Considerations

1. **Pairing:** Uses "Just Works" - no authentication required
2. **Encryption:** Supported but not mandatory
3. **Bond Management:** Old bonds deleted on repeat pairing
4. **Attack Surface:**
   - Unencrypted communication possible
   - No authentication for sensitive commands
   - WiFi credentials transmitted (ensure encryption)
   - Local server URLs can be modified

**Recommendations:**
- Use in trusted environments only
- Enable BLE encryption when possible
- Implement additional application-level authentication if needed
- Sanitize all input parameters
- Validate URLs before configuration

---

## Troubleshooting

### Connection Issues

**Problem:** Cannot connect to device
- Check device is advertising (LED indicator or app status)
- Verify BLE is enabled on client device
- Ensure no other client is connected
- Try power cycling the device

**Problem:** Connection drops frequently
- Check signal strength (keep devices close)
- Verify no interference from other BLE devices
- Check battery level on device

### Command Issues

**Problem:** Command not recognized
- Verify command format matches pattern
- Ensure `\r\n` terminator is present
- Check for extra whitespace
- Validate JSON structure

**Problem:** No response to command
- Verify subscription to Read characteristic
- Check connection is still active
- Wait adequate time (some commands take seconds)
- Verify MTU is sufficient for response

**Problem:** Incomplete responses
- Check MTU negotiation succeeded
- Ensure proper fragmentation handling
- Verify indication acknowledgment

### Configuration Issues

**Problem:** Settings not applied
- Check response `code` field for errors
- Verify parameter ranges are valid
- Ensure device has necessary permissions
- Check device storage is not full

**Problem:** WiFi connection fails
- Verify correct SSID and password
- Check WiFi network is in range
- Ensure network security type is supported
- Check for special characters in credentials

---

## Reference

### File Locations
- BLE Implementation: `examples/factory_firmware/main/app/app_ble.c`
- AT Command Handler: `examples/factory_firmware/main/app/at_cmd.c`
- Data Definitions: `examples/factory_firmware/main/app/data_defs.h`

### Key Constants
```c
// UUIDs
Service UUID:    49535343-FE7D-4AE5-8FA9-9FAFD205E455
Write Char UUID: 49535343-8841-43F4-A8D4-ECBE34729BB3
Read Char UUID:  49535343-1E4D-4BD9-BA61-23C647249616

// Buffer Sizes
AT_CMD_BUFFER_LEN_STEP:   102400  // 100KB
AT_CMD_BUFFER_MAX_LEN:    512000  // 500KB
BLE_MSG_Q_SIZE:           10

// Timeouts
Default MTU:              23 bytes
Connection timeout:       ~15 seconds
Command timeout:          15-30 seconds (recommended)
```

### Related Documentation
- ESP32 NimBLE Documentation
- SenseCAP Watcher Task Flow API
- Local Server API Documentation
