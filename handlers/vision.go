package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/brianhealey/sensecap-server/models"
)

// VisionHandler handles /v1/watcher/vision POST requests
func VisionHandler(w http.ResponseWriter, r *http.Request) {
	// Read device EUI from header
	deviceEUI := r.Header.Get("API-OBITER-DEVICE-EUI")
	authToken := r.Header.Get("Authorization")

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("ERROR: Failed to read request body: %v", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse JSON request
	var req models.ImageAnalyzerRequest
	if err := json.Unmarshal(body, &req); err != nil {
		log.Printf("ERROR: Failed to parse JSON: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Log the request
	logVisionRequest(r, deviceEUI, authToken, &req, body)

	// Validate request has image
	if req.Img == "" {
		log.Printf("ERROR: No image provided in request")
		http.Error(w, "No image provided", http.StatusBadRequest)
		return
	}

	// Use default prompt if none provided
	prompt := req.Prompt
	if prompt == "" {
		prompt = "what's in the picture?"
	}

	// Step 1: Analyze image with LLaVA
	log.Println("Step 1: Analyzing image with LLaVA...")
	analysis, err := analyzeImageWithLLaVA(req.Img, prompt)
	if err != nil {
		log.Printf("ERROR: Image analysis failed: %v", err)
		http.Error(w, "Image analysis failed", http.StatusInternalServerError)
		return
	}
	log.Printf("Analysis result: '%s'", analysis)

	// Step 2: Determine if event should be triggered
	// For monitoring mode (type=1), we need to determine if the condition is met
	state := 0 // Default: no event

	if req.Type == 1 {
		// MONITORING mode - analyze if the prompt condition is met
		// Look for positive indicators in the analysis response
		analysisLower := strings.ToLower(analysis)

		// Check if LLaVA gave a positive response
		isPositive := strings.Contains(analysisLower, "yes") ||
			strings.Contains(analysisLower, "there is") ||
			strings.Contains(analysisLower, "i can see") ||
			strings.Contains(analysisLower, "visible") ||
			strings.Contains(analysisLower, "present") ||
			strings.Contains(analysisLower, "wearing") ||
			strings.Contains(analysisLower, "detected")

		isNegative := strings.Contains(analysisLower, "no") ||
			strings.Contains(analysisLower, "not") ||
			strings.Contains(analysisLower, "cannot") ||
			strings.Contains(analysisLower, "can't") ||
			strings.Contains(analysisLower, "unable")

		if isPositive && !isNegative {
			state = 1 // Event detected!
			log.Printf("MONITORING MODE: Event detected! Analysis indicates positive match.")
		} else {
			log.Printf("MONITORING MODE: No event detected. Analysis indicates no match or negative.")
		}
	} else {
		// RECOGNIZE mode - just analysis, no event triggering
		log.Printf("RECOGNIZE MODE: Analysis complete, no event triggering.")
	}

	// Step 3: Optionally synthesize speech with Piper TTS
	var audioBase64 *string
	if req.AudioTxt != "" {
		log.Println("Step 3: Synthesizing speech with Piper TTS...")
		audioData, err := synthesizeSpeech(req.AudioTxt)
		if err != nil {
			log.Printf("WARNING: Speech synthesis failed: %v (continuing without audio)", err)
		} else {
			audioB64 := base64.StdEncoding.EncodeToString(audioData)
			audioBase64 = &audioB64
			log.Printf("Generated audio: %d bytes WAV, %d bytes base64", len(audioData), len(audioB64))
		}
	}

	// Prepare response
	response := models.ImageAnalyzerResponse{
		Code: 200,
		Data: models.ImageAnalyzerResponseData{
			State: state,     // 0 = no event, 1 = event detected
			Type:  req.Type,  // Echo back the request type
			Audio: audioBase64,
			Img:   nil,       // No processed image to return
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)

	log.Printf("Vision analysis complete. State=%d, Analysis: %s", state, analysis)
}

func logVisionRequest(r *http.Request, deviceEUI, authToken string, req *models.ImageAnalyzerRequest, rawBody []byte) {
	log.Println("================================================================================")
	log.Println("IMAGE ANALYZER REQUEST RECEIVED")
	log.Println("================================================================================")
	log.Printf("Timestamp:   %s", time.Now().Format(time.RFC3339))
	log.Printf("Action:      %s %s", r.Method, r.URL.Path)
	if r.URL.RawQuery != "" {
		log.Printf("Query:       %s", r.URL.RawQuery)
	}
	log.Printf("Remote Addr: %s", r.RemoteAddr)

	// Log all headers
	log.Println("--------------------------------------------------------------------------------")
	log.Println("REQUEST HEADERS")
	log.Println("--------------------------------------------------------------------------------")
	for name, values := range r.Header {
		for _, value := range values {
			log.Printf("  %s: %s", name, value)
		}
	}

	// Log request details
	log.Println("--------------------------------------------------------------------------------")
	log.Println("REQUEST DETAILS")
	log.Println("--------------------------------------------------------------------------------")

	analyzerType := "MONITORING (1)"
	if req.Type == 0 {
		analyzerType = "RECOGNIZE (0)"
	}
	log.Printf("Type:        %s", analyzerType)

	if req.Prompt != "" {
		log.Printf("Prompt:      %s", req.Prompt)
	} else {
		log.Println("Prompt:      (empty)")
	}

	if req.AudioTxt != "" {
		log.Printf("Audio Text:  %s", req.AudioTxt)
	} else {
		log.Println("Audio Text:  (empty)")
	}

	imgLen := len(req.Img)
	if imgLen > 0 {
		log.Printf("Image:       %d bytes (base64-encoded JPEG)", imgLen)

		// Estimate decoded size (base64 is ~33% larger than binary)
		estimatedBytes := (imgLen * 3) / 4
		log.Printf("             ~%d KB decoded", estimatedBytes/1024)
	} else {
		log.Println("Image:       (empty)")
	}

	// Log raw JSON for debugging
	log.Println("--------------------------------------------------------------------------------")
	log.Println("RAW JSON REQUEST")
	log.Println("--------------------------------------------------------------------------------")

	// Pretty print JSON (but truncate the image field for readability)
	var prettyJSON map[string]interface{}
	if err := json.Unmarshal(rawBody, &prettyJSON); err == nil {
		// Truncate image for display
		if img, ok := prettyJSON["img"].(string); ok && len(img) > 100 {
			prettyJSON["img"] = fmt.Sprintf("%s... (%d bytes total)", img[:100], len(img))
		}

		if formatted, err := json.MarshalIndent(prettyJSON, "", "  "); err == nil {
			fmt.Println(string(formatted))
		}
	}

	log.Println("================================================================================")
	log.Println()
}

// analyzeImageWithLLaVA sends base64-encoded image to Ollama's LLaVA model for analysis
func analyzeImageWithLLaVA(imageBase64, prompt string) (string, error) {
	// Prepare request for Ollama LLaVA API
	requestBody := map[string]interface{}{
		"model":  cfg.AI.LLaVAModel,
		"prompt": prompt,
		"images": []string{imageBase64},
		"stream": false,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal LLaVA request: %w", err)
	}

	// Send request to Ollama
	ollamaURL := cfg.AI.OllamaURL + "/api/generate"
	resp, err := http.Post(ollamaURL, "application/json", bytes.NewReader(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to call LLaVA: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("LLaVA returned %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result struct {
		Response string `json:"response"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode LLaVA response: %w", err)
	}

	return result.Response, nil
}
