#!/bin/bash

# Test script for the agent locally using curl
# Run this script while 'sst dev' is running
# Usage: ./test-local-curl.sh

# Use HTTP_PORT from environment or default to 3000 (our test server default)
PORT="${HTTP_PORT:-8080}"

echo "🤖 Testing Python Strands Agent locally with curl"
echo "📍 Endpoint: http://localhost:${PORT}"
echo "═══════════════════════════════════════════════════════════════════"
echo ""

# Test 1: Simple greeting
echo "Test 1: Simple Greeting"
echo "───────────────────────"

(curl -X POST http://localhost:${PORT}/invocations \
  -H "Content-Type: application/json" \
  -d '{"prompt": "Hello! Can you introduce yourself?"}' \
  -s | jq '.' 2>/dev/null)

echo ""
echo ""
sleep 2

# Test 2: Calculator tool test
echo "Test 2: Calculator Tool"
echo "───────────────────────"

(curl -X POST http://localhost:${PORT}/invocations \
  -H "Content-Type: application/json" \
  -d '{"prompt": "What is 42 multiplied by 137?"}' \
  -s | jq '.' 2>/dev/null)

echo ""
echo ""
sleep 2

# Test 3: Current time tool
echo "Test 3: Current Time Tool"
echo "─────────────────────────"

(curl -X POST http://localhost:${PORT}/invocations \
  -H "Content-Type: application/json" \
  -d '{"prompt": "What time is it right now?"}' \
  -s | jq '.' 2>/dev/null)

echo ""
echo ""
sleep 2

# Test 4: Letter counter tool (custom tool)
echo "Test 4: Letter Counter Tool (Custom)"
echo "────────────────────────────────────"

(curl -X POST http://localhost:${PORT}/invocations \
  -H "Content-Type: application/json" \
  -d '{"prompt": "How many R letters are in the word strawberry? 🍓"}' \
  -s | jq '.' 2>/dev/null)

echo ""
echo ""
sleep 2

# Test 5: Multi-tool test (like the Python test)
echo "Test 5: Multi-Tool Test (Strawberry Challenge)"
echo "─────────────────────────────────────────────"

(curl -X POST http://localhost:${PORT}/invocations \
  -H "Content-Type: application/json" \
  -d '{"prompt": "I have 4 requests:\n\n1. What is the time right now?\n2. Calculate 3111696 / 74088\n3. Tell me how many letter Rs are in the word \"strawberry\" 🍓\n4. What is 42 * 137?"}' \
  -s | jq '.' 2>/dev/null)

echo ""
echo ""
sleep 2

# Test 6: AWS Bedrock compatible endpoint
echo "Test 6: AWS Bedrock Compatible Endpoint"
echo "───────────────────────────────────────"

(curl -X POST http://localhost:${PORT}/invocations \
  -H "Content-Type: application/json" \
  -d '{"prompt": "What is 15 + 27?"}' \
  -s | jq '.' 2>/dev/null)

echo ""
echo ""



echo "✅ Local curl tests completed!"
echo ""
echo "💡 Tips:"
echo "   - Make sure 'sst dev' is running for local testing"
echo "   - The agent uses Python + Strands with AWS Bedrock"
echo "   - Use /invocations endpoint for agent messages (Bedrock compatible)"