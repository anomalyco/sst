# The python version to use is supplied as an arg from SST
ARG PYTHON_VERSION=3.11

# Use an official AWS Lambda base image for Python
FROM public.ecr.aws/lambda/python:${PYTHON_VERSION}

# # Ensure git is installed so we can install git based dependencies (such as sst)
RUN if command -v dnf > /dev/null 2>&1; then \
      dnf update -y && dnf install -y git gcc && dnf clean all; \
    elif command -v yum > /dev/null 2>&1; then \
      yum install -y git gcc && yum clean all; \
    fi

# Install UV to manage your python runtime
COPY --from=ghcr.io/astral-sh/uv:latest /uv /bin/uv

# Copy everything first so workspace packages (referenced as ./pkg in requirements.txt)
# are available during dependency installation
COPY . ${LAMBDA_TASK_ROOT}

# Install the dependencies to the lambda runtime
RUN uv pip install -r requirements.txt --target ${LAMBDA_TASK_ROOT} --system

# Perform any steps that you want here:
# Example: pre-bake in model weights from huggingface to image

# No need to configure the handler or entrypoint - SST will do that
