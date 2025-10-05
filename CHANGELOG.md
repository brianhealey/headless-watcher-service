# Changelog

All notable changes to the SenseCAP Watcher Local Server project.

## [1.0.0] - 2025-10-04

### Added
- Complete Go implementation of SenseCAP Watcher Local Server API
- Two main endpoints:
  - `POST /v1/notification/event` - Receive device alarms/notifications
  - `POST /v1/watcher/vision` - Receive images for AI analysis
- Health check endpoint: `GET /health`
- Comprehensive request logging with:
  - Full HTTP headers
  - Request method and URL
  - Query string parameters
  - Request body (JSON and raw)
  - Remote address
  - Timestamp
- 404 detection and logging for unmatched routes
  - Logs all headers, body, method, path, and query string
  - Still returns proper 404 response
  - Helps identify missed API endpoints
- Optional token-based authentication
- CORS middleware for development
- Device EUI header validation
- Pretty-printed JSON logs for easy debugging
- Detailed inference data logging:
  - Bounding box detections
  - Classification results
  - Class names
- Sensor data logging:
  - Temperature (°C)
  - Humidity (%)
  - CO2 (ppm)
- OpenAPI 3.0.3 specification
- Complete API documentation
- Test script for all endpoints
- Makefile for common tasks
- Production-ready project structure

### Configuration
- Default port: 8834
- Configurable via CLI flags: `-port`, `-token`
- Configurable via environment variables: `PORT`, `AUTH_TOKEN`

### Documentation
- README.md - Complete project documentation
- QUICK_START.md - Quick start guide
- LOCAL_SERVER_API.md - Full API specification
- openapi-local-server.yaml - OpenAPI spec
- CHANGELOG.md - This file

### Developer Tools
- Makefile with targets: run, build, test, clean, dev, prod
- test-endpoints.sh - Automated endpoint testing
- .gitignore - Git ignore rules

### Models
- Complete Go structs matching OpenAPI specification:
  - NotificationEventRequest
  - ImageAnalyzerRequest
  - InferenceData (boxes and classifications)
  - SensorData
  - All response types

### Middleware
- Logger - Request/response logging with status codes
- NotFoundLogger - Detailed 404 logging
- AuthValidator - Token validation (optional)
- DeviceEUIValidator - Device EUI header validation
- CORS - Cross-origin resource sharing

### Architecture
- Clean separation of concerns:
  - handlers/ - Request handlers
  - middleware/ - HTTP middleware
  - models/ - Data structures
- Standards-compliant API implementation
- Source code verified against SenseCAP Watcher firmware

### Testing
- Manual test script included
- curl examples in documentation
- Example requests for all endpoints

## Future Enhancements

Potential future additions:
- [ ] Database storage for received events
- [ ] Webhook forwarding to other services
- [ ] Real AI image analysis integration
- [ ] Audio response generation
- [ ] Web dashboard for viewing events
- [ ] Metrics and monitoring
- [ ] Docker container
- [ ] Systemd service file
- [ ] HTTPS/TLS support
- [ ] Rate limiting
- [ ] Request replay/debugging tools
