"""
Test the deployed Python HTTP Agent using boto3 bedrock-agentcore client

This tests the HTTP protocol agent (not MCP) with direct invocation.
Run with:
AGENT_ARN="" AWS_REGION="us-east-1" uv run test-deploy.py
"""

import uuid
import boto3
import json
import os

agent_arn = os.environ.get("AGENT_ARN", "")
region = os.environ.get("AWS_REGION", "us-east-1")

print("🤖 Testing Python HTTP Agent with boto3")
print(f"📍 Agent ARN: {agent_arn}")
print(f"🌍 Region: {region}\n")

agentcore_client = boto3.client('bedrock-agentcore', region_name=region)

# Test 1: Letter counting (custom tool)
session_id = uuid.uuid4()
print("=" * 80)
print(f"📋 Test 1: Letter counting tool")
print(f"   Session ID: {session_id}")
print("=" * 80)

payload = {"prompt": "How many letter R's are in the word 'strawberry'? 🍓"}
print(f"💬 Request: {json.dumps(payload)}\n")

try:
    invoke_response = agentcore_client.invoke_agent_runtime(
        agentRuntimeArn=agent_arn,
        runtimeSessionId=str(session_id),
        payload=json.dumps(payload)
    )

    response_body = invoke_response['response'].read().decode()
    print(f"✅ Response received:")
    print(response_body)

except Exception as e:
    print(f"❌ Error: {e}")
    print(f"   Type: {type(e).__name__}")

# Test 2: Calculator tool
print("\n" + "=" * 80)
print(f"📋 Test 2: Calculator tool")
print(f"   Session ID: {session_id}")
print("=" * 80)

payload2 = {"prompt": "What is 3111696 divided by 74088?"}
print(f"💬 Request: {json.dumps(payload2)}\n")

try:
    invoke_response2 = agentcore_client.invoke_agent_runtime(
        agentRuntimeArn=agent_arn,
        runtimeSessionId=str(session_id),
        payload=json.dumps(payload2)
    )

    response_body2 = invoke_response2['response'].read().decode()
    print(f"✅ Response received:")
    print(response_body2)

except Exception as e:
    print(f"❌ Error: {e}")
    print(f"   Type: {type(e).__name__}")

# Test 3: Current time tool
print("\n" + "=" * 80)
print(f"📋 Test 3: Current time tool")
print("=" * 80)

session_id3 = uuid.uuid4()
payload3 = {"prompt": "What is the current time?"}
print(f"💬 Request: {json.dumps(payload3)}\n")

try:
    invoke_response3 = agentcore_client.invoke_agent_runtime(
        agentRuntimeArn=agent_arn,
        runtimeSessionId=str(session_id3),
        payload=json.dumps(payload3)
    )

    response_body3 = invoke_response3['response'].read().decode()
    print(f"✅ Response received:")
    print(response_body3)

except Exception as e:
    print(f"❌ Error: {e}")
    print(f"   Type: {type(e).__name__}")

# Test 4: Multi-tool strawberry challenge
print("\n" + "=" * 80)
print(f"📋 Test 4: Multi-tool Strawberry Challenge")
print("=" * 80)

session_id4 = uuid.uuid4()
payload4 = {
    "prompt": "I have 4 requests:\n\n1. What is the time right now?\n2. Calculate 3111696 / 74088\n3. Tell me how many letter R's are in the word \"strawberry\" 🍓\n4. What is 42 * 137?"
}
print(f"💬 Request: {json.dumps(payload4)}\n")

try:
    invoke_response4 = agentcore_client.invoke_agent_runtime(
        agentRuntimeArn=agent_arn,
        runtimeSessionId=str(session_id4),
        payload=json.dumps(payload4)
    )

    response_body4 = invoke_response4['response'].read().decode()
    print(f"✅ Response received:")
    print(response_body4)

except Exception as e:
    print(f"❌ Error: {e}")
    print(f"   Type: {type(e).__name__}")

print("\n" + "=" * 80)
print("🎉 All tests completed!")