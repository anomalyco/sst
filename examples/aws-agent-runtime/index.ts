/**
 * AI Agent with AWS Bedrock integration using AI SDK
 * Uses AI SDK's Agent class for tool calling and conversation management
 */

import { Hono } from "hono";
import { createAmazonBedrock } from "@ai-sdk/amazon-bedrock";
import { fromNodeProviderChain } from "@aws-sdk/credential-providers";
import {
  Experimental_Agent as Agent,
  tool,
  stepCountIs,
  generateText,
} from "ai";
import { z } from "zod";
import { Resource } from "sst";

const app = new Hono();
// Port 8080 is required for HTTP protocol in Bedrock Agent Core Runtime
const PORT = parseInt(process.env.PORT || "8080", 10);

// Initialize Bedrock provider with credential provider chain
// This automatically uses AWS credentials from environment, SST, or AWS profiles
const bedrock = createAmazonBedrock({
  credentialProvider: fromNodeProviderChain(),
});

const MODEL_ID =
  process.env.MODEL_NAME || "us.amazon.nova-micro-v1:0";

// Define tools using AI SDK's tool format
// Reference: https://ai-sdk.dev/docs/reference/ai-sdk-core/tool#tool
const getCurrentTime = tool({
  description: "Get the current date and time",
  inputSchema: z.object({
    format: z.enum(["12h", "24h"]).describe("Time format preference"),
  }),
  execute: async ({ format }) => {
    console.log(`🔧 Executing tool: getCurrentTime`, { format });
    const now = new Date();
    return {
      date: now.toLocaleDateString(),
      time:
        format === "24h"
          ? now.toLocaleTimeString("en-US", { hour12: false })
          : now.toLocaleTimeString(),
      iso: now.toISOString(),
    };
  },
});

const calculator = tool({
  description: "Perform mathematical calculations",
  inputSchema: z.object({
    operation: z
      .enum(["add", "subtract", "multiply", "divide"])
      .describe("The mathematical operation to perform"),
    a: z.number().describe("First number"),
    b: z.number().describe("Second number"),
  }),
  execute: async ({ operation, a, b }) => {
    console.log(`🔧 Executing tool: calculator`, { operation, a, b });
    switch (operation) {
      case "add":
        return { result: a + b };
      case "subtract":
        return { result: a - b };
      case "multiply":
        return { result: a * b };
      case "divide":
        if (b === 0) return { error: "Division by zero" };
        return { result: a / b };
      default:
        return { error: "Invalid operation" };
    }
  },
});

const getWeather = tool({
  description: "Get weather information for a location",
  inputSchema: z.object({
    location: z.string().describe("City or location name"),
  }),
  execute: async ({ location }) => {
    console.log(`🔧 Executing tool: getWeather`, { location });
    return {
      location,
      temperature: Math.floor(Math.random() * 30) + 10,
      condition: ["sunny", "cloudy", "rainy"][Math.floor(Math.random() * 3)],
      humidity: Math.floor(Math.random() * 40) + 40,
    };
  },
});

const tools = {
  getCurrentTime,
  calculator,
  getWeather,
};

// System prompt for the agent (will be cached)
const SYSTEM_PROMPT = `You are a helpful AI assistant with access to tools. Use them when needed to help answer user questions.
  
Your approach:
- Be concise and helpful
- Use tools when they can provide accurate information
- Explain your reasoning when using tools
- Provide clear and actionable responses

Available tools:
- getCurrentTime: Get current date and time in 12h or 24h format
- calculator: Perform mathematical operations (add, subtract, multiply, divide)
- getWeather: Get weather information for any location`;

// Create the AI Agent using the Agent class
// Reference: https://ai-sdk.dev/docs/agents/building-agents
// Note: Agent class uses string system prompts. For caching, we'll use generateText with messages
const agent = new Agent({
  model: bedrock(MODEL_ID),
  system: SYSTEM_PROMPT,
  tools,
  stopWhen: stepCountIs(10), // Allow up to 10 steps for tool calling
});

// Request logging middleware
app.use("*", async (c, next) => {
  const timestamp = new Date().toISOString();
  console.log(`\n📥 [${timestamp}] ${c.req.method} ${c.req.path}`);
  await next();
  console.log(`   ✓ Response sent\n`);
});

// Agent handler function using AI SDK with tool calling
async function handleAgentRequest(c: any) {
  try {
    console.log("📨 Received request");
    console.log(`Method: ${c.req.method}, Path: ${c.req.path}`);
    console.log(`Content-Type: ${c.req.header("content-type")}`);

    const body = await c.req.json();
    console.log("Body keys:", Object.keys(body));
    console.log("Body preview:", JSON.stringify(body).substring(0, 200));

    const userMessage =
      body.message || body.prompt || body.inputText || "Hello";

    console.log(`💬 User message: "${userMessage.substring(0, 100)}..."`);

    // Use the Agent class's generate method - it handles the loop automatically
    const result = await agent.generate({
      prompt: userMessage,
    });

    console.log(`✅ Response generated (${result.text.length} chars)`);
    console.log(
      `📊 Steps: ${result.steps.length}, Tool calls: ${result.toolCalls.length}`
    );
    console.log(`💬 Final response: ${result.text.substring(0, 200)}...`);

    // Debug: Log all steps to understand the flow
    console.log(`\n🔍 DEBUG - Steps breakdown:`);
    result.steps.forEach((step, index) => {
      console.log(`  Step ${index + 1}:`, {
        text: step.text?.substring(0, 100) || "(no text)",
        toolCalls: step.toolCalls?.length || 0,
        toolResults: step.toolResults?.length || 0,
        finishReason: step.finishReason,
      });
    });
    console.log(``);

    // Debug: Log all steps to see what's happening
    console.log(
      "🔍 Debug - Steps:",
      result.steps.map((step, i) => ({
        step: i + 1,
        text: step.text?.substring(0, 100),
        toolCalls: step.toolCalls?.length,
        toolResults: step.toolResults?.length,
        finishReason: step.finishReason,
      }))
    );

    // Get the final response - if there are multiple steps, get the last meaningful text
    // Sometimes the model returns thinking in intermediate steps, we want the final answer
    let finalResponse = result.text;

    // If the response is just thinking tags and we have multiple steps, get the last step's text
    if (result.steps.length > 1 && result.text.includes("<thinking>")) {
      const lastStep = result.steps[result.steps.length - 1];
      if (lastStep.text && !lastStep.text.includes("<thinking>")) {
        finalResponse = lastStep.text;
      }
    }

    const responseJson = {
      status: "success",
      response: finalResponse,
      metadata: {
        model: MODEL_ID,
        temperature: process.env.TEMPERATURE || "0.7",
        steps: result.steps.length,
        toolCalls: result.toolCalls.length,
        timestamp: new Date().toISOString(),
        usage: result.usage,
      },
    };

    console.log(
      "📤 Sending response (status: success, response length:",
      result.text.length,
      "chars)"
    );
    return c.json(responseJson);
  } catch (error: any) {
    console.error("❌ Error:", error.message);
    console.error("Stack:", error.stack);
    return c.json(
      {
        status: "error",
        message: "Failed to generate AI response",
        error: error.message,
      },
      500
    );
  }
}

// Main agent endpoints
// /mcp for general use, /invocations for AWS Bedrock HTTP protocol
app.post("/mcp", handleAgentRequest);
app.post("/invocations", handleAgentRequest);

// Health check endpoint
app.get("/health", (c) => {
  return c.json({
    status: "healthy",
    timestamp: new Date().toISOString(),
    region: process.env.AWS_REGION || "us-east-1",
    model: MODEL_ID,
  });
});

// Root endpoint
app.get("/", (c) => {
  return c.json({
    name: "AI Customer Support Agent",
    version: "3.0.0",
    framework: "Hono + AI SDK Agent + Amazon Bedrock",
    status: "running",
    endpoints: {
      agent: "POST /mcp - Send a message to the AI agent",
      invocations: "POST /invocations - AWS Bedrock compatible endpoint",
      health: "GET /health - Check agent health",
    },
  });
});

// Start server
console.log("🤖 AI Agent starting...");
console.log(`📦 Framework: Hono + AI SDK Agent + Amazon Bedrock`);
console.log(`🧠 Model: ${MODEL_ID}`);
console.log(`🌍 Region: ${process.env.AWS_REGION || "us-east-1"}`);
console.log(`🌡️  Temperature: ${process.env.TEMPERATURE || "0.7"}`);
console.log(
  `🔧 Tools: ${Object.keys(tools).length} tools (${Object.keys(tools).join(
    ", "
  )})`
);
console.log(`🔁 Max steps: 10`);

// Export with explicit port (Hono + Bun pattern)
// export default {
//   port: PORT,
//   fetch: app.fetch,
// };

Bun.serve({
  port: PORT,
  fetch: app.fetch,
});