#!/bin/bash
# Start the Python audio processing service

cd "$(dirname "$0")"

echo "Starting Audio Processing Service (Whisper + Piper TTS)..."
echo "This service will run on http://localhost:5000"
echo ""

source venv/bin/activate
python3 audio_service.py
