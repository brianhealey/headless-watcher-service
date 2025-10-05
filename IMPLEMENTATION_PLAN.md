# SenseCAP Local Server Implementation Plan

## Overview
This document outlines the implementation plan for completing the local server replacement for the SenseCAP Watcher cloud service.

## Current Status
- ✅ `/v2/watcher/talk/audio_stream` - Chat mode working (STT → Ollama → TTS)
- ⚠️ `/v1/watcher/vision` - Stub exists, needs LLaVA integration
- ⚠️ `/v2/watcher/talk/view_task_detail` - Not implemented
- ⚠️ `/v1/notification/event` - Stub exists, needs database storage
- ⚠️ Mode detection - Always returns mode=0 (chat), needs intelligence

## Phase 1: Vision Analysis (Current Priority)

### 1.1 Update Vision Handler (`handlers/vision.go`)
- [x] LLaVA 7B model installed
- [ ] Implement `analyzeImageWithLLaVA()` function
  - Decode base64 image from request
  - Send to Ollama LLaVA API with prompt
  - Return analysis text
- [ ] Optionally synthesize audio response using Piper TTS
- [ ] Return proper response format with analysis results

### 1.2 Response Format
```json
{
  "code": 200,
  "data": {
    "state": 0,  // 0=no event, 1=event detected
    "type": 0,   // Echo request type
    "audio": "base64_wav_audio",  // Optional
    "img": null  // Optional
  }
}
```

### 1.3 Testing
- Test with manual curl request
- Test with device if possible

## Phase 2: Task Mode Detection & Implementation

### 2.1 Function Selection in Audio Stream
Add pre-processing step to `AudioStreamHandler`:
```go
// Analyze transcription to determine mode
mode := determineMode(transcription)

if mode == 0 {
    // Existing chat flow
} else if mode == 1 || mode == 2 {
    // Task extraction flow
}
```

### 2.2 Implement Task Mode Prompts
- Function Selection Assistant prompt
- Trigger Condition Extraction prompt
- Word Matching Assistant (map to COCO classes)
- Headline Assistant (generate task name)

### 2.3 Task Flow Storage
Create database schema:
```sql
CREATE TABLE task_flows (
    id INTEGER PRIMARY KEY,
    device_eui TEXT,
    name TEXT,
    headline TEXT,
    trigger_condition TEXT,
    target_objects TEXT,  -- JSON array of COCO class IDs
    actions TEXT,  -- JSON array of actions
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);
```

## Phase 3: Task Detail View

### 3.1 Implement `/v2/watcher/talk/view_task_detail`
- Accept task ID or name in request
- Query database for task flow
- Return task configuration details

### 3.2 Request/Response Format
To be determined after analyzing firmware requests

## Phase 4: Notification Events

### 4.1 Update `/v1/notification/event`
Currently just logs and returns success. Need to:
- Store events in database
- Optional: Forward to webhook
- Optional: Trigger MQTT publish

### 4.2 Event Storage Schema
```sql
CREATE TABLE notification_events (
    id INTEGER PRIMARY KEY,
    request_id TEXT,
    device_eui TEXT,
    timestamp BIGINT,
    text TEXT,
    img TEXT,  -- Base64 image
    inference_data TEXT,  -- JSON
    sensor_data TEXT,  -- JSON
    created_at TIMESTAMP
);
```

## Phase 5: Enhanced Chat Assistant

### 5.1 Upgrade Prompt
Replace simple prompt with official Chat Assistant:
```
Your name is watcher, and you're a chatbot that can have a nice chat with users based on their input.
At the same time, you'll reject all answers to questions about terrorism, racism, yellow violence,
political sensitivity, LGBT issues, etc.
```

### 5.2 Add Session Management
- Store conversation history per session
- Implement context window (last N messages)
- Clear history on session timeout

## Testing Strategy

### Unit Tests
- Test each prompt with Ollama
- Test image analysis with sample images
- Test task extraction with various inputs

### Integration Tests
- End-to-end voice interaction test
- Image analysis workflow test
- Task creation and retrieval test

### Device Tests
- Test with actual SenseCAP Watcher device
- Verify audio quality
- Verify image analysis results
- Verify task flow execution

## Dependencies

### Software
- ✅ Ollama with llama3.1:8b-instruct-q4_1
- ✅ Ollama with llava:7b
- ✅ Whisper (base model)
- ✅ Piper TTS (en_US-lessac-medium)
- [ ] SQLite or PostgreSQL
- [ ] Optional: Redis for session storage

### Hardware Requirements
- ~10GB disk space for models
- 16GB+ RAM recommended for LLaVA
- GPU optional but recommended

## Timeline Estimate

- Phase 1 (Vision): 2-3 hours
- Phase 2 (Task Mode): 4-6 hours
- Phase 3 (Task Detail): 1-2 hours
- Phase 4 (Notifications): 2-3 hours
- Phase 5 (Chat Enhancement): 1-2 hours
- Testing & Debugging: 4-6 hours

**Total: 14-22 hours**

## Next Steps

1. ✅ Install LLaVA
2. [ ] Implement vision handler with LLaVA
3. [ ] Test vision endpoint
4. [ ] Implement mode detection
5. [ ] Add database for task flows
6. [ ] Implement task detail view
7. [ ] Enhance notification event storage
8. [ ] Comprehensive testing
