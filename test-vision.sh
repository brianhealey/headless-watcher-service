#!/bin/bash

# Test script for /v1/watcher/vision endpoint

echo "=== Testing Vision Endpoint ==="
echo

# Create a simple test image (1x1 red pixel PNG)
echo "Creating test image..."
TEST_IMAGE=$(python3 -c "
import base64
# 1x1 red pixel PNG
png_data = bytes.fromhex('89504e470d0a1a0a0000000d49484452000000010000000108020000009077531000000d6049444154789c62f8cf00000003010100189db28d000000004945445444ae426082')
print(base64.b64encode(png_data).decode())
")

echo "Test image created (1x1 red pixel PNG)"
echo

# Test 1: Basic vision analysis
echo "Test 1: Basic image analysis with default prompt"
echo "Request:"
cat <<EOF | jq .
{
  "type": 0,
  "prompt": "What color is this pixel?",
  "img": "${TEST_IMAGE:0:50}...",
  "audio_txt": ""
}
EOF

echo
echo "Response:"
curl -X POST http://localhost:8834/v1/watcher/vision \
  -H "Content-Type: application/json" \
  -H "API-OBITER-DEVICE-EUI: test-device-001" \
  -H "Authorization: Bearer test-token" \
  -d "{
    \"type\": 0,
    \"prompt\": \"What color is this pixel?\",
    \"img\": \"$TEST_IMAGE\",
    \"audio_txt\": \"\"
  }" | jq .

echo
echo "==================================="
echo

# Test 2: Vision with audio response
echo "Test 2: Image analysis with TTS audio response"
echo "Request:"
cat <<EOF | jq .
{
  "type": 0,
  "prompt": "Describe this image briefly",
  "img": "${TEST_IMAGE:0:50}...",
  "audio_txt": "This is a test image"
}
EOF

echo
echo "Response:"
curl -X POST http://localhost:8834/v1/watcher/vision \
  -H "Content-Type: application/json" \
  -H "API-OBITER-DEVICE-EUI: test-device-001" \
  -H "Authorization: Bearer test-token" \
  -d "{
    \"type\": 0,
    \"prompt\": \"Describe this image briefly\",
    \"img\": \"$TEST_IMAGE\",
    \"audio_txt\": \"This is a test image\"
  }" | jq .

echo
echo "==================================="
echo

# Test 3: Monitoring mode
echo "Test 3: Monitoring mode (type=1)"
echo "Request:"
cat <<EOF | jq .
{
  "type": 1,
  "prompt": "Is there any activity in this image?",
  "img": "${TEST_IMAGE:0:50}...",
  "audio_txt": ""
}
EOF

echo
echo "Response:"
curl -X POST http://localhost:8834/v1/watcher/vision \
  -H "Content-Type: application/json" \
  -H "API-OBITER-DEVICE-EUI: test-device-001" \
  -H "Authorization: Bearer test-token" \
  -d "{
    \"type\": 1,
    \"prompt\": \"Is there any activity in this image?\",
    \"img\": \"$TEST_IMAGE\",
    \"audio_txt\": \"\"
  }" | jq .

echo
echo "==================================="
echo "Tests complete!"
