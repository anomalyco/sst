# The python version to use is supplied as an arg from SST
ARG PYTHON_VERSION=3.13

# Use uv's Python base image
FROM ghcr.io/astral-sh/uv:python${PYTHON_VERSION}-bookworm-slim

WORKDIR /app

# Install system dependencies for git
RUN apt-get update && \
    apt-get install -y git && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# All environment variables in one layer
ENV UV_SYSTEM_PYTHON=1 \
    UV_COMPILE_BYTECODE=1 \
    UV_NO_PROGRESS=1 \
    PYTHONUNBUFFERED=1 \
    DOCKER_CONTAINER=1 \

# Copy project files
COPY pyproject.toml ./
COPY src/ ./src/

# Install dependencies with uv
RUN uv pip install .

# Install OpenTelemetry
RUN uv pip install aws-opentelemetry-distro>=0.10.1

# Create non-root user
RUN useradd -m -u 1000 bedrock_agentcore
USER bedrock_agentcore

# Expose port (8000 required for MCP protocol in Bedrock Agent Core)
EXPOSE 8000

# Run with OpenTelemetry instrumentation
# The module path matches the package structure: src/agent/main.py
CMD ["opentelemetry-instrument", "python", "-m", "src.agent.main"]
