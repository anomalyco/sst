/**
 * Test using InvokeAgentRuntimeCommand from bedrock-agentcore
 * Run with:
 * AGENT_ARN="your-agent-arn" AWS_REGION="us-east-1" bun run test-deploy.ts
 */



import {
  BedrockAgentCoreClient,
  InvokeAgentRuntimeCommand,
} from "@aws-sdk/client-bedrock-agentcore";

async function testInvokeRuntime() {
  const agentArn = process.env.AGENT_ARN;
  
  if (!agentArn) {
    console.error("❌ AGENT_ARN environment variable not set");
    process.exit(1);
  }

  console.log("🤖 Testing with InvokeAgentRuntimeCommand");
  console.log(`📍 Agent ARN: ${agentArn}\n`);

  const client = new BedrockAgentCoreClient({
    region: process.env.AWS_REGION || "us-east-1",
  });

  const testMessage = {
    message: "What is 42 multiplied by 137?",
  };

  console.log(`💬 Request: ${JSON.stringify(testMessage)}\n`);

  try {
    // Session ID must be at least 33 characters
    const sessionId = `test-session-${Date.now()}-${Math.random().toString(36).substring(2)}`;
    console.log(`🆔 Session ID: ${sessionId} (${sessionId.length} chars)\n`);
    
    const command = new InvokeAgentRuntimeCommand({
      agentRuntimeArn: agentArn,
      runtimeSessionId: sessionId,
      payload: JSON.stringify(testMessage),
    });

    console.log("🚀 Invoking agent runtime...");
    const response = await client.send(command);

    console.log("✅ Response received!\n");
    console.log("📦 Response keys:", Object.keys(response));
    console.log("📦 $metadata:", response.$metadata);

    // Try to parse response
    if (response.response) {
      console.log("\n📝 Response info:");
      console.log("   Type:", typeof response.response);
      console.log("   Constructor:", response.response.constructor.name);
      
      // Check if it's a stream
      if (Symbol.asyncIterator in response.response) {
        console.log("   ⚡ It's an async iterable stream!\n");
        
        let fullResponse = "";
        for await (const chunk of response.response as any) {
          if (chunk instanceof Uint8Array) {
            fullResponse += new TextDecoder().decode(chunk);
          } else if (typeof chunk === "string") {
            fullResponse += chunk;
          } else {
            console.log("   Chunk:", chunk);
          }
        }
        
        console.log("📝 Full response body:");
        console.log(fullResponse);

        try {
          const jsonResponse = JSON.parse(fullResponse);
          
          if (jsonResponse.response) {
            console.log("\n💬 Agent says:");
            console.log("═".repeat(80));
            console.log(jsonResponse.response);
            console.log("═".repeat(80));
          }

          if (jsonResponse.metadata) {
            console.log("\n📊 Metadata:");
            console.log(`   Steps: ${jsonResponse.metadata.steps}`);
            console.log(`   Tool calls: ${jsonResponse.metadata.toolCalls}`);
          }
        } catch (e) {
          // Not JSON, just raw text
          console.log("\n💬 Agent response (raw):");
          console.log("═".repeat(80));
          console.log(fullResponse);
          console.log("═".repeat(80));
        }
      } else {
        // Try other methods
        console.log("   Raw value:", response.response);
      }
    }

    console.log("\n✅ Test completed!");
  } catch (error: any) {
    console.error("❌ Error:", error.message);
    console.error("\nFull error:", error);
    throw error;
  }
}

testInvokeRuntime().catch((error) => {
  console.error("Test failed:", error);
  process.exit(1);
});

