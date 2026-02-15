/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS Agent Runtime (Python + Strands)
 *
 * Deploy an AI agent using AWS Bedrock Agent Core Runtime with Python and Strands.
 *
 * SST uses [uv](https://docs.astral.sh/uv/) to manage Python dependencies, make sure 
 * you have it [installed](https://docs.astral.sh/uv/getting-started/installation/).
 *
 * This example shows how to deploy a containerized AI agent built with:
 * - Python 3.13 with uv for dependency management
 * - Strands framework for agentic workflows with simple @tool decorators
 * - AWS Bedrock for LLM inference (Claude models)
 * - Built-in tools from strands-tools (calculator, current_time)
 *
 * Based on patterns from: https://github.com/awslabs/amazon-bedrock-agentcore-samples
 *
 * The agent uses a [uv workspace](https://docs.astral.sh/uv/concepts/projects/workspaces/) 
 * for dependency management and is packaged in a Docker container.
 * 
 * ## Project Structure
 *
 * ```txt
 * ├── sst.config.ts
 * ├── pyproject.toml          # Root uv workspace
 * └── agent/                  # Agent workspace member
 *     ├── pyproject.toml      # Agent dependencies
 *     ├── Dockerfile          # Container with uv (AWS Lambda base)
 *     └── src/
 *         └── agent/
 *             ├── __init__.py
 *             └── main.py     # Agent with bedrock-agentcore entrypoint
 * ```
 *
 * The agent uses `bedrock-agentcore` package which provides the `@app.entrypoint` 
 * decorator for streaming responses:
 *
 * ```py title="agent/src/agent/main.py" {1,3}
 * from bedrock_agentcore import BedrockAgentCoreApp
 * 
 * app = BedrockAgentCoreApp()
 * 
 * @app.entrypoint
 * async def invoke(payload):
 *     user_message = payload["prompt"]
 *     async for event in agent.stream_async(user_message):
 *         if "data" in event:
 *             yield event["data"]
 * ```
 *
 * The Python version is set in the agent's `pyproject.toml` to match the Docker base image:
 *
 * ```toml title="agent/pyproject.toml"
 * requires-python = "==3.13.*"
 * ```
 */
export default $config({
  app(input) {
    return {
      name: "aws-agent-runtime-python",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
      providers: {
        aws: {
          region: "us-east-1",
        },
      },
    };
  },
  async run() {
    const agent = new sst.aws.AgentRuntime("PythonAgent", {
      description: "AI customer support agent using Strands and Bedrock",
      image: {
        context: "./agent",
        dockerfile: "Dockerfile",
      },
      dev: {
        command: "python src/agent/main.py",
        directory: "./agent",
      },
      environment: {
        MODEL_NAME: "us.amazon.nova-micro-v1:0",
        TEMPERATURE: "0.7",
      },
      protocolConfiguration: "HTTP",
      permissions: [
        {
          actions: [
            "bedrock:InvokeModel",
            "bedrock:InvokeModelWithResponseStream",
          ],
          resources: ["*"],
        },
      ],
    });

    return {
      agentArn: agent.agentRuntimeArn,
      agentId: agent.agentRuntimeId,
      workloadIdentityArn: agent.workloadIdentityArn,
    };
  },
});

