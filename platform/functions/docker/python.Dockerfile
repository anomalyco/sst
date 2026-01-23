# Specify the Python version as an ARG
ARG PYTHON_VERSION=3.11
ARG PYTHON_RUNTIME

# Use the Lambda Python base image
FROM public.ecr.aws/lambda/python:${PYTHON_VERSION}

# Copy the pre-built application code and dependencies into the final image
# The -src directory already contains all dependencies synced by SST
COPY . ${LAMBDA_TASK_ROOT}

# No need to configure the handler or entrypoint - SST will do that
