#!/bin/bash

# Test script for SenseCAP Watcher Local Server API endpoints

HOST="${1:-localhost:3000}"
TOKEN="${2:-my-test-token}"

echo "================================================================================"
echo "Testing SenseCAP Watcher Local Server"
echo "================================================================================"
echo "Host:  $HOST"
echo "Token: $TOKEN"
echo ""

# Test health endpoint
echo "1. Testing Health Endpoint"
echo "--------------------------------------------------------------------------------"
curl -s -X GET "http://$HOST/health" | jq '.'
echo ""
echo ""

# Test notification endpoint with object detection
echo "2. Testing Notification Endpoint (Object Detection)"
echo "--------------------------------------------------------------------------------"
curl -s -X POST "http://$HOST/v1/notification/event" \
  -H "Content-Type: application/json" \
  -H "Authorization: $TOKEN" \
  -H "API-OBITER-DEVICE-EUI: 2CF7F1C04430000C" \
  -d '{
    "requestId": "550e8400-e29b-41d4-a716-446655440000",
    "deviceEui": "2CF7F1C04430000C",
    "events": {
      "timestamp": 1704067200000,
      "text": "Motion detected in hallway",
      "img": "/9j/4AAQSkZJRgABAQEAYABgAAD...",
      "data": {
        "inference": {
          "boxes": [
            [120, 80, 200, 300, 95, 0],
            [350, 100, 150, 250, 87, 1]
          ],
          "classes_name": ["person", "car", "dog"]
        },
        "sensor": {
          "temperature": 23.5,
          "humidity": 65,
          "CO2": 450
        }
      }
    }
  }' | jq '.'
echo ""
echo ""

# Test notification endpoint with classification
echo "3. Testing Notification Endpoint (Classification)"
echo "--------------------------------------------------------------------------------"
curl -s -X POST "http://$HOST/v1/notification/event" \
  -H "Content-Type: application/json" \
  -H "Authorization: $TOKEN" \
  -H "API-OBITER-DEVICE-EUI: 2CF7F1C04430000C" \
  -d '{
    "requestId": "550e8400-e29b-41d4-a716-446655440001",
    "deviceEui": "2CF7F1C04430000C",
    "events": {
      "timestamp": 1704067260000,
      "text": "Animal detected",
      "data": {
        "inference": {
          "classes": [
            [98, 2],
            [75, 1]
          ],
          "classes_name": ["background", "dog", "cat"]
        }
      }
    }
  }' | jq '.'
echo ""
echo ""

# Test vision endpoint - monitoring mode
echo "4. Testing Vision Endpoint (Monitoring Mode)"
echo "--------------------------------------------------------------------------------"
curl -s -X POST "http://$HOST/v1/watcher/vision" \
  -H "Content-Type: application/json" \
  -H "Authorization: $TOKEN" \
  -H "API-OBITER-DEVICE-EUI: 2CF7F1C04430000C" \
  -d '{
    "img": "/9j/4AAQSkZJRgABAQEAYABgAAD...(base64 image data here)...",
    "prompt": "Is there a person in the frame?",
    "audio_txt": "Check if anyone is in the living room",
    "type": 1
  }' | jq '.'
echo ""
echo ""

# Test vision endpoint - recognize mode
echo "5. Testing Vision Endpoint (Recognize Mode)"
echo "--------------------------------------------------------------------------------"
curl -s -X POST "http://$HOST/v1/watcher/vision" \
  -H "Content-Type: application/json" \
  -H "Authorization: $TOKEN" \
  -H "API-OBITER-DEVICE-EUI: 2CF7F1C04430000C" \
  -d '{
    "img": "/9j/4AAQSkZJRgABAQEAYABgAAD...",
    "prompt": "What objects are visible?",
    "audio_txt": "",
    "type": 0
  }' | jq '.'
echo ""
echo ""

# Test missing authentication
echo "6. Testing Missing Authentication (should still work but log warning)"
echo "--------------------------------------------------------------------------------"
curl -s -X POST "http://$HOST/v1/notification/event" \
  -H "Content-Type: application/json" \
  -H "API-OBITER-DEVICE-EUI: 2CF7F1C04430000C" \
  -d '{
    "requestId": "550e8400-e29b-41d4-a716-446655440002",
    "deviceEui": "2CF7F1C04430000C",
    "events": {
      "timestamp": 1704067300000,
      "text": "Test without auth"
    }
  }' | jq '.'
echo ""
echo ""

echo "================================================================================"
echo "Testing Complete"
echo "================================================================================"
echo ""
echo "Check the server logs for detailed request information."
echo ""
