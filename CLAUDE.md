# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a local replacement server for SenseCAP Watcher IoT devices, providing offline AI capabilities for voice interaction, vision analysis, and task automation. The server implements the complete SenseCAP Watcher local API discovered through firmware analysis.

**Key Purpose:** Replace cloud-based SenseCAP services with local AI processing using Whisper STT, Ollama LLM, LLaVA vision, and Piper TTS.

## Build & Run Commands

### Quick Start
```bash
# Install dependencies and download Piper TTS model
make install

# Run server (development mode, no auth)
make run

# Run with authentication
make run TOKEN=your-secret-token

# Build binary
make build

# Run tests
make test

# Format and lint code
make lint
```

### Docker Deployment
```bash
# Start all services (Ollama, Audio Service, Go Server)
docker-compose up -d

# Pull required AI models (after first start)
docker exec sensecap-ollama ollama pull llama3.1:8b-instruct-q4_1
docker exec sensecap-ollama ollama pull llava:7b

# View logs
docker-compose logs -f

# Check service health
curl http://localhost:8834/health  # Go server
curl http://localhost:8835/health  # Audio service
```

### CLI Tool (Bluetooth Configuration)
```bash
# Build CLI tool
go build -o cmd/cli/watcher-config ./cmd/cli

# Run interactive configuration tool
./cmd/cli/watcher-config
```

### Testing Specific Features
```bash
# Test voice interaction endpoint
curl -X POST http://localhost:8834/v2/watcher/talk/audio_stream \
  -H "Authorization: your-token" \
  -H "API-OBITER-DEVICE-EUI: 2CF7F1C04430000C" \
  -H "Content-Type: application/octet-stream" \
  --data-binary @test-audio.pcm

# Test vision endpoint
curl -X POST http://localhost:8834/v1/watcher/vision \
  -H "Authorization: your-token" \
  -H "API-OBITER-DEVICE-EUI: 2CF7F1C04430000C" \
  -H "Content-Type: application/json" \
  -d '{"img": "base64-image", "prompt": "What do you see?", "type": 1}'
```

## Architecture Overview

### Service Components

**1. Go Server (Port 8834)** - Main HTTP server
   - Entry point: `cmd/server/main.go`
   - HTTP handlers: `internal/handlers/` (audio_stream.go, vision.go, notification.go, task_detail.go)
   - Database layer: `internal/database/database.go` (SQLite)
   - Configuration: `internal/config/config.go` (environment variables + flags)
   - Middleware: `internal/middleware/middleware.go` (CORS, logging, auth, device EUI validation)

**2. Python Audio Service (Port 8835)** - AI audio processing
   - Implementation: `python/audio_service.py`
   - Endpoints: `/transcribe` (Whisper STT), `/synthesize` (Piper TTS), `/health`
   - Models: Whisper (auto-downloaded), Piper ONNX (downloaded via `make install`)

**3. Ollama (Port 11434)** - LLM and vision models
   - LLM: `llama3.1:8b-instruct-q4_1` (conversational AI)
   - Vision: `llava:7b` (image understanding)

**4. CLI Tool** - Bluetooth device configuration
   - Entry point: `cmd/cli/main.go`
   - BLE package: `internal/watcher/` (ble.go, commands.go, types.go)

### Voice Interaction Pipeline

The core voice pipeline in `internal/handlers/audio_stream.go` orchestrates:
1. **Whisper STT** - Audio → Text transcription
2. **Mode Detection** - Determines chat (0) vs task (1/2) mode using LLM
3. **LLM Processing** - Generates conversational or task-based response
4. **Task Flow Creation** - For task mode, extracts triggers/objects/actions and stores in DB
5. **Piper TTS** - Text → Speech synthesis
6. **Multipart Response** - Returns JSON metadata + audio boundary + WAV data

### API Endpoints

**V2 API (Voice & Tasks):**
- `POST /v2/watcher/talk/audio_stream` - Voice interaction (chat/task modes)
- `POST /v2/watcher/talk/view_task_detail` - Get task flow details

**V1 API (Vision & Events):**
- `POST /v1/watcher/vision` - Image analysis with LLaVA
- `POST /v1/notification/event` - Receive device notifications/alarms

**Health:**
- `GET /health` - Server health check

### Database Schema

SQLite database (`data/sensecap.db`) with two tables:

**task_flows** - User-created monitoring tasks
- Fields: device_eui, name, headline, trigger_condition, target_objects, actions, model_type
- Used for: Task automation storage

**notification_events** - Device alarm/notification history
- Fields: request_id, device_eui, timestamp, text, img, inference_data, sensor_data
- Used for: Event logging and analytics

## Configuration

All configuration via environment variables (`.env` for Docker) or command-line flags:

**Server:**
- `SERVER_PORT` (default: 8834)
- `DB_PATH` (default: data/sensecap.db)
- `AUTH_TOKEN` (optional, enables auth middleware)

**AI Services:**
- `WHISPER_URL` (default: http://localhost:8835)
- `PIPER_URL` (default: http://localhost:8835)
- `OLLAMA_URL` (default: http://localhost:11434)
- `OLLAMA_MODEL` (default: llama3.1:8b-instruct-q4_1)
- `LLAVA_MODEL` (default: llava:7b)
- `PIPER_VOICE` (default: en_US-lessac-medium)

**API Callbacks:**
- `API_HOST` (default: localhost) - Used for task flow callback URLs
- `API_SCHEMA` (default: http) - http or https

### Changing TTS Voice

```bash
# Download new voice model
PIPER_VOICE=en_US-amy-medium make download-models

# Set in .env
PIPER_VOICE=en_US-amy-medium

# Restart audio service
docker-compose restart audio-service
```

## Important Implementation Details

### Audio Format Handling
- **Device sends:** 16kHz PCM audio, 16-bit mono
- **Device expects:** WAV format response with proper headers
- **Padding:** Device audio may contain 0xFF padding bytes (strip before processing)
- **Duration calculation:** PCM data size / 32000 bytes per second

### Authentication
- **Local service:** Token used as-is in `Authorization` header
- **Cloud service:** Token prefixed with `"Device "` (not used in this server)
- **Device header:** `API-OBITER-DEVICE-EUI` contains 16-character hex EUI (REQUIRED)

### Task Mode Processing
Uses official SenseCAP prompts (`internal/handlers/prompts.go`):
- **Function Selection Assistant** - Detects chat vs task intent
- **Trigger Condition Extraction** - Parses "notify me when..." into conditions
- **Word Matching Assistant** - Maps user words to COCO object classes
- **Headline Assistant** - Generates task summaries

### Vision Analysis
- **Type 0 (RECOGNIZE):** General image recognition/analysis
- **Type 1 (MONITORING):** Event detection for monitoring tasks
- **Response state:** 0=no event, 1=event detected (triggers notifications)

## Development Workflow

### Adding a New API Endpoint

1. **Define handler** in `internal/handlers/`:
   ```go
   func MyHandler(w http.ResponseWriter, r *http.Request) {
       deviceEUI := r.Header.Get("API-OBITER-DEVICE-EUI")
       // Implementation
   }
   ```

2. **Register route** in `cmd/server/main.go`:
   ```go
   v1.HandleFunc("/my/endpoint", handlers.MyHandler).Methods("POST")
   ```

3. **Update API documentation** in `LOCAL_SERVER_API.md`

### Modifying AI Prompts

All AI system prompts are in `internal/handlers/prompts.go`. These are the official SenseCAP prompts extracted from firmware and should match device expectations for task mode to work correctly.

### Database Migrations

The database schema is created automatically on startup in `internal/database/database.go`. For schema changes:
1. Update `createTables()` function
2. Handle data migration if needed
3. Test with fresh database: `rm data/sensecap.db && make run`

### Bluetooth Commands

The CLI tool uses AT commands over BLE. To add new commands:
1. Add builder to `internal/watcher/commands.go`
2. Add menu handler to `cmd/cli/main.go`
3. Document in `BLUETOOTH_API.md`

## Testing Notes

- The device sends real audio data as `application/octet-stream`
- Test with actual device or use `ffmpeg` to generate 16kHz PCM test files
- Vision endpoint expects base64-encoded JPEG images
- All endpoints require `API-OBITER-DEVICE-EUI` header (16-char hex)
- Multipart response format is critical - device expects exact boundary format

## Reference Documentation

- **LOCAL_SERVER_API.md** - Complete API specification from firmware analysis
- **PIPELINE.md** - Voice interaction pipeline details
- **BLUETOOTH_API.md** - Bluetooth AT command reference
- **CLI-README.md** - CLI tool usage guide
- **QUICK_START.md** - Quick setup guide
- **README.md** - Main project documentation

## SenseCAP Watcher Firmware Source

The firmware source code is located in a separate repository at `../SenseCAP-Watcher-Firmware/`. This contains the official factory firmware that this server replaces. Use it for:
- Understanding device behavior and expectations
- Discovering undocumented API endpoints
- Validating response formats
- Finding official AI prompts and task flow logic
- Analyzing bluetooth commands, GATT services, and characteristic values

**Note:** This is read-only reference material - never modify the firmware source.

## Development Principles

- **Language Preference:** Keep all code in Go except where AI processing absolutely requires Python (audio service)
- **Python Isolation:** Use virtual environments; isolate Python code within service boundaries
- **Configuration:** Never hard-code configuration - use environment variables with smart defaults
- **Verification:** Always verify assumptions before making code changes
- **Automation:** Maximize setup automation for easy onboarding