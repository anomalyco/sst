# Specify the Python version as an ARG
ARG PYTHON_VERSION=3.11

# Use the Lambda Python base image
FROM public.ecr.aws/lambda/python:${PYTHON_VERSION}

# Install uv for fast dependency installation
COPY --from=ghcr.io/astral-sh/uv:latest /uv /bin/uv

# Install git (needed for git-based dependencies like sst SDK)
# and gcc/build tools (needed for C-extension packages like cryptography, numpy)
# Python <=3.11 uses AL2 (yum), Python >=3.12 uses AL2023 (dnf)
RUN if command -v dnf > /dev/null 2>&1; then \
      dnf install -y git gcc python3-devel && dnf clean all; \
    elif command -v yum > /dev/null 2>&1; then \
      yum install -y git gcc python3-devel && yum clean all; \
    fi

# Copy everything first so workspace packages (referenced as ./pkg in requirements.txt)
# are available during dependency installation.
#
# NOTE: This copies source code before installing deps, which means any code change
# invalidates Docker's layer cache for the pip install step. This is a deliberate
# tradeoff — workspace packages must be present for `uv pip install` to resolve
# relative path dependencies (e.g. ./shared, ./core). Users who need better caching
# should provide a custom Dockerfile that copies requirements.txt first.
COPY . ${LAMBDA_TASK_ROOT}

# Install dependencies inside the container to ensure native binaries
# are compiled for the correct platform (Linux)
RUN uv pip install -r requirements.txt --target ${LAMBDA_TASK_ROOT} --system

# No need to configure the handler or entrypoint - SST will do that
