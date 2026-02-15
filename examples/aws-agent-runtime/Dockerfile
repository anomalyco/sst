# Modern AI Agent with Hono + AI SDK
FROM oven/bun:1 AS base
WORKDIR /app

# Install dependencies
COPY package.json bun.lockb* ./
RUN bun install --frozen-lockfile

# Copy application code
COPY . .

# Expose port (8080 required for HTTP protocol in Bedrock Agent Core)
EXPOSE 8080

# Start the agent
CMD ["bun", "run", "index.ts"]
