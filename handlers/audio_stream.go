package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// AudioStreamHandler handles /v2/watcher/talk/audio_stream POST requests
func AudioStreamHandler(w http.ResponseWriter, r *http.Request) {
	// Read device EUI and session from headers
	deviceEUI := r.Header.Get("API-OBITER-DEVICE-EUI")
	sessionID := r.Header.Get("Session-Id")
	authToken := r.Header.Get("Authorization")

	// Read audio stream body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("ERROR: Failed to read audio stream body: %v", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Log the request
	logAudioStreamRequest(r, deviceEUI, sessionID, authToken, body)

	// Save audio file for debugging
	debugFile := fmt.Sprintf("debug_audio_%s.bin", sessionID)
	if err := os.WriteFile(debugFile, body, 0644); err == nil {
		log.Printf("DEBUG: Saved audio to %s", debugFile)
	}

	// Step 1: Transcribe audio using Whisper
	log.Println("Step 1: Transcribing audio with Whisper...")
	transcription, err := transcribeAudio(body)
	if err != nil {
		log.Printf("ERROR: Transcription failed: %v", err)
		http.Error(w, "Transcription failed", http.StatusInternalServerError)
		return
	}
	log.Printf("Transcription: '%s'", transcription)

	// Step 2: Process with Ollama
	log.Println("Step 2: Processing with Ollama...")
	ollamaResponse, err := processWithOllama(transcription)
	if err != nil {
		log.Printf("ERROR: Ollama processing failed: %v", err)
		http.Error(w, "LLM processing failed", http.StatusInternalServerError)
		return
	}
	log.Printf("Ollama response: '%s'", ollamaResponse)

	// Step 3: Synthesize speech with Piper TTS
	log.Println("Step 3: Synthesizing speech with Piper TTS...")
	audioData, err := synthesizeSpeech(ollamaResponse)
	if err != nil {
		log.Printf("ERROR: Speech synthesis failed: %v", err)
		http.Error(w, "Speech synthesis failed", http.StatusInternalServerError)
		return
	}
	log.Printf("Generated %d bytes of audio", len(audioData))

	// Calculate audio duration from WAV file
	// WAV header is 44 bytes, then raw PCM data
	// Format: 16kHz, mono, 16-bit = 32000 bytes/sec
	audioDataSize := len(audioData) - 44 // Subtract WAV header
	if audioDataSize < 0 {
		audioDataSize = 0
	}
	audioDurationMs := int((float64(audioDataSize) / 32000.0) * 1000)
	log.Printf("Audio duration: %dms (%d bytes WAV, %d bytes PCM)", audioDurationMs, len(audioData), audioDataSize)

	// Prepare JSON response metadata
	// Based on app_voice_interaction.c lines 1189-1310
	jsonResponse := map[string]interface{}{
		"code": 200,
		"data": map[string]interface{}{
			"mode":        0,              // VI_MODE_CHAT
			"duration":    audioDurationMs, // Audio duration in ms
			"stt_result":  transcription,
			"screen_text": ollamaResponse,
		},
	}

	// Marshal JSON
	jsonBytes, err := json.Marshal(jsonResponse)
	if err != nil {
		log.Printf("ERROR: Failed to marshal JSON response: %v", err)
		http.Error(w, "Failed to create response", http.StatusInternalServerError)
		return
	}

	// Build multipart response: JSON + boundary + binary audio
	// Based on app_voice_interaction.c lines 313-348
	boundary := "---sensecraftboundary---"

	// Calculate total response size
	totalSize := len(jsonBytes) + len(boundary) + 1 + len(audioData) // +1 for newline after boundary

	// Set headers - Content-Length is critical for device to download all audio
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", totalSize))
	w.WriteHeader(http.StatusOK)

	// Write JSON metadata
	w.Write(jsonBytes)

	// Write boundary
	w.Write([]byte(boundary + "\n"))

	// Write audio data
	w.Write(audioData)

	log.Printf("Sent multipart response: %d bytes total (%d JSON + boundary + %d audio)",
		totalSize, len(jsonBytes), len(audioData))
}

func logAudioStreamRequest(r *http.Request, deviceEUI, sessionID, authToken string, audioData []byte) {
	log.Println("================================================================================")
	log.Println("AUDIO STREAM RECEIVED")
	log.Println("================================================================================")
	log.Printf("Timestamp:   %s", time.Now().Format(time.RFC3339))
	log.Printf("Action:      %s %s", r.Method, r.URL.Path)
	if r.URL.RawQuery != "" {
		log.Printf("Query:       %s", r.URL.RawQuery)
	}
	log.Printf("Remote Addr: %s", r.RemoteAddr)
	log.Printf("Device EUI:  %s", deviceEUI)
	log.Printf("Session ID:  %s", sessionID)

	// Log all headers
	log.Println("--------------------------------------------------------------------------------")
	log.Println("REQUEST HEADERS")
	log.Println("--------------------------------------------------------------------------------")
	for name, values := range r.Header {
		for _, value := range values {
			log.Printf("  %s: %s", name, value)
		}
	}

	// Log audio stream details
	log.Println("--------------------------------------------------------------------------------")
	log.Println("AUDIO STREAM DATA")
	log.Println("--------------------------------------------------------------------------------")
	log.Printf("Content-Type:  %s", r.Header.Get("Content-Type"))
	log.Printf("Audio Size:    %d bytes", len(audioData))

	// Analyze audio data format
	if len(audioData) > 0 {
		// Check for common audio format headers
		if len(audioData) >= 4 {
			header := audioData[0:4]

			// Check for WAV (RIFF)
			if string(header[0:4]) == "RIFF" {
				log.Println("Audio Format:  WAV (detected RIFF header)")
			} else if header[0] == 0xFF && (header[1]&0xE0) == 0xE0 {
				log.Println("Audio Format:  MP3 (detected sync word)")
			} else if header[0] == 0x4F && header[1] == 0x67 && header[2] == 0x67 && header[3] == 0x53 {
				log.Println("Audio Format:  OGG (detected magic number)")
			} else if len(audioData) >= 12 && string(audioData[4:12]) == "ftypM4A " {
				log.Println("Audio Format:  M4A/AAC")
			} else {
				log.Printf("Audio Format:  Unknown/Raw (first 4 bytes: %02X %02X %02X %02X)",
					header[0], header[1], header[2], header[3])
			}
		}

		// Show first few bytes for debugging
		previewSize := 16
		if len(audioData) < previewSize {
			previewSize = len(audioData)
		}
		log.Printf("First %d bytes: % X", previewSize, audioData[0:previewSize])
	}

	// Estimate duration (rough estimate for common formats)
	// This is a very rough estimate - actual duration depends on sample rate and encoding
	if len(audioData) > 0 {
		// Rough estimate: 16kHz, 16-bit mono PCM = 32KB/sec
		estimatedSeconds := float64(len(audioData)) / 32000.0
		log.Printf("Estimated:     ~%.2f seconds (assuming 16kHz 16-bit mono PCM)", estimatedSeconds)
	}

	log.Println("================================================================================")
	log.Println()
}

// transcribeAudio sends audio to the Python audio service for transcription
func transcribeAudio(audioData []byte) (string, error) {
	resp, err := http.Post("http://localhost:8835/transcribe", "application/octet-stream", bytes.NewReader(audioData))
	if err != nil {
		return "", fmt.Errorf("failed to call transcription service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("transcription service returned %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Text     string `json:"text"`
		Language string `json:"language"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode transcription response: %w", err)
	}

	return result.Text, nil
}

// processWithOllama sends text to Ollama for LLM processing
func processWithOllama(text string) (string, error) {
	requestBody := map[string]interface{}{
		"model":  "llama3.1:8b-instruct-q4_1",
		"prompt": fmt.Sprintf("You are a helpful AI assistant. The user said: \"%s\"\n\nProvide a brief, conversational response (1-2 sentences max).", text),
		"stream": false,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal Ollama request: %w", err)
	}

	resp, err := http.Post("http://localhost:11434/api/generate", "application/json", bytes.NewReader(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to call Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Ollama returned %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Response string `json:"response"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode Ollama response: %w", err)
	}

	return result.Response, nil
}

// synthesizeSpeech sends text to the Python audio service for TTS
func synthesizeSpeech(text string) ([]byte, error) {
	requestBody := map[string]string{
		"text":   text,
		"format": "wav", // Request WAV format for device playback
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal TTS request: %w", err)
	}

	resp, err := http.Post("http://localhost:8835/synthesize", "application/json", bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to call TTS service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("TTS service returned %d: %s", resp.StatusCode, string(body))
	}

	audioData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read TTS audio: %w", err)
	}

	return audioData, nil
}
