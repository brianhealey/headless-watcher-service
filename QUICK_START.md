# Quick Start Guide

## Installation & Running

### 1. Start the Server

```bash
# Run with default settings (port 8834, no auth)
go run main.go

# Or use the Makefile
make run
```

### 2. Configure Your Device

Connect to your SenseCAP Watcher via Bluetooth and send these AT commands:

**For Notifications:**
```
AT+localservice={"data":{"notification_proxy":{"switch":1,"url":"http://192.168.1.100:8834","token":""}}}
```

**For Image Analysis:**
```
AT+localservice={"data":{"image_analyzer":{"switch":1,"url":"http://192.168.1.100:8834","token":""}}}
```

Replace `192.168.1.100` with your server's IP address.

### 3. Watch the Logs

The server will log detailed information for every request:

```
================================================================================
NOTIFICATION EVENT RECEIVED
================================================================================
Timestamp:   2025-01-15T10:30:45-08:00
Device EUI:  2CF7F1C04430000C (header)
Request ID:  550e8400-e29b-41d4-a716-446655440000
Event Time:  2025-01-15T10:30:40-08:00
Text:        Motion detected
--------------------------------------------------------------------------------
INFERENCE DATA
--------------------------------------------------------------------------------
Detected 2 objects (bounding boxes):
  [0] person: confidence=95%, bbox=(120,80,200,300)
  [1] car: confidence=87%, bbox=(350,100,150,250)
--------------------------------------------------------------------------------
SENSOR DATA
--------------------------------------------------------------------------------
Temperature: 23.5°C
Humidity:    65%
CO2:         450 ppm
```

## Testing

### Using the Test Script

```bash
# Test all endpoints
./test-endpoints.sh

# Test with custom host/token
./test-endpoints.sh localhost:8834 my-token
```

### Manual Testing with curl

```bash
# Health check
curl http://localhost:8834/health

# Notification event
curl -X POST http://localhost:8834/v1/notification/event \
  -H "Content-Type: application/json" \
  -H "API-OBITER-DEVICE-EUI: 2CF7F1C04430000C" \
  -d '{
    "requestId": "test-123",
    "deviceEui": "2CF7F1C04430000C",
    "events": {
      "timestamp": 1704067200000,
      "text": "Test event",
      "data": {
        "sensor": {
          "temperature": 23.5,
          "humidity": 65
        }
      }
    }
  }'

# Image analysis
curl -X POST http://localhost:8834/v1/watcher/vision \
  -H "Content-Type: application/json" \
  -H "API-OBITER-DEVICE-EUI: 2CF7F1C04430000C" \
  -d '{
    "img": "base64-image-data",
    "prompt": "What do you see?",
    "audio_txt": "",
    "type": 1
  }'
```

## Production Use

### With Authentication

```bash
# Run with token
go run main.go -token my-secret-token

# Or with environment variable
AUTH_TOKEN=my-secret-token go run main.go
```

Then configure your device:
```
AT+localservice={"data":{"notification_proxy":{"switch":1,"url":"http://192.168.1.100:8834","token":"my-secret-token"}}}
```

### Build and Deploy

```bash
# Build binary
make build

# Run binary
./sensecap-server -port 8834 -token my-production-token
```

## Key Features

✅ **404 Detection** - Logs full request details for unmatched routes
✅ **Detailed Logging** - Every request is logged with full context
✅ **Standards Compliant** - Matches exact firmware API specification
✅ **Optional Auth** - Token-based authentication when needed
✅ **Easy Testing** - Included test scripts and examples

## Next Steps

- See [README.md](./README.md) for full documentation
- See [LOCAL_SERVER_API.md](./LOCAL_SERVER_API.md) for complete API reference
- See [openapi-local-server.yaml](./openapi-local-server.yaml) for OpenAPI spec
