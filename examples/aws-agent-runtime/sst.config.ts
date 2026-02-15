/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS Agent Runtime
 *
 * Deploy an AI agent using AWS Bedrock Agent Core Runtime.
 *
 * This example shows how to deploy a containerized AI agent that can be invoked
 * via AWS Bedrock.
 *
 * The agent is built with a Dockerfile and runs locally during `sst dev` for fast iteration.
 */
export default $config({
  app(input) {
    return {
      name: "aws-agent-runtime",
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

    const agent = new sst.aws.AgentRuntime("MyAgent", {
      description: "AI agent for customer support",
      image: {
        context: ".",
        dockerfile: "Dockerfile",
      },
      dev: {
        command: "bun run dev",
        directory: ".",
      },
      environment: {
        PORT: "8080",
        MODEL_NAME: "us.amazon.nova-micro-v1:0",
        TEMPERATURE: "0.7",
      },
      protocolConfiguration: "HTTP",
      permissions: [
        {
          actions: ["bedrock:InvokeModel"],
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

