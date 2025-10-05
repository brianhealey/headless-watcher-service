# SenseCAP Watcher Local Server API Documentation

This document describes the complete local server API for the SenseCAP Watcher IoT device, extracted from the factory firmware source code.

## Table of Contents
- [Overview](#overview)
- [Configuration](#configuration)
- [API Endpoints](#api-endpoints)
  - [V1 API](#v1-api)
    - [HTTP Alarm / Notification Proxy](#http-alarm--notification-proxy)
    - [Image Analyzer](#image-analyzer)
    - [Audio Task Composer](#audio-task-composer)
    - [Training](#training)
  - [V2 API](#v2-api)
    - [Audio Stream](#audio-stream)
- [Data Structures](#data-structures)
- [Authentication](#authentication)

---

## Overview

The SenseCAP Watcher device supports local server integration for processing AI tasks, notifications, and training data without relying on cloud services. The local server configuration allows users to:

1. **Process image analysis locally** via custom AI services
2. **Receive device notifications/alarms** via HTTP webhooks
3. **Handle audio task composition** for voice interactions
4. **Perform model training** with local data

All local services are configurable via the device's settings (stored in NVS) and can be enabled/disabled independently.

---

## Configuration

### Local Service Configuration Structure

The device stores local service configurations with the following structure:

```c
typedef struct {
    bool enable;      // Enable/disable the service
    char *url;        // Service endpoint URL
    char *token;      // Authentication token (optional)
} local_service_cfg_type1_t;
```

### Configuration Types

The device supports 4 types of local services (stored as `CFG_ITEM_TYPE1_*`):

1. **CFG_ITEM_TYPE1_AUDIO_TASK_COMPOSER** (Index 0)
   - Voice interaction and audio processing

2. **CFG_ITEM_TYPE1_IMAGE_ANALYZER** (Index 1)
   - AI image analysis and inference

3. **CFG_ITEM_TYPE1_TRAINING** (Index 2)
   - Model training data submission

4. **CFG_ITEM_TYPE1_NOTIFICATION_PROXY** (Index 3)
   - Alarm and event notifications

### Getting/Setting Configuration

The firmware provides these APIs:

```c
// Get configuration for a specific service
esp_err_t get_local_service_cfg_type1(int caller, int cfg_index, local_service_cfg_type1_t *pcfg);

// Set configuration for a specific service
esp_err_t set_local_service_cfg_type1(int caller, int cfg_index, bool enable, char *url, char *token);
```

---

## API Endpoints

### V1 API

The V1 API endpoints are the primary endpoints documented in the firmware source code.

#### HTTP Alarm / Notification Proxy

**Purpose:** Receive device alarm/notification events (e.g., object detection alerts, sensor readings)

**Configuration Index:** `CFG_ITEM_TYPE1_NOTIFICATION_PROXY` (3)

**Default Cloud Endpoint:**
- Test: `https://sensecraft-aiservice-test-api.seeed.cc/v1/notification/event`
- Production: `https://sensecraft-aiservice-api.seeed.cc/v1/notification/event`

#### Request

**Method:** `POST`

**Headers:**
```
Content-Type: application/json
Authorization: <token>          // For local: token as-is, for cloud: "Device <token>"
API-OBITER-DEVICE-EUI: <device_eui>
```

**Request Body:**
```json
{
  "requestId": "<uuid>",
  "deviceEui": "<16-char hex EUI>",
  "events": {
    "timestamp": <unix_timestamp_ms>,
    "text": "<optional text message>",
    "img": "<base64 encoded image>",
    "data": {
      "inference": {
        "boxes": [
          [x, y, w, h, score, target],
          ...
        ],
        // OR
        "classes": [
          [score, target],
          ...
        ],
        "classes_name": ["class1", "class2", ...]
      },
      "sensor": {
        "temperature": <float>,
        "humidity": <int>,
        "CO2": <int>
      }
    }
  }
}
```

**Field Descriptions:**
- `requestId`: UUID v4 generated for this request
- `deviceEui`: 16-character hex device EUI
- `timestamp`: Unix timestamp in milliseconds
- `text`: Optional text message/description
- `img`: Base64-encoded JPEG image (small preview)
- `inference.boxes`: Bounding box detections `[x, y, width, height, confidence_score, class_id]`
- `inference.classes`: Classification results `[confidence_score, class_id]`
- `inference.classes_name`: Array of class names corresponding to class IDs
- `sensor.*`: Sensor readings (if enabled)

#### Response

**Success (code: 200):**
```json
{
  "code": 200
}
```

**Error:**
```json
{
  "code": <error_code>
}
```

**Expected Behavior:**
- The firmware only considers `code: 200` as success
- Any other code value is treated as an error
- The device logs the error code but continues operation

#### Configuration Parameters

Configure via taskflow JSON:

```json
{
  "time_en": true,        // Include timestamp in events (default: true)
  "text_en": true,        // Include text field in events (default: true)
  "image_en": true,       // Include base64 image in events (default: true)
  "sensor_en": true,      // Include sensor data in events (default: true)
  "text": "Custom text",  // Custom text message to include (optional)
  "silence_duration": 30  // Minimum seconds between alarms (default: 30)
}
```

**Silence Duration Behavior:**
- After an alarm is sent, subsequent alarms are suppressed for `silence_duration` seconds
- Prevents flooding the server with rapid-fire notifications
- Uses `difftime()` to track time since last alarm
- First alarm always goes through (when `last_alarm_time == 0`)

---

### Image Analyzer

**Purpose:** Send images to a local AI service for analysis and receive inference results

**Configuration Index:** `CFG_ITEM_TYPE1_IMAGE_ANALYZER` (1)

**Default Cloud Endpoint:**
- Test: `https://sensecraft-aiservice-test-api.seeed.cc/v1/watcher/vision`
- Production: `https://sensecraft-aiservice-api.seeed.cc/v1/watcher/vision`

**Timeout:**
- Cloud service: 30 seconds
- Local service: 120 seconds (2 minutes)

#### Request

**Method:** `POST`

**Headers:**
```
Content-Type: application/json
Authorization: <token>              // For local: token as-is, for cloud: "Device <token>"
API-OBITER-DEVICE-EUI: <device_eui>
```

**Request Body:**
```json
{
  "img": "<base64 encoded JPEG image>",
  "prompt": "<optional AI prompt/instruction>",
  "audio_txt": "<optional audio transcription text>",
  "type": <analyzer_type>
}
```

**Field Descriptions:**
- `img`: Base64-encoded JPEG image (large resolution)
- `prompt`: Optional text prompt for AI analysis (default: empty string)
- `audio_txt`: Optional audio transcription to provide context (default: empty string)
- `type`: Analysis type
  - `0` = `TF_MODULE_IMG_ANALYZER_TYPE_RECOGNIZE` - Analyze/recognize objects in pictures
  - `1` = `TF_MODULE_IMG_ANALYZER_TYPE_MONITORING` - Monitor behavior (default)

#### Response

**Success Response (code: 200):**
```json
{
  "code": 200,
  "data": {
    "state": <0_or_1>,
    "type": <analyzer_type>,
    "audio": "<base64 encoded audio response>",
    "img": "<base64 encoded image (optional)>"
  }
}
```

**Field Descriptions:**
- `code`: HTTP-style status code (200 = success)
- `data.state`: Processing state/result status (0 or 1)
  - `0`: No significant event detected
  - `1`: Event/object detected requiring action
- `data.type`: Echo of the request type or updated type from server
- `data.audio`: Optional base64-encoded audio response (WAV/MP3 format)
- `data.img`: Optional base64-encoded replacement/annotated image

**Behavior:**
- If `result.type == 0` (RECOGNIZE) OR `result.state == 1`: The output is forwarded to downstream modules
- Otherwise: The result is consumed and not propagated

#### Configuration Parameters

Configure via taskflow JSON `body` object:

```json
{
  "body": {
    "prompt": "<default prompt>",
    "audio_txt": "<default audio text>",
    "type": 1
  }
}
```

**Notes:**
- These parameters are set when the task flow module is configured
- They provide default values that are sent with each image analysis request
- Can be dynamically updated by the task flow configuration

---

#### Audio Task Composer

**Purpose:** Process voice interactions and compose audio task responses

**Configuration Index:** `CFG_ITEM_TYPE1_AUDIO_TASK_COMPOSER` (0)

**Note:** Implementation details are not fully exposed in the analyzed files, but the service follows the same configuration pattern.

##### Expected Interface

**Method:** `POST`

**Headers:**
```
Content-Type: application/json
Authorization: <token>
API-OBITER-DEVICE-EUI: <device_eui>
```

**Request Body:**
```json
{
  "audio": "<base64 encoded audio>",
  "context": "<optional context>"
}
```

##### Expected Response

```json
{
  "code": 0,
  "data": {
    "text": "<transcribed/processed text>",
    "audio": "<base64 encoded response audio>"
  }
}
```

---

#### Training

**Purpose:** Submit training data to local model training services

**Configuration Index:** `CFG_ITEM_TYPE1_TRAINING` (2)

**Note:** Implementation details are not fully exposed in the analyzed files, but the service follows the same configuration pattern.

#### Expected Interface

**Method:** `POST`

**Headers:**
```
Content-Type: application/json
Authorization: <token>
API-OBITER-DEVICE-EUI: <device_eui>
```

**Request Body:**
```json
{
  "image": "<base64 encoded image>",
  "label": "<training label>",
  "metadata": {
    // Additional training metadata
  }
}
```

#### Expected Response

```json
{
  "code": 0,
  "data": {
    "status": "accepted"
  }
}
```

---

### V2 API

The V2 API endpoints were discovered through live device testing and represent newer functionality.

#### Audio Stream

**Purpose:** Receive streaming audio data from the device during voice interactions

**Endpoint:** `POST /v2/watcher/talk/audio_stream`

**Discovery:** This endpoint was found through 404 logging when the device initiated a voice interaction.

##### Request

**Method:** `POST`

**Headers:**
```
Content-Type: application/octet-stream
Authorization: <token>
API-OBITER-DEVICE-EUI: <device_eui>
Session-Id: <uuid>
```

**Request Body:**
- Raw binary audio data (not JSON)
- Format: application/octet-stream
- Typical size: 8-16 KB per chunk
- Audio encoding: Device-dependent (likely PCM, MP3, or AAC)

**Field Descriptions:**
- `Session-Id`: UUID v4 that identifies the voice interaction session
- Body: Raw audio bytes streamed from the device microphone

##### Response

**Success (code: 200):**
```json
{
  "code": 200,
  "message": "Audio stream received"
}
```

**Error:**
```json
{
  "error": "Error message"
}
```

**Expected Behavior:**
- Device sends multiple audio chunks for a single voice interaction
- Each chunk uses the same `Session-Id`
- Server must respond with 200 to allow device to continue streaming
- The device uses `User-Agent: ESP32 HTTP Client/1.0`

##### Audio Format Detection

The server can detect common audio formats by inspecting the first few bytes:

- **WAV**: Starts with `RIFF` (52 49 46 46)
- **MP3**: Starts with sync word `FF Ex` (where x is E or F)
- **OGG**: Starts with `OggS` (4F 67 67 53)
- **M4A/AAC**: Contains `ftypM4A ` at offset 4-12

##### Example Session

A typical voice interaction might look like:
```
Session 1: 2267a122-a31e-4a4a-a013-3d5d39d84f68
  Chunk 1: 8634 bytes
  Chunk 2: 8192 bytes
  Chunk 3: 7890 bytes
  ...
```

Each chunk is sent as a separate POST request with the same Session-Id.

---

## Data Structures

### Inference Data Types

The device supports two types of inference results:

#### 1. Bounding Box Detection (`INFERENCE_TYPE_BOX`)

```c
typedef struct {
    uint16_t x;       // X coordinate
    uint16_t y;       // Y coordinate
    uint16_t w;       // Width
    uint16_t h;       // Height
    uint8_t score;    // Confidence score (0-100)
    uint8_t target;   // Class ID
} sscma_client_box_t;
```

#### 2. Classification (`INFERENCE_TYPE_CLASS`)

```c
typedef struct {
    uint8_t score;    // Confidence score (0-100)
    uint8_t target;   // Class ID
} sscma_client_class_t;
```

### Sensor Data Types

#### SHT4x (Temperature & Humidity)
```json
{
  "temperature": <float>,  // Celsius, formula: (raw + 50) / 1000
  "humidity": <int>        // Percentage, formula: raw / 1000
}
```

#### SCD4x (CO2, Temperature & Humidity)
```json
{
  "temperature": <float>,  // Celsius, formula: (raw + 50) / 1000
  "humidity": <int>,       // Percentage, formula: raw / 1000
  "CO2": <int>             // PPM, formula: raw / 1000
}
```

---

## Authentication

### Token Generation

The device generates authentication tokens differently for local vs cloud services:

**For Cloud Services:**
```
Authorization: Device <base64(eui:ai_key)>
```
- Token is generated by Base64 encoding `"EUI:AI_KEY"` (from factory info)
- The word "Device " is prepended
- Example: `Authorization: Device MTIzNDU2Nzg5MEFCQ0RFRjpteS1haS1rZXk=`

**For Local Services:**
```
Authorization: <token>
```
- Token is used exactly as configured in local service settings
- No "Device " prefix
- Can be any string (e.g., `Bearer mytoken`, `mytoken`, etc.)

**Implementation Detail:**
```c
if (local_svc_cfg.enable) {
    // Local: use token as-is
    snprintf(token_buffer, size, "%s", token);
} else {
    // Cloud: prepend "Device "
    snprintf(token_buffer, size, "Device %s", token);
}
```

### Device Headers

All HTTP requests include the device EUI header:

```
API-OBITER-DEVICE-EUI: <16-character hex EUI>
```

**Example:**
```
API-OBITER-DEVICE-EUI: 2CF7F1C04430000C
```

**Source:**
- EUI is retrieved from factory info (`factory_info_eui_get()`)
- It's a 16-character hexadecimal string representing the 8-byte device EUI
- This header is set for ALL local service requests (alarm, image analyzer, etc.)

---

## Configuration via AT Commands

The device can be configured via Bluetooth AT commands:

### Query Local Service Configuration

```
AT+localservice?
```

**Response:**
```json
{
  "name": "localservice",
  "code": 0,
  "data": {
    "audio_task_composer": {
      "switch": 0,
      "url": "",
      "token": ""
    },
    "image_analyzer": {
      "switch": 0,
      "url": "",
      "token": ""
    },
    "training": {
      "switch": 0,
      "url": "",
      "token": ""
    },
    "notification_proxy": {
      "switch": 1,
      "url": "http://192.168.1.100:8080/api/notifications",
      "token": "my-secret-token"
    }
  }
}
```

### Set Local Service Configuration

```
AT+localservice={"data":{"notification_proxy":{"switch":1,"url":"http://192.168.1.100:8080","token":"secret"}}}
```

**Response:**
```json
{
  "code": 0
}
```

---

## Notes

### URL Validation
- URLs must NOT contain spaces (validation performed when setting config)
- Trailing slashes in URLs are automatically removed by the firmware
- URLs should be fully qualified (include protocol: `http://` or `https://`)
- Minimum URL length: 8 characters (checked as `> 7`)

### Timeouts
- **HTTP Alarm (Notification Proxy):** 30,000ms (30 seconds) - hardcoded
- **Image Analyzer:**
  - Cloud service: 30,000ms (30 seconds)
  - Local service: 120,000ms (2 minutes) - allows more processing time for local AI
- Connection timeout: Managed by ESP-IDF HTTP client

### Silence Duration
- HTTP alarms implement a "silence duration" to prevent flooding
- Default: 30 seconds minimum between alarm events (`TF_MODULE_HTTP_ALARM_DEFAULT_SILENCE_DURATION`)
- Configurable via `silence_duration` parameter
- Uses `time()` and `difftime()` to track elapsed time since last alarm
- First alarm always sends (when `last_alarm_time == 0`)

### Data Lifecycle
- Images are base64-encoded JPEG format
- **Small images:** Used for previews and notifications (in HTTP alarm)
- **Large images:** Used for detailed AI analysis (in image analyzer)
- All dynamically allocated data must be freed after processing
- Memory is allocated using `tf_malloc()` (PSRAM) for large buffers
- Queue depth:
  - HTTP Alarm: `TF_MODULE_HTTP_ALARM_QUEUE_SIZE`
  - Image Analyzer: Queue size defined in module header

---

## Example: Setting Up a Local Notification Server

### 1. Configure the Device

Via AT command:
```
AT+localservice={"data":{"notification_proxy":{"switch":1,"url":"http://192.168.1.100:3000/webhook","token":"my-token-123"}}}
```

### 2. Implement the Server Endpoint

```python
from flask import Flask, request, jsonify
import base64
import json
from datetime import datetime

app = Flask(__name__)

@app.route('/webhook', methods=['POST'])
def webhook():
    # Verify token (matches local service config)
    auth_header = request.headers.get('Authorization')
    if auth_header != 'my-token-123':
        return jsonify({'code': 401}), 401

    # Get device EUI
    device_eui = request.headers.get('API-OBITER-DEVICE-EUI')

    # Parse request
    data = request.json
    print(f"\n{'='*60}")
    print(f"Device EUI: {device_eui}")
    print(f"Request ID: {data.get('requestId')}")
    print(f"Device EUI (from body): {data.get('deviceEui')}")

    events = data.get('events', {})

    # Timestamp
    if 'timestamp' in events:
        ts_ms = events['timestamp']
        dt = datetime.fromtimestamp(ts_ms / 1000)
        print(f"Timestamp: {dt.isoformat()}")

    # Text message
    if 'text' in events:
        print(f"Text: {events['text']}")

    # Decode and save image if present
    if 'img' in events and events['img']:
        img_data = base64.b64decode(events['img'])
        filename = f"alert_{data['requestId']}.jpg"
        with open(filename, 'wb') as f:
            f.write(img_data)
        print(f"Saved image: {filename} ({len(img_data)} bytes)")

    # Process inference results
    event_data = events.get('data', {})

    # Inference data (object detection or classification)
    if 'inference' in event_data:
        inference = event_data['inference']
        classes_name = inference.get('classes_name', [])

        # Bounding boxes (object detection)
        if 'boxes' in inference:
            print(f"\nDetected {len(inference['boxes'])} objects:")
            for box in inference['boxes']:
                x, y, w, h, score, target = box
                class_name = classes_name[target] if target < len(classes_name) else f"Class_{target}"
                print(f"  [{class_name}] confidence={score}%, bbox=({x},{y},{w},{h})")

        # Classes (classification)
        if 'classes' in inference:
            print(f"\nClassification results:")
            for cls in inference['classes']:
                score, target = cls
                class_name = classes_name[target] if target < len(classes_name) else f"Class_{target}"
                print(f"  [{class_name}] confidence={score}%")

    # Sensor data
    if 'sensor' in event_data:
        sensor = event_data['sensor']
        print(f"\nSensor readings:")
        if 'temperature' in sensor:
            print(f"  Temperature: {sensor['temperature']}°C")
        if 'humidity' in sensor:
            print(f"  Humidity: {sensor['humidity']}%")
        if 'CO2' in sensor:
            print(f"  CO2: {sensor['CO2']} ppm")

    print(f"{'='*60}\n")

    # Return success - MUST be code 200
    return jsonify({'code': 200})

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=3000)
```

### 3. Test the Integration

The device will now send all alarm events to your local server instead of (or in addition to) the cloud service.

---

## Error Codes

Common error codes returned by the device:

- `0`: Success
- `200`: HTTP success (alarm endpoint)
- `-1`: General failure
- `ERROR_CMD_*`: AT command errors (see source for complete list)
- `ERROR_DATA_*`: Data operation errors

---

## Security Considerations

1. **Authentication**: Always use tokens for production deployments
2. **Network**: Local services should be on a trusted network
3. **HTTPS**: Use HTTPS for sensitive data transmission
4. **Token Storage**: Tokens are stored in device NVS (non-volatile storage)
5. **Validation**: Server should validate device EUI and token on each request

---

## Firmware Version Compatibility

This documentation is based on the factory firmware source code from the SenseCAP-Watcher-Firmware repository. API may vary between firmware versions.

For the latest firmware and updates, refer to the official SenseCAP Watcher documentation.
