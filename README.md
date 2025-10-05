# SenseCAP Watcher Local Server

A Go implementation of the SenseCAP Watcher local server API for receiving device notifications and processing image analysis requests.

## Features

- ✅ **Full API Implementation** - Implements both endpoints from the OpenAPI specification
- ✅ **Detailed Request Logging** - Logs ALL requests with:
  - HTTP method and full URL (including query strings)
  - All request headers
  - Complete request body (pretty-printed JSON)
  - Remote address and timestamp
- ✅ **404 Detection & Logging** - Unknown routes logged with complete details to catch missed endpoints
- ✅ **Authentication Support** - Optional token-based authentication
- ✅ **Standards Compliant** - Follows the exact API specification from firmware source code
- ✅ **Easy Configuration** - Command-line flags and environment variables

## Quick Start

### 1. Install Dependencies

```bash
go mod download
```

### 2. Run the Server

**Without authentication:**
```bash
go run main.go
```

**With authentication:**
```bash
go run main.go -token my-secret-token
```

**Custom port:**
```bash
go run main.go -port 8080 -token my-secret-token
```

**Using environment variables:**
```bash
PORT=8080 AUTH_TOKEN=my-token go run main.go
```

### 3. Configure Your SenseCAP Watcher Device

Use the AT commands via Bluetooth to configure the device:

**For Notification/Alarm Proxy:**
```
AT+localservice={"data":{"notification_proxy":{"switch":1,"url":"http://192.168.1.100:3000","token":"my-secret-token"}}}
```

**For Image Analyzer:**
```
AT+localservice={"data":{"image_analyzer":{"switch":1,"url":"http://192.168.1.100:3000","token":"my-secret-token"}}}
```

Replace `192.168.1.100` with your server's IP address.

## API Endpoints

### POST /v1/notification/event

Receives device alarm/notification events with:
- Object detection results (bounding boxes)
- Classification results
- Sensor readings (temperature, humidity, CO2)
- Base64-encoded images

**Response:** `{"code": 200}`

### POST /v1/watcher/vision

Receives images for AI analysis with prompts.

**Response:**
```json
{
  "code": 200,
  "data": {
    "state": 0,
    "type": 1,
    "audio": null,
    "img": null
  }
}
```

### GET /health

Health check endpoint.

**Response:** `{"status":"ok","service":"sensecap-local-server"}`

## Request Logging

The server logs detailed information about each request:

### Notification Event Log Example:
```
================================================================================
NOTIFICATION EVENT RECEIVED
================================================================================
Timestamp:   2024-01-15T10:30:45-08:00
Device EUI:  2CF7F1C04430000C (header)
Auth Token:  my-secret-token
Request ID:  550e8400-e29b-41d4-a716-446655440000
Device EUI:  2CF7F1C04430000C (body)
Event Time:  2024-01-15T10:30:40-08:00 (1705340440000 ms)
Text:        Motion detected
Image:       15234 bytes (base64)
--------------------------------------------------------------------------------
INFERENCE DATA
--------------------------------------------------------------------------------
Detected 2 objects (bounding boxes):
  [0] person: confidence=95%, bbox=(120,80,200,300)
  [1] car: confidence=87%, bbox=(350,100,150,250)
Available classes: [person, car, dog, cat]
--------------------------------------------------------------------------------
SENSOR DATA
--------------------------------------------------------------------------------
Temperature: 23.5°C
Humidity:    65%
CO2:         450 ppm
```

## Project Structure

```
sensecap-server/
├── main.go                 # Server entry point
├── go.mod                  # Go module definition
├── handlers/
│   ├── notification.go     # Notification endpoint handler
│   └── vision.go          # Image analyzer endpoint handler
├── middleware/
│   └── middleware.go      # Authentication & logging middleware
├── models/
│   └── models.go          # Data models from OpenAPI spec
├── openapi-local-server.yaml  # OpenAPI 3.0 specification
└── LOCAL_SERVER_API.md    # Complete API documentation
```

## Configuration

### Command-Line Flags

| Flag    | Default | Description                          |
|---------|---------|--------------------------------------|
| -port   | 3000    | Server port                          |
| -token  | ""      | Authentication token (optional)      |

### Environment Variables

| Variable   | Description                               |
|------------|-------------------------------------------|
| PORT       | Server port (overrides -port flag)       |
| AUTH_TOKEN | Authentication token (overrides -token)  |

### Headers Required by Device

All requests from the SenseCAP Watcher include:

```
Authorization: <token>                    # Your configured token
API-OBITER-DEVICE-EUI: <16-char hex EUI> # Device identifier
Content-Type: application/json
```

## Development

### Building

```bash
go build -o sensecap-server
```

### Running the Binary

```bash
./sensecap-server -port 3000 -token my-secret-token
```

### Testing with curl

**Test notification endpoint:**
```bash
curl -X POST http://localhost:3000/v1/notification/event \
  -H "Content-Type: application/json" \
  -H "Authorization: my-secret-token" \
  -H "API-OBITER-DEVICE-EUI: 2CF7F1C04430000C" \
  -d '{
    "requestId": "550e8400-e29b-41d4-a716-446655440000",
    "deviceEui": "2CF7F1C04430000C",
    "events": {
      "timestamp": 1704067200000,
      "text": "Test alert",
      "data": {
        "sensor": {
          "temperature": 23.5,
          "humidity": 65
        }
      }
    }
  }'
```

**Test vision endpoint:**
```bash
curl -X POST http://localhost:3000/v1/watcher/vision \
  -H "Content-Type: application/json" \
  -H "Authorization: my-secret-token" \
  -H "API-OBITER-DEVICE-EUI: 2CF7F1C04430000C" \
  -d '{
    "img": "base64-encoded-image-data",
    "prompt": "Is there a person?",
    "audio_txt": "",
    "type": 1
  }'
```

## Production Deployment

For production use, consider:

1. **Enable Authentication:** Always use the `-token` flag in production
2. **HTTPS:** Put the server behind a reverse proxy (nginx, Caddy) with SSL
3. **Firewall:** Restrict access to trusted devices only
4. **Logging:** Configure log rotation and monitoring
5. **Systemd Service:** Run as a system service for automatic restart

### Example systemd Service

```ini
[Unit]
Description=SenseCAP Watcher Local Server
After=network.target

[Service]
Type=simple
User=sensecap
WorkingDirectory=/opt/sensecap-server
Environment="PORT=3000"
Environment="AUTH_TOKEN=your-production-token"
ExecStart=/opt/sensecap-server/sensecap-server
Restart=always

[Install]
WantedBy=multi-user.target
```

## License

This implementation is based on the SenseCAP Watcher factory firmware source code analysis.

## References

- [SenseCAP Watcher Firmware](https://github.com/Seeed-Studio/SenseCAP-Watcher-Firmware)
- [OpenAPI Specification](./openapi-local-server.yaml)
- [Complete API Documentation](./LOCAL_SERVER_API.md)
