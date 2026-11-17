# The python version to use is supplied as an arg from SST
ARG PYTHON_VERSION=3.11

# Use an official AWS Lambda base image for Python
FROM public.ecr.aws/lambda/python:${PYTHON_VERSION}

# Declare build args AFTER FROM to make them available in the build stage
# This arg will be passed from sst.config.ts via python.container.buildArgs
ARG MY_BUILD_ARG="default_value"

# Set as ENV so it's available at runtime
ENV MY_BUILD_ARG=${MY_BUILD_ARG}

# Ensure git is installed so we can install git based dependencies (such as sst)
RUN dnf update -y && \
    dnf install -y git gcc && \
    dnf clean all

# Install UV to manage your python runtime
COPY --from=ghcr.io/astral-sh/uv:latest /uv /bin/uv

# Install the dependencies to the lambda runtime
COPY requirements.txt ${LAMBDA_TASK_ROOT}/requirements.txt
RUN uv pip install -r requirements.txt --target ${LAMBDA_TASK_ROOT} --system

# Example: Install private packages using secrets (not persisted in image)
# RUN --mount=type=secret,id=CODEARTIFACT_AUTH_TOKEN \
#   CODEARTIFACT_AUTH_TOKEN=$(cat /run/secrets/CODEARTIFACT_AUTH_TOKEN) && \
#   pip install --extra-index-url https://aws:${CODEARTIFACT_AUTH_TOKEN}@my-domain.d.codeartifact.us-east-1.amazonaws.com/pypi/my-repo/simple/ my-private-package

# Copy the rest of the code
COPY . ${LAMBDA_TASK_ROOT}

# No need to configure the handler or entrypoint - SST will do that
