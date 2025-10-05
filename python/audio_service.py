#!/usr/bin/env python3
"""
Audio Processing Service for SenseCAP Server
Provides Speech-to-Text (Whisper) and Text-to-Speech (Piper) endpoints
"""

import os
import io
import tempfile
import wave
import numpy as np
import whisper
from piper import PiperVoice
from flask import Flask, request, jsonify, send_file
import logging

# Configure logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

app = Flask(__name__)

# Initialize models
logger.info("Loading Whisper model (base)...")
whisper_model = whisper.load_model("base")
logger.info("Whisper model loaded")

logger.info("Loading Piper TTS model...")
piper_voice_name = os.environ.get("PIPER_VOICE", "en_US-lessac-medium")
piper_model_path = f"models/piper/{piper_voice_name}.onnx"
logger.info(f"Loading Piper voice: {piper_voice_name}")
piper_voice = PiperVoice.load(piper_model_path)
logger.info("Piper TTS model loaded")


@app.route('/health', methods=['GET'])
def health():
    """Health check endpoint"""
    return jsonify({"status": "ok", "models": {"whisper": "base", "piper": piper_voice_name}})


@app.route('/transcribe', methods=['POST'])
def transcribe():
    """
    Transcribe audio to text using Whisper
    Expects: Raw PCM audio data (16kHz, 16-bit, mono) OR WAV/MP3 file
    Returns: {"text": "transcribed text", "language": "en"}
    """
    try:
        # Save uploaded audio to temp file
        audio_data = request.data

        if len(audio_data) == 0:
            return jsonify({"error": "No audio data provided"}), 400

        logger.info(f"Received {len(audio_data)} bytes of audio for transcription")

        # Check if it's raw PCM or a file format
        is_raw_pcm = False
        if len(audio_data) >= 4:
            # Check for common audio headers
            header = audio_data[0:4]
            if header != b'RIFF' and header[0:2] != b'\xff\xfb' and header[0:2] != b'\xff\xfa':
                # No recognizable header, assume raw PCM
                is_raw_pcm = True
                logger.info("Detected raw PCM audio (no file header)")

        # Create temp file
        if is_raw_pcm:
            # Convert raw PCM to WAV format that Whisper can read
            # Raw PCM: 16kHz, 16-bit signed, mono

            # Skip leading 0xFF padding bytes (device buffer initialization)
            start_idx = 0
            for i in range(0, len(audio_data) - 16, 16):
                # Check if we're past the 0xFF padding
                chunk = audio_data[i:i+16]
                if chunk != b'\xff' * 16:
                    start_idx = i
                    break

            if start_idx > 0:
                logger.info(f"Skipped {start_idx} bytes of 0xFF padding")
                audio_data = audio_data[start_idx:]

            pcm_data = np.frombuffer(audio_data, dtype=np.int16)

            with tempfile.NamedTemporaryFile(suffix=".wav", delete=False) as temp_audio:
                temp_path = temp_audio.name
                with wave.open(temp_path, 'wb') as wav_file:
                    wav_file.setnchannels(1)  # Mono
                    wav_file.setsampwidth(2)  # 16-bit = 2 bytes
                    wav_file.setframerate(16000)  # 16kHz
                    wav_file.writeframes(pcm_data.tobytes())
            logger.info(f"Converted raw PCM to WAV: {len(pcm_data)} samples")
        else:
            # It's already a file format (WAV, MP3, etc.)
            with tempfile.NamedTemporaryFile(suffix=".wav", delete=False) as temp_audio:
                temp_audio.write(audio_data)
                temp_path = temp_audio.name

        try:
            # Transcribe with Whisper
            logger.info("Transcribing audio...")
            result = whisper_model.transcribe(temp_path)

            text = result["text"].strip()
            language = result["language"]

            logger.info(f"Transcription complete: '{text}' (language: {language})")

            return jsonify({
                "text": text,
                "language": language
            })
        finally:
            # Clean up temp file
            if os.path.exists(temp_path):
                os.unlink(temp_path)

    except Exception as e:
        logger.error(f"Transcription error: {e}")
        return jsonify({"error": str(e)}), 500


@app.route('/synthesize', methods=['POST'])
def synthesize():
    """
    Synthesize speech from text using Piper TTS
    Expects: JSON {"text": "text to speak", "format": "pcm" or "wav"}
    Returns: Raw PCM or WAV audio file
    """
    try:
        data = request.get_json()

        if not data or 'text' not in data:
            return jsonify({"error": "No text provided"}), 400

        text = data['text']
        output_format = data.get('format', 'pcm')  # Default to raw PCM
        logger.info(f"Synthesizing speech for: '{text}' (format: {output_format})")

        # Generate speech with Piper (returns audio chunks)
        audio_chunks = []
        for audio_chunk in piper_voice.synthesize(text):
            audio_chunks.append(audio_chunk)

        # Combine all audio chunks
        if not audio_chunks:
            return jsonify({"error": "No audio generated"}), 500

        # Combine all raw PCM data
        pcm_data = b''.join(chunk.audio_int16_bytes for chunk in audio_chunks)
        logger.info(f"Generated {len(pcm_data)} bytes of raw PCM from {len(audio_chunks)} chunks")

        if output_format == 'wav':
            # Return as WAV file
            wav_io = io.BytesIO()
            with wave.open(wav_io, 'wb') as wav_file:
                wav_file.setnchannels(1)
                wav_file.setsampwidth(2)
                wav_file.setframerate(16000)
                wav_file.writeframes(pcm_data)

            wav_io.seek(0)
            return send_file(
                wav_io,
                mimetype='audio/wav',
                as_attachment=False,
                download_name='speech.wav'
            )
        else:
            # Return raw PCM (default)
            return send_file(
                io.BytesIO(pcm_data),
                mimetype='application/octet-stream',
                as_attachment=False,
                download_name='speech.pcm'
            )

    except Exception as e:
        logger.error(f"Synthesis error: {e}")
        return jsonify({"error": str(e)}), 500


if __name__ == '__main__':
    # Run on port 8835
    logger.info("Starting Audio Service on http://localhost:8835")
    app.run(host='0.0.0.0', port=8835, debug=False)
