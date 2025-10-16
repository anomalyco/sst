#!/bin/bash

# Use PORT from environment or default to 8080 (HTTP protocol standard)
PORT="${PORT:-8080}"

echo "🤖 Testing Hono + Bedrock Agent locally (sst dev)"
echo "📍 Endpoint: http://localhost:${PORT}"
echo ""

# Test 1: Health check
echo "═══════════════════════════════════════════════════════════════════"
echo "Test 1: Health Check"
echo "═══════════════════════════════════════════════════════════════════"

curl -s http://localhost:${PORT}/health | jq '.'

echo ""
echo ""

# Test 2: Simple question
echo "═══════════════════════════════════════════════════════════════════"
echo "Test 2: Simple Question"
echo "═══════════════════════════════════════════════════════════════════"

curl -X POST http://localhost:${PORT}/mcp \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello! What can you help me with?"}' \
  -s | jq '.'

echo ""
echo ""

# Test 3: Using a tool (calculator)
echo "═══════════════════════════════════════════════════════════════════"
echo "Test 3: Calculator Tool"
echo "═══════════════════════════════════════════════════════════════════"

curl -X POST http://localhost:${PORT}/mcp \
  -H "Content-Type: application/json" \
  -d '{"message": "What is 42 multiplied by 137?"}' \
  -s | jq -r '.response' 2>/dev/null || echo "Failed to parse response"

echo ""
echo ""

# Test 4: Current time tool
echo "═══════════════════════════════════════════════════════════════════"
echo "Test 4: Current Time Tool"
echo "═══════════════════════════════════════════════════════════════════"

curl -X POST http://localhost:${PORT}/mcp \
  -H "Content-Type: application/json" \
  -d '{"message": "What time is it right now?"}' \
  -s | jq -r '.response' 2>/dev/null || echo "Failed to parse response"

echo ""
echo ""

# Test 5: Weather tool
echo "═══════════════════════════════════════════════════════════════════"
echo "Test 5: Weather Tool"
echo "═══════════════════════════════════════════════════════════════════"

curl -X POST http://localhost:${PORT}/mcp \
  -H "Content-Type: application/json" \
  -d '{"message": "What is the weather like in Paris?"}' \
  -s | jq -r '.response' 2>/dev/null || echo "Failed to parse response"

echo ""
echo ""

# Test 6: Root endpoint
echo "═══════════════════════════════════════════════════════════════════"
echo "Test 6: Root Endpoint (API Info)"
echo "═══════════════════════════════════════════════════════════════════"

curl -s http://localhost:${PORT}/ | jq '.'

echo ""
echo "✅ Local tests completed!"
