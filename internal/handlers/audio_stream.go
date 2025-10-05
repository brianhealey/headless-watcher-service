package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/brianhealey/sensecap-server/internal/database"
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

	// Step 1: Transcribe audio using Whisper
	log.Println("Step 1: Transcribing audio with Whisper...")
	transcription, err := transcribeAudio(body)
	if err != nil {
		log.Printf("ERROR: Transcription failed: %v", err)
		http.Error(w, "Transcription failed", http.StatusInternalServerError)
		return
	}
	log.Printf("Transcription: '%s'", transcription)

	// Step 2: Determine mode (chat vs task)
	log.Println("Step 2: Determining interaction mode...")
	mode := determineMode(transcription)
	log.Printf("Mode determined: %d", mode)

	var ollamaResponse string
	if mode == 0 {
		// Chat mode - conversational response
		log.Println("Step 3: Processing chat with Ollama...")
		response, err := processChatMode(transcription)
		if err != nil {
			log.Printf("ERROR: Chat processing failed: %v", err)
			http.Error(w, "Chat processing failed", http.StatusInternalServerError)
			return
		}
		ollamaResponse = response
	} else {
		// Task mode - extract trigger and create task
		log.Println("Step 3: Processing task mode...")
		response, err := processTaskMode(transcription, mode, deviceEUI)
		if err != nil {
			log.Printf("ERROR: Task processing failed: %v", err)
			http.Error(w, "Task processing failed", http.StatusInternalServerError)
			return
		}
		ollamaResponse = response
	}
	log.Printf("Response: '%s'", ollamaResponse)

	// Step 4: Synthesize speech with Piper TTS
	log.Println("Step 4: Synthesizing speech with Piper TTS...")
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
			"mode":        mode,            // 0=chat, 1=task, 2=task_auto
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
	whisperURL := cfg.AI.WhisperURL + "/transcribe"
	resp, err := http.Post(whisperURL, "application/octet-stream", bytes.NewReader(audioData))
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

// determineMode analyzes the transcription to determine the interaction mode
// Returns: 0 = VI_MODE_CHAT, 1 = VI_MODE_TASK, 2 = VI_MODE_TASK_AUTO
func determineMode(transcription string) int {
	// Use Function Selection Assistant prompt to determine mode
	prompt := fmt.Sprintf(`Your name is "watcher" and you are a function selection assistant. You analyze the user's input in relation to the definition of the "Mode List" and then select the most appropriate function from the list.

Mode List:
- Mode 0 (CHAT): General conversation, questions, casual interaction
- Mode 1 (TASK): User wants to set up a monitoring task or automation (e.g., "notify me when...", "alert me if...", "watch for...")
- Mode 2 (TASK_AUTO): Automatic task execution (rarely used)

User input: "%s"

Respond with ONLY the mode number (0, 1, or 2). No explanation.`, transcription)

	requestBody := map[string]interface{}{
		"model":  cfg.AI.OllamaModel,
		"prompt": prompt,
		"stream": false,
	}

	jsonData, _ := json.Marshal(requestBody)
	ollamaURL := cfg.AI.OllamaURL + "/api/generate"
	resp, err := http.Post(ollamaURL, "application/json", bytes.NewReader(jsonData))
	if err != nil {
		log.Printf("WARNING: Mode detection failed, defaulting to chat mode: %v", err)
		return 0 // Default to chat mode
	}
	defer resp.Body.Close()

	var result struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("WARNING: Failed to decode mode detection response, defaulting to chat mode: %v", err)
		return 0
	}

	// Parse mode from response
	modeStr := strings.TrimSpace(result.Response)
	if strings.Contains(modeStr, "1") {
		return 1
	} else if strings.Contains(modeStr, "2") {
		return 2
	}
	return 0 // Default to chat mode
}

// processChatMode handles conversational chat requests
func processChatMode(transcription string) (string, error) {
	// Use official Chat Assistant prompt
	prompt := fmt.Sprintf(`Your name is watcher, and you're a chatbot that can have a nice chat with users based on their input. At the same time, you'll reject all answers to questions about terrorism, racism, yellow violence, political sensitivity, LGBT issues, etc.

User said: "%s"

Provide a brief, conversational response (1-2 sentences max).`, transcription)

	requestBody := map[string]interface{}{
		"model":  cfg.AI.OllamaModel,
		"prompt": prompt,
		"stream": false,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal chat request: %w", err)
	}

	resp, err := http.Post(cfg.AI.OllamaURL + "/api/generate", "application/json", bytes.NewReader(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to call Ollama for chat: %w", err)
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
		return "", fmt.Errorf("failed to decode chat response: %w", err)
	}

	return result.Response, nil
}

// processTaskMode handles task automation requests
func processTaskMode(transcription string, mode int, deviceEUI string) (string, error) {
	// Step 1: Extract trigger condition
	triggerPrompt := fmt.Sprintf(`Extract the trigger condition from this request. Remove time, place, intervals, and actions. Focus on what to detect.

User input: "%s"

CRITICAL: Respond with a simple phrase describing what to detect. No quotes. No punctuation at the end. Maximum 5 words.
Example: "person enters room" or "cat on counter"`, transcription)

	trigger, err := callOllamaSimple(triggerPrompt)
	if err != nil {
		return "", fmt.Errorf("failed to extract trigger: %w", err)
	}
	trigger = cleanLLMResponse(trigger)
	log.Printf("Extracted trigger condition: '%s'", trigger)

	// Step 2: Match to COCO object classes
	cocoClasses := []string{
		"person", "bicycle", "car", "motorcycle", "airplane", "bus", "train", "truck", "boat",
		"traffic light", "fire hydrant", "stop sign", "parking meter", "bench",
		"bird", "cat", "dog", "horse", "sheep", "cow", "elephant", "bear", "zebra", "giraffe",
		"backpack", "umbrella", "handbag", "tie", "suitcase",
	}

	matchPrompt := fmt.Sprintf(`You are the word matching assistant. Match the scenario to ONE keyword from the list.

Scenario: "%s"

Target Keywords: %s

CRITICAL: Respond with ONLY ONE WORD from the list above. No explanation. No quotes. No punctuation.
If the scenario mentions a human/man/woman/person, respond with: person
Otherwise pick the most relevant keyword from the list.`, trigger, strings.Join(cocoClasses, ", "))

	targetObject, err := callOllamaSimple(matchPrompt)
	if err != nil {
		log.Printf("WARNING: Object matching failed: %v", err)
		targetObject = "person" // Default
	}
	targetObject = cleanLLMResponse(targetObject)
	targetObject = strings.TrimSpace(strings.ToLower(targetObject))
	log.Printf("Matched target object: '%s'", targetObject)

	// Step 3: Determine which local model to use
	modelSelectionPrompt := fmt.Sprintf(`Target object: "%s"

The device has 3 built-in TinyML models:
- Model 1: Person detection (person, human, people, man, woman)
- Model 2: Pet detection (dog, cat, puppy, kitten, pet)
- Model 3: Gesture detection (rock, paper, scissors, hand gesture)

CRITICAL: Which model should be used? Respond with ONLY ONE NUMBER: 1, 2, 3, or 0
- 1 if person/human related
- 2 if dog/cat/pet related
- 3 if rock/paper/scissors gesture
- 0 if none match (will require cloud model download)

Respond with ONLY the number. No explanation.`, targetObject)

	modelTypeStr, err := callOllamaSimple(modelSelectionPrompt)
	if err != nil {
		log.Printf("WARNING: Model selection failed, defaulting to person model: %v", err)
		modelTypeStr = "1" // Default to person model
	}
	modelTypeStr = cleanLLMResponse(modelTypeStr)

	// Parse model type
	modelType := 1 // Default to person model
	if strings.Contains(modelTypeStr, "2") {
		modelType = 2 // Pet model
	} else if strings.Contains(modelTypeStr, "3") {
		modelType = 3 // Gesture model
	} else if strings.Contains(modelTypeStr, "0") {
		modelType = 0 // Cloud model
	}
	log.Printf("Selected model type: %d", modelType)

	// Step 4: Generate headline
	headlinePrompt := fmt.Sprintf(`Create a short headline summarizing this task.

User input: "%s"

CRITICAL: Respond with a short headline. Maximum 6 words. No quotes. No punctuation at the end.
Example: "Watch for delivery person" or "Monitor front door activity"`, transcription)

	headline, err := callOllamaSimple(headlinePrompt)
	if err != nil {
		headline = "Task created" // Fallback
	}
	headline = cleanLLMResponse(headline)
	headline = strings.TrimSpace(headline)
	log.Printf("Generated headline: '%s'", headline)

	// Step 4: Delete old tasks and store new task in database
	// Device only supports one task at a time
	oldTasks, err := database.GetTaskFlowsByDevice(deviceEUI)
	if err == nil && len(oldTasks) > 0 {
		for _, oldTask := range oldTasks {
			if err := database.DeleteTaskFlow(oldTask.ID); err != nil {
				log.Printf("WARNING: Failed to delete old task %d: %v", oldTask.ID, err)
			} else {
				log.Printf("Deleted old task: ID=%d, Headline='%s'", oldTask.ID, oldTask.Headline)
			}
		}
	}

	taskFlow := &database.TaskFlow{
		DeviceEUI:        deviceEUI,
		Name:             transcription, // Full original request
		Headline:         headline,
		TriggerCondition: trigger,
		TargetObjects:    []string{targetObject},
		Actions:          []string{"notify"}, // Default action
		ModelType:        modelType,          // LLM-selected model type
	}

	if err := database.SaveTaskFlow(taskFlow); err != nil {
		log.Printf("WARNING: Failed to save task flow to database: %v", err)
		// Continue anyway - return success to user
	} else {
		log.Printf("Task flow saved to database: ID=%d", taskFlow.ID)
	}

	// Return confirmation message
	return fmt.Sprintf("I've created a monitoring task: %s. I'll watch for %s.", headline, trigger), nil
}

// cleanLLMResponse removes quotes, extra whitespace, and trailing punctuation
func cleanLLMResponse(response string) string {
	// Trim whitespace
	result := strings.TrimSpace(response)

	// Remove surrounding quotes (single or double)
	result = strings.Trim(result, "\"'")

	// Remove trailing punctuation
	result = strings.TrimRight(result, ".,!?;:")

	// Trim again
	result = strings.TrimSpace(result)

	return result
}

// callOllamaSimple is a helper to call Ollama with a simple prompt
func callOllamaSimple(prompt string) (string, error) {
	requestBody := map[string]interface{}{
		"model":  cfg.AI.OllamaModel,
		"prompt": prompt,
		"stream": false,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := http.Post(cfg.AI.OllamaURL + "/api/generate", "application/json", bytes.NewReader(jsonData))
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
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Response, nil
}

// processWithOllama sends text to Ollama for LLM processing
// DEPRECATED: Use processChatMode or processTaskMode instead
func processWithOllama(text string) (string, error) {
	requestBody := map[string]interface{}{
		"model":  cfg.AI.OllamaModel,
		"prompt": fmt.Sprintf("You are a helpful AI assistant. The user said: \"%s\"\n\nProvide a brief, conversational response (1-2 sentences max).", text),
		"stream": false,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal Ollama request: %w", err)
	}

	resp, err := http.Post(cfg.AI.OllamaURL + "/api/generate", "application/json", bytes.NewReader(jsonData))
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

	piperURL := cfg.AI.PiperURL + "/synthesize"
	resp, err := http.Post(piperURL, "application/json", bytes.NewReader(jsonData))
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
