# SenseCAP Watcher Local Server

A complete local replacement for SenseCAP cloud services with offline AI capabilities for voice interaction, vision analysis, and task automation.

## Features

### Core Services
- ✅ **Voice Interaction Pipeline** - Complete audio chat with Whisper STT, Ollama LLM, and Piper TTS
- ✅ **Vision Analysis** - LLaVA-powered image understanding and event detection
- ✅ **Task Automation** - Create and manage monitoring tasks with natural language
- ✅ **Offline AI** - All AI processing runs locally (no cloud required)
- ✅ **Docker Support** - Easy deployment with docker-compose
- ✅ **Dual Mode Operation** - Chat mode for conversations, Task mode for automation

### API Implementation
- ✅ **Full API Compatibility** - Implements all SenseCAP Watcher local server endpoints
- ✅ **Authentication** - Optional token-based authentication
- ✅ **Database Storage** - SQLite persistence for tasks and events
- ✅ **Standards Compliant** - Follows exact API spec from firmware source

## Architecture

```
┌─────────────────┐      ┌──────────────────┐      ┌─────────────────┐
│  SenseCAP       │      │  Local Server    │      │  AI Services    │
│  Watcher Device │◄────►│  (Go)            │◄────►│                 │
│                 │      │  Port 8834       │      │  - Ollama LLM   │
│  - Camera       │      │                  │      │  - LLaVA Vision │
│  - Microphone   │      │  ┌────────────┐  │      │  - Whisper STT  │
│  - Speaker      │      │  │ SQLite DB  │  │      │  - Piper TTS    │
└─────────────────┘      │  └────────────┘  │      └─────────────────┘
                         │                  │
                         │  Audio Service   │
                         │  (Python)        │
                         │  Port 8835       │
                         └──────────────────┘
```

## Quick Start

### Option 1: Docker (Recommended)

**1. Copy environment file:**
```bash
cp .env.example .env
# Edit .env to set your AUTH_TOKEN
```

**2. Start all services:**
```bash
docker-compose up -d
```

**3. Pull AI models:**
```bash
docker exec sensecap-ollama ollama pull llama3.1:8b-instruct-q4_1
docker exec sensecap-ollama ollama pull llava:7b
```

**4. Check status:**
```bash
docker-compose ps
curl http://localhost:8834/health
curl http://localhost:8835/health
```

### Option 2: Local Development

**1. Install Dependencies:**

Go:
```bash
go mod download
```

Python:
```bash
python3 -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate
pip install -r python/requirements.txt
```

**2. Install AI Services:**

Ollama (LLM + Vision):
```bash
# macOS/Linux
curl -fsSL https://ollama.com/install.sh | sh
ollama pull llama3.1:8b-instruct-q4_1
ollama pull llava:7b

# Or via brew on macOS
brew install ollama
brew services start ollama
```

**3. Download Piper TTS Model:**
```bash
mkdir -p models/piper
cd models/piper
wget https://huggingface.co/rhasspy/piper-voices/resolve/v1.0.0/en/en_US/lessac/medium/en_US-lessac-medium.onnx
wget https://huggingface.co/rhasspy/piper-voices/resolve/v1.0.0/en/en_US/lessac/medium/en_US-lessac-medium.onnx.json
```

**4. Start Services:**

Using the startup script:
```bash
chmod +x scripts/start-all.sh
./scripts/start-all.sh
```

Or manually:
```bash
# Terminal 1: Audio service
source venv/bin/activate
python3 python/audio_service.py

# Terminal 2: Go server
make run
# Or with authentication:
make run TOKEN=your-secret-token
```

## Project Structure

```
sensecap-server/
├── cmd/
│   └── server/
│       └── main.go              # Application entry point
├── internal/
│   ├── config/                  # Configuration management
│   ├── handlers/                # HTTP handlers
│   │   ├── audio_stream.go     # Voice interaction endpoint
│   │   ├── vision.go           # Image analysis endpoint
│   │   ├── notification.go     # Event notification endpoint
│   │   ├── task_detail.go      # Task flow endpoint
│   │   ├── constants.go        # API constants
│   │   └── prompts.go          # AI prompts
│   ├── middleware/              # HTTP middleware
│   ├── database/                # SQLite layer
│   └── models/                  # Data models
├── python/
│   ├── audio_service.py         # Whisper STT + Piper TTS service
│   ├── requirements.txt         # Python dependencies
│   └── Dockerfile              # Python service container
├── scripts/
│   └── start-all.sh            # Startup script
├── Dockerfile                   # Go service container
├── docker-compose.yaml          # Multi-service orchestration
├── Makefile                     # Development commands
├── go.mod                       # Go dependencies
└── *.md                         # Documentation
```

## API Endpoints

### V2 API (Voice & Tasks)

#### POST /v2/watcher/talk/audio_stream
Voice interaction endpoint with dual-mode support.

**Modes:**
- **Chat Mode:** Natural conversation with the AI
- **Task Mode:** Create monitoring tasks via voice ("notify me when...")

**Request Headers:**
```
Authorization: <token>
API-OBITER-DEVICE-EUI: <16-char hex EUI>
Content-Type: application/octet-stream
```

**Request Body:** Raw PCM audio (16kHz, 16-bit, mono)

**Response:** Multipart (JSON + audio)

#### POST /v2/watcher/talk/view_task_detail
Get task flow details for a created monitoring task.

**Request:** `{"task_id": "abc123"}`

**Response:** Task flow JSON with nodes and edges

### V1 API (Vision & Events)

#### POST /v1/watcher/vision
Image analysis with LLaVA vision model.

**Request:**
```json
{
  "img": "base64-image",
  "prompt": "What do you see?",
  "type": 1
}
```

#### POST /v1/notification/event
Receive device notifications and sensor data.

**Request:**
```json
{
  "requestId": "uuid",
  "deviceEui": "...",
  "events": {
    "timestamp": 1234567890,
    "text": "Alert message",
    "data": {
      "inference": {...},
      "sensor": {...}
    }
  }
}
```

### Health Checks

- `GET /health` - Go server health
- `GET http://localhost:8835/health` - Python audio service health
- `GET http://localhost:11434/api/tags` - Ollama service

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | 8834 | Go server port |
| `DB_PATH` | data/sensecap.db | SQLite database path |
| `AUTH_TOKEN` | (none) | Authentication token |
| `WHISPER_URL` | http://localhost:8835 | Whisper STT service |
| `PIPER_URL` | http://localhost:8835 | Piper TTS service |
| `OLLAMA_URL` | http://localhost:11434 | Ollama LLM service |
| `OLLAMA_MODEL` | llama3.1:8b-instruct-q4_1 | LLM model |
| `LLAVA_MODEL` | llava:7b | Vision model |
| `API_HOST` | localhost | API callback host |
| `API_SCHEMA` | http | API callback schema |

### Command-Line Flags

```bash
go run ./cmd/server \
  -port 8834 \
  -token my-secret-token \
  -whisper-url http://localhost:8835 \
  -ollama-url http://localhost:11434 \
  -ollama-model llama3.1:8b-instruct-q4_1 \
  -llava-model llava:7b \
  -piper-url http://localhost:8835 \
  -db-path data/sensecap.db
```

## Configure Your Device

Use AT commands via Bluetooth to configure your SenseCAP Watcher:

**Notification Proxy:**
```
AT+localservice={"data":{"notification_proxy":{
  "switch":1,"url":"http://<your-ip>:8834","token":"your-token"}}}
```

**Image Analyzer:**
```
AT+localservice={"data":{"image_analyzer":{
  "switch":1,"url":"http://<your-ip>:8834","token":"your-token"}}}
```

Replace `<your-ip>` with your server's IP address.

## Development

### Make Commands

```bash
make help          # Show all available commands
make build         # Build the Go binary
make run           # Run without auth
make run TOKEN=xx  # Run with auth
make test          # Run tests
make clean         # Clean build artifacts
make fmt           # Format code
make lint          # Run linters
```

### Docker Commands

```bash
docker-compose up -d        # Start all services
docker-compose down         # Stop all services
docker-compose logs -f      # View logs
docker-compose ps           # Check status
docker-compose restart      # Restart services
```

### Testing Endpoints

**Test voice endpoint:**
```bash
curl -X POST http://localhost:8834/v2/watcher/talk/audio_stream \
  -H "Authorization: your-token" \
  -H "API-OBITER-DEVICE-EUI: 2CF7F1C04430000C" \
  -H "Content-Type: application/octet-stream" \
  --data-binary @test-audio.pcm
```

**Test vision endpoint:**
```bash
curl -X POST http://localhost:8834/v1/watcher/vision \
  -H "Authorization: your-token" \
  -H "API-OBITER-DEVICE-EUI: 2CF7F1C04430000C" \
  -H "Content-Type: application/json" \
  -d '{
    "img": "base64-encoded-image",
    "prompt": "Is there a person in this image?",
    "type": 1
  }'
```

## Production Deployment

### Security Best Practices

1. **Always use authentication** - Set a strong `AUTH_TOKEN`
2. **Use HTTPS** - Deploy behind a reverse proxy (Caddy, nginx)
3. **Firewall rules** - Restrict access to trusted devices
4. **Regular updates** - Keep AI models and dependencies updated
5. **Monitor logs** - Set up log aggregation and alerting

### Performance Tuning

- **GPU Acceleration:** Configure Ollama to use GPU for faster inference
- **Model Selection:** Use smaller models (7B-8B) for lower latency
- **Resource Limits:** Set Docker memory/CPU limits based on your hardware
- **Database:** Regular VACUUM for SQLite optimization

### Scaling

For multiple devices:
- Run multiple Go server instances behind a load balancer
- Share a single Ollama/audio service instance
- Use PostgreSQL instead of SQLite for better concurrency

## Troubleshooting

**Ollama connection errors:**
```bash
# Check if Ollama is running
curl http://localhost:11434/api/tags

# Start Ollama
brew services start ollama  # macOS
# or
docker exec sensecap-ollama ollama list
```

**Audio service not responding:**
```bash
# Check Python service
curl http://localhost:8835/health

# View Python logs
docker logs sensecap-audio
# or
tail -f python-audio.log
```

**Build errors after restructuring:**
```bash
go mod tidy
go clean -cache
make build
```

## Documentation

- [LOCAL_SERVER_API.md](LOCAL_SERVER_API.md) - Complete API reference
- [PIPELINE.md](PIPELINE.md) - Voice interaction pipeline details
- [QUICK_START.md](QUICK_START.md) - Quick setup guide
- [openapi-local-server.yaml](openapi-local-server.yaml) - OpenAPI specification

## License

This implementation is based on the SenseCAP Watcher factory firmware source code analysis.

## References

- [SenseCAP Watcher Firmware](https://github.com/Seeed-Studio/SenseCAP-Watcher-Firmware)
- [Ollama](https://ollama.com)
- [Whisper](https://github.com/openai/whisper)
- [Piper TTS](https://github.com/rhasspy/piper)
- [LLaVA Vision Model](https://llava-vl.github.io)
