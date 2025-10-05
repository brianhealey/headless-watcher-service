# SenseCAP Voice Interaction Pipeline

## Overview

This server implements a complete voice interaction pipeline for the SenseCAP Watcher device using local AI services.

## Architecture

```
Device Audio (PCM/WAV)
        ↓
   Go Server (port 8834)
        ↓
    Whisper STT (Python, port 5000)
        ↓
    Transcribed Text
        ↓
    Ollama LLM (port 11434)
        ↓
    AI Response Text
        ↓
    Piper TTS (Python, port 5000)
        ↓
    Synthesized Audio (WAV)
        ↓
   Multipart Response (JSON + audio)
        ↓
   Device Playback
```

## Components

### 1. Go Server (`main.go` + `handlers/audio_stream.go`)
- Receives audio from device at `/v2/watcher/talk/audio_stream`
- Orchestrates the pipeline
- Returns multipart response (JSON metadata + boundary + audio)
- Port: 8834

### 2. Python Audio Service (`audio_service.py`)
- **Whisper STT**: `/transcribe` - Converts audio to text
- **Piper TTS**: `/synthesize` - Converts text to speech
- Port: 5000

### 3. Ollama LLM
- Model: `llama3.1:8b-instruct-q4_1`
- Processes transcribed text and generates responses
- Port: 11434

## Installation

All dependencies are already installed:
- ✅ Ollama (via Homebrew)
- ✅ Whisper (in Python venv)
- ✅ Piper TTS (in Python venv)
- ✅ Flask (in Python venv)

Voice model downloaded:
- ✅ `models/piper/en_US-lessac-medium.onnx`

## Usage

### Option 1: Start All Services Together
```bash
./start-all.sh
```

This will:
1. Start Ollama (if not running)
2. Start Audio Processing Service (port 5000)
3. Start SenseCAP Server (port 8834)

### Option 2: Start Services Manually

**Terminal 1: Audio Service**
```bash
./start-audio-service.sh
```

**Terminal 2: Go Server**
```bash
make run
```

## Testing

### Test Audio Service
```bash
# Transcribe audio
curl -X POST http://localhost:5000/transcribe \
  --data-binary @test-audio.wav \
  -H "Content-Type: application/octet-stream"

# Synthesize speech
curl -X POST http://localhost:5000/synthesize \
  -H "Content-Type: application/json" \
  -d '{"text":"Hello, this is a test"}' \
  --output test-output.wav
```

### Test with Device
1. Configure device to use local server (see `README.md`)
2. Press push-to-talk button on device
3. Speak into device
4. Watch server logs for pipeline execution:
   ```
   Step 1: Transcribing audio with Whisper...
   Transcription: 'what is the weather today'
   Step 2: Processing with Ollama...
   Ollama response: 'I don't have access to real-time weather...'
   Step 3: Synthesizing speech with Piper TTS...
   Generated 156800 bytes of audio
   ```
5. Device will play the AI response

## Pipeline Flow Example

**User speaks**: "What is the weather today?"

1. **Device** → Sends 8KB audio (PCM, 16kHz) to server
2. **Whisper** → Transcribes: "What is the weather today?"
3. **Ollama** → Responds: "I don't have access to real-time weather data, but you can check your local weather service."
4. **Piper TTS** → Generates ~150KB WAV audio
5. **Server** → Returns multipart response:
   ```
   {"code":200,"data":{"mode":0,"duration":4800,"stt_result":"What is the weather today?","screen_text":"I don't have access to real-time weather data..."}}---sensecraftboundary---
   [WAV audio binary data]
   ```
6. **Device** → Displays text on screen and plays audio

## Configuration

### Change Ollama Model
Edit `handlers/audio_stream.go`:
```go
"model":  "llama3.1:8b-instruct-q4_1",  // Change this
```

### Change Voice Model
1. Download different Piper voice from https://huggingface.co/rhasspy/piper-voices
2. Update `audio_service.py`:
   ```python
   piper_model_path = "models/piper/YOUR-MODEL.onnx"
   ```

### Adjust AI Prompt
Edit `handlers/audio_stream.go` in `processWithOllama()`:
```go
"prompt": fmt.Sprintf("Your custom prompt: \"%s\"", text),
```

## Performance Notes

- **Whisper (base model)**: ~2-5 seconds for typical voice query
- **Ollama**: ~1-3 seconds for short response
- **Piper TTS**: <1 second for typical sentence
- **Total pipeline**: ~4-10 seconds end-to-end

## Troubleshooting

### Audio service won't start
```bash
source venv/bin/activate
pip install flask openai-whisper piper-tts
```

### Ollama not responding
```bash
brew services restart ollama
ollama list  # Should show llama3.1:8b-instruct-q4_1
```

### Whisper model not found
First run will download the model automatically (~150MB)

### No audio playback on device
- Check server logs for "Generated X bytes of audio"
- Verify `duration` field in JSON is non-zero
- Audio format must be valid WAV

## Logs

The Go server logs show the complete pipeline:
```
================================================================================
AUDIO STREAM RECEIVED
================================================================================
Step 1: Transcribing audio with Whisper...
Transcription: 'hello how are you'
Step 2: Processing with Ollama...
Ollama response: 'I'm doing well, thank you for asking!'
Step 3: Synthesizing speech with Piper TTS...
Generated 89600 bytes of audio
Sent multipart response: 116 bytes JSON + boundary + 89600 bytes audio
```

## Next Steps

- Add context/memory to Ollama conversations (track session_id)
- Implement task mode (mode=1) for device automation
- Add voice activity detection for better transcription
- Cache TTS responses for common phrases
- Add support for multiple languages
