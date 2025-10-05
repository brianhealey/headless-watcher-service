# SenseCAP Watcher System Prompts

This document contains the system prompts used by the SenseCAP Watcher cloud-based AI system (SenseCraft AI) for various features. These prompts are sent to the cloud LLM service and correspond to different device modes and activities.

## Voice Interaction Modes

The device firmware supports three voice interaction modes (defined in `app_voice_interaction.h`):
- **VI_MODE_CHAT (0)**: Conversational chat mode
- **VI_MODE_TASK (1)**: Task execution mode
- **VI_MODE_TASK_AUTO (2)**: Automatic task mode

## 1. Image Analysis Prompt

**Activity**: Camera/Vision Analysis
**API Endpoint**: `POST /v1/watcher/vision`
**Firmware Module**: `tf_module_img_analyzer.c`
**Cloud Host**: `https://sensecraft-aiservice-api.seeed.cc`

**Prompt**:
```
what's in the picture?
```

**Description**: Default prompt sent to vision AI when analyzing camera images. Used for general scene understanding and object detection.

**Request Format**:
- Multipart form data with image and prompt
- Parameters from `tf_module_img_analyzer_params`:
  - `type`: 0 (recognize) or 1 (monitoring)
  - `p_prompt`: Custom prompt or default "what's in the picture?"
  - `p_audio_txt`: Optional audio description

**Response Format**:
- JSON with image analysis results
- Optional audio response for TTS

---

## 2. Chat Assistant (VI_MODE_CHAT)

**Activity**: Voice Interaction - Chat Mode
**Mode**: VI_MODE_CHAT (0)
**API Endpoint**: `POST /v2/watcher/talk/audio_stream`
**Firmware Module**: `app_voice_interaction.c`
**Cloud Host**: Configured via device settings (can be local server)

**Prompt**:
```
Your name is watcher, and you're a chatbot that can have a nice chat with users based on their input.At the same time, you'll reject all answers to questions about terrorism, racism, yellow violence, political sensitivity, LGBT issues, etc.
```

**Description**: Used when device is in conversational chat mode. Processes transcribed speech and generates friendly responses while filtering inappropriate content.

**Request Headers**:
- `API-OBITER-DEVICE-EUI`: Device unique identifier
- `Session-Id`: Voice interaction session ID
- `Authorization`: Device auth token
- `Content-Type`: `application/octet-stream`

**Request Body**:
- Raw PCM audio data (16kHz, 16-bit, mono)
- May include 0xFF padding at beginning

**Response Format** (Multipart):
```json
{
  "code": 200,
  "data": {
    "mode": 0,
    "duration": <audio_duration_ms>,
    "stt_result": "<transcription>",
    "screen_text": "<llm_response>"
  }
}
```
Followed by:
- Boundary: `---sensecraftboundary---\n`
- Binary WAV audio data (TTS output)

---

## 3. Function Selection Assistant

**Activity**: Voice Command Mode Detection
**API Endpoint**: Internal to `/v2/watcher/talk/audio_stream` processing
**Firmware Module**: `app_voice_interaction.c`
**Usage**: Pre-processing step before main prompt

**Prompt**:
```
Your name is "watcher" and you are a function selection assistant. You analyse the user's input in relation to the definition of the "Mode List" and then select the most appropriate function from the list.
```

**Description**: Analyzes user voice input to determine whether to use chat mode, task mode, or automatic task mode. Routes requests to appropriate handler. This is called internally by the cloud service before selecting which prompt to use.

**Mode List Context**:
- VI_MODE_CHAT (0): General conversation
- VI_MODE_TASK (1): Execute specific task
- VI_MODE_TASK_AUTO (2): Automatic task execution

**Input**: Transcribed user speech
**Output**: Mode selection (0, 1, or 2) that determines response `data.mode` field

---

## 4. Trigger Condition Extraction (Task Flow)

**Activity**: Task Flow Condition Parsing
**API Endpoint**:
- `POST /v2/watcher/talk/audio_stream` (when mode=1 or mode=2)
- Also used via MQTT task configuration
**Firmware Modules**: `tf_module_alarm_trigger.c`, task flow engine
**Cloud Service**: Part of task flow creation workflow

**Prompt** (appears twice in different contexts):
```
You are a trigger condition extraction assistant, first you will remove the time, place, intervals, action after the trigger condition is triggered, device operations(such as turning on lights and playing sound.) from your input, and then you can present simple and clear conditions based on user input. playing sound.), which can be presented as simple and clear conditions based on user input. Just focus on the parts that are the object and the adverb or verb.output according to the "Output Function".
```

**Description**: Extracts core trigger conditions from natural language input for task automation. Removes temporal, spatial, and action components to focus on detection criteria. Used when creating task flows via voice or app.

**Example**:
- Input: "Turn on the light when a person enters the room after 6pm"
- Output: "person enters"

**Integration Points**:
- MQTT message: Task flow configuration in `order.value.tl.task_flow`
- Voice task mode: When `data.mode` = 1 or 2 in audio stream response

---

## 5. Word Matching Assistant (Object Detection)

**Activity**: Target Object Identification
**API Endpoint**: Internal to task flow configuration
**Firmware Module**: `tf_module_ai_camera.c`
**Usage**: Part of task creation workflow (cloud-side or app-side)

**Prompt**:
```
You are the word matching assistant. You start by analysing the "Scenario", extracting keywords,or static keywords where behaviours occur (animals are preferred, e.g. human) and matching them with the "Target Keyword Selection List", and finally output according to the "Output Function".
```

**Description**: Maps user-described scenarios to specific detection targets (person, cat, dog, car, etc.). Helps convert natural language to model class labels for the AI camera module.

**Target Keyword Selection List** (80 COCO dataset classes):
- person, bicycle, car, motorcycle, airplane, bus, train, truck, boat
- traffic light, fire hydrant, stop sign, parking meter, bench
- bird, cat, dog, horse, sheep, cow, elephant, bear, zebra, giraffe
- backpack, umbrella, handbag, tie, suitcase, frisbee, skis, snowboard
- sports ball, kite, baseball bat, baseball glove, skateboard, surfboard
- tennis racket, bottle, wine glass, cup, fork, knife, spoon, bowl
- banana, apple, sandwich, orange, broccoli, carrot, hot dog, pizza
- donut, cake, chair, couch, potted plant, bed, dining table, toilet
- tv, laptop, mouse, remote, keyboard, cell phone, microwave, oven
- toaster, sink, refrigerator, book, clock, vase, scissors, teddy bear
- hair drier, toothbrush

**Example**:
- Scenario: "Notify me when my cat jumps on the counter"
- Matched keyword: "cat"
- AI camera configured to detect class ID for "cat"

---

## 6. Headline Assistant

**Activity**: Task Summary Generation
**API Endpoint**: Internal to task flow configuration
**Firmware Module**: `app_taskflow.c`
**Usage**: Part of task creation workflow

**Prompt**:
```
You are a headline assistant that takes what a user enters and summarises it into a headline of six words or less:
```

**Description**: Creates concise summaries of user-configured tasks and automations for display on device screen and in task lists. Called when creating new task flows to generate user-friendly names.

**Example**:
- Input: "Send me a notification when someone opens the front door while I'm away from home"
- Output: "Front Door Open Alert"

**Storage**: Headline stored in task flow metadata, displayed in:
- Device UI task list
- Mobile app task management
- MQTT task flow messages

---

## 7. Task Detail View

**Activity**: Get detailed task flow information
**API Endpoint**: `POST /v2/watcher/talk/view_task_detail`
**Firmware Module**: `app_voice_interaction.c`
**Cloud Host**: Configured via device settings

**Description**: Retrieves detailed information about a configured task flow, including trigger conditions, actions, and current status.

**Request**: Likely includes task flow ID or name
**Response**: Task flow configuration details (format to be determined from cloud API)

---

## 8. HTTP Alarm Notification

**Activity**: Send alarm/event notifications
**API Endpoint**: `POST /v1/notification/event`
**Firmware Module**: `tf_module_http_alarm.c`
**Cloud Host**: `https://sensecraft-aiservice-api.seeed.cc` (or configured server)

**Description**: Sends alarm notifications when task flow triggers fire. Includes event details, captured images, and metadata.

**Request Format**:
- Event type
- Timestamp
- Device EUI
- Captured image (optional)
- Trigger condition details

**Response**: Acknowledgment of notification receipt

---

## Empty/Placeholder Prompts

The system also includes several instances of dynamic prompts that are filled at runtime:

- `prompt: ''` - Empty placeholder for custom prompts
- `prompt: prompt` - Variable that references a prompt parameter
- `prompt: input` - User input passed directly as the prompt

These are used in contexts where the prompt is dynamically constructed based on:
- User voice input
- Task configuration
- Real-time sensor data
- Image analysis context

---

## Prompt Usage Flow

### Voice Interaction Flow
1. User speaks → Audio captured by device
2. Device sends to `/v2/watcher/talk/audio_stream`
3. Server transcribes with Whisper (STT)
4. **Function Selection Assistant** determines mode (chat/task)
5. If chat mode → **Chat Assistant** prompt generates response
6. If task mode → **Trigger Condition Extraction** parses intent
7. Server synthesizes response with TTS
8. Device plays audio response

### Task Automation Setup Flow
1. User describes automation scenario (voice or app)
2. **Trigger Condition Extraction** isolates detection criteria
3. **Word Matching Assistant** maps to object classes
4. **Headline Assistant** creates task summary
5. Task stored in device with trigger conditions
6. AI camera monitors for trigger conditions
7. When triggered → execute actions (alarm, notification, etc.)

### Image Analysis Flow
1. Camera captures image
2. Device sends to AI vision endpoint
3. **Image Analysis Prompt** ("what's in the picture?") sent to LLM
4. LLM analyzes image and returns description
5. Result displayed on screen or used in task flow

---

## Notes

- All prompts reference "Output Function" which is defined in the cloud API response schema
- The "Mode List" maps to VI_MODE_CHAT (0), VI_MODE_TASK (1), VI_MODE_TASK_AUTO (2)
- The "Target Keyword Selection List" contains 80 COCO dataset object classes
- These prompts are used by the **cloud-based SenseCraft AI service**, not embedded in device firmware
- The firmware sends requests with these prompts to the cloud service for processing
- Some prompts have duplicates for different contexts (Trigger Condition Extraction appears twice)
- Dynamic prompts are constructed at runtime from user input or system variables

---

---

## API Endpoints Summary

### Local Server APIs (Required for Device Operation)

| Endpoint | Method | Purpose | Prompts Used | Status |
|----------|--------|---------|--------------|--------|
| `/v2/watcher/talk/audio_stream` | POST | Voice interaction (chat/task) | Chat Assistant, Function Selection, Trigger Extraction | ✅ Implemented |
| `/v1/watcher/vision` | POST | Image analysis with LLM | Image Analysis | ✅ Implemented (LLaVA 7B) |
| `/v2/watcher/talk/view_task_detail` | POST | View task flow details | None | ⚠️ Not implemented |
| `/v1/notification/event` | POST | Alarm/event notifications | None | ⚠️ Partial (logs only) |

### Cloud-Only APIs (SenseCraft AI Service)

These are used by the cloud service internally for prompt processing and are not called directly by the device:
- Function Selection Assistant (determines mode before response)
- Trigger Condition Extraction (parses task automation requests)
- Word Matching Assistant (maps natural language to object detection classes)
- Headline Assistant (generates task summaries)

---

## Implementation Notes for Local Server

When implementing a local server replacement for the cloud service:

### ✅ Currently Implemented

1. **Chat Mode** (`/v2/watcher/talk/audio_stream`)
   - STT: Whisper (base model)
   - LLM: Ollama (llama3.1:8b-instruct-q4_1)
   - TTS: Piper (en_US-lessac-medium)
   - Using simple chat prompt (can upgrade to official Chat Assistant prompt)
   - Returns multipart response: JSON + boundary + WAV audio
   - Correctly handles 0xFF padding in incoming PCM audio
   - Sets Content-Length header for complete audio download

2. **Image Analysis** (`/v1/watcher/vision`)
   - Vision LLM: Ollama LLaVA 7B
   - Accepts base64-encoded JPEG images
   - Supports custom prompts or defaults to "what's in the picture?"
   - Optional TTS audio response via Piper
   - Returns JSON with analysis results
   - Supports both RECOGNIZE (type=0) and MONITORING (type=1) modes

### ⚠️ To Be Implemented

2. **Task Mode** (same endpoint, mode=1/2)
   - Implement Function Selection Assistant to detect task requests
   - Use Trigger Condition Extraction prompt
   - Use Word Matching Assistant for object detection mapping
   - Return task configuration in response

3. **Image Analysis** (`/v1/watcher/vision`)
   - Requires vision-capable LLM (e.g., LLaVA, GPT-4 Vision)
   - Accept multipart image + prompt
   - Return analysis results with optional TTS audio

4. **Task Detail View** (`/v2/watcher/talk/view_task_detail`)
   - Store task flows in database
   - Return task configuration details
   - Support CRUD operations

5. **Alarm Notifications** (`/v1/notification/event`)
   - Receive and store alarm events
   - Optional: Forward to external notification services
   - Webhook integration for custom actions

### Prompt Enhancements

To make local server match cloud behavior:

```python
# In handlers/audio_stream.go - processWithOllama()
# Replace simple prompt with official Chat Assistant prompt:

prompt = """Your name is watcher, and you're a chatbot that can have a nice chat with users based on their input. At the same time, you'll reject all answers to questions about terrorism, racism, yellow violence, political sensitivity, LGBT issues, etc.

User said: "{transcription}"

Provide a brief, conversational response (1-2 sentences max)."""
```

### Architecture Recommendations

1. **Mode Detection**: Add pre-processing step to analyze transcription and determine if it's a chat request or task request
2. **Task Storage**: Use SQLite or PostgreSQL to store task flows
3. **Vision Model**: Deploy LLaVA locally or use OpenAI API for vision analysis
4. **Notification System**: Implement webhook system for alarm notifications
5. **MQTT Integration**: Support MQTT for task flow configuration (optional)
