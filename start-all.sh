#!/bin/bash
# Start all services for SenseCAP voice interaction pipeline

cd "$(dirname "$0")"

echo "========================================"
echo "SenseCAP Voice Interaction Pipeline"
echo "========================================"
echo ""

# Check if Ollama is running
if ! pgrep -x "ollama" > /dev/null; then
    echo "Starting Ollama service..."
    brew services start ollama
    sleep 3
fi

# Start audio service in background
echo "Starting Audio Processing Service (port 5000)..."
./start-audio-service.sh &
AUDIO_PID=$!

# Wait for audio service to be ready
sleep 5

# Start Go server
echo "Starting SenseCAP Local Server (port 8834)..."
echo ""
echo "Pipeline:"
echo "  1. Device audio → Whisper (STT)"
echo "  2. Transcribed text → Ollama (LLM)"
echo "  3. Ollama response → Piper (TTS)"
echo "  4. Synthesized audio → Device"
echo ""
echo "Services:"
echo "  - Audio Service:  http://localhost:5000"
echo "  - Ollama API:     http://localhost:11434"
echo "  - SenseCAP Server: http://localhost:8834"
echo ""
echo "Press Ctrl+C to stop all services"
echo "========================================"
echo ""

# Trap to kill audio service on exit
trap "kill $AUDIO_PID 2>/dev/null" EXIT

# Start Go server (this will block)
make run
