import importlib
import json
import logging
import os
import sys
import traceback
import time
import requests
from pathlib import Path


# Configure Python logging to output to stdout so it appears alongside print statements
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    stream=sys.stdout,
    force=True  # Override any existing logging configuration
)

# Also ensure that all loggers use our configuration
root_logger = logging.getLogger()
root_logger.handlers.clear()
handler = logging.StreamHandler(sys.stdout)
handler.setFormatter(logging.Formatter('%(asctime)s - %(name)s - %(levelname)s - %(message)s'))
root_logger.addHandler(handler)
root_logger.setLevel(logging.INFO)


# Error handling function to report errors back to the Lambda runtime API
def report_error(ex, context=None):
    error_response = {
        "errorType": "Error",
        "errorMessage": str(ex),
        "trace": traceback.format_exc().split("\n"),
    }

    endpoint = (
        f"{AWS_LAMBDA_RUNTIME_API}/runtime/init/error"
        if context is None
        else f"{AWS_LAMBDA_RUNTIME_API}/runtime/invocation/{context['awsRequestId']}/error"
    )
    requests.post(
        endpoint,
        headers={"Content-Type": "application/json"},
        data=json.dumps(error_response),
    )


def log(message):
    print(message, flush=True)
    sys.stdout.flush()
    sys.stderr.flush()


def resolve_handler_simple(handler_path):
    """
    Simple handler resolution using standard Python imports.
    Works when PYTHONPATH is properly set (modern layouts).
    
    Args:
        handler_path: Handler path like 'services/api/handler.main'
        
    Returns:
        tuple: (module, function) or raises ImportError
    """
    # Parse handler path
    if "." not in handler_path:
        raise ImportError(f"Invalid handler format: {handler_path}. Expected 'module.function'")
    
    module_path, function_name = handler_path.rsplit(".", 1)
    
    # If handler_path is an absolute path, extract just the relative portion
    if os.path.isabs(module_path):
        # For absolute paths, we need to find the relative path from PYTHONPATH
        # Since we can't determine this reliably, just use the basename
        # This shouldn't happen in modern layouts, but handle it gracefully
        module_path = os.path.basename(module_path)
        log(f"Warning: Absolute path detected, using basename: {module_path}")
    
    # Convert file path to module path (replace / with .)
    # Remove any leading dots that might have been created
    python_module_path = module_path.replace("/", ".").replace("\\", ".").lstrip(".")
    
    # Simple import - Python will use PYTHONPATH
    module = importlib.import_module(python_module_path)
    
    # Get the function from the module
    if not hasattr(module, function_name):
        available_functions = [name for name in dir(module) if not name.startswith('_')]
        raise ImportError(
            f"Function '{function_name}' not found in module '{module.__name__}'. "
            f"Available functions: {available_functions}"
        )
    
    handler_function = getattr(module, function_name)
    if not callable(handler_function):
        raise ImportError(
            f"'{function_name}' is not a callable function in module '{module.__name__}'"
        )
    
    return module, handler_function


def resolve_handler_legacy(handler_path, artifact_dir):
    """
    Legacy handler resolution for flattened artifact directories.
    Used when files are copied and flattened (legacy layouts).
    
    Args:
        handler_path: Handler path like 'handler.main' or 'functions/user/handler.main'
        artifact_dir: Root directory of the Lambda artifact
        
    Returns:
        tuple: (module, function) or raises ImportError
    """
    # Parse handler path
    if "." not in handler_path:
        raise ImportError(f"Invalid handler format: {handler_path}. Expected 'module.function'")
    
    module_path, function_name = handler_path.rsplit(".", 1)
    
    # If handler_path is an absolute path, make it relative to artifact_dir
    if os.path.isabs(module_path):
        try:
            module_path = os.path.relpath(module_path, artifact_dir)
        except ValueError:
            # If paths are on different drives (Windows), just use basename
            module_path = os.path.basename(module_path)
    
    # Convert file path to module path (replace / with .)
    # Remove any leading dots that might have been created
    python_module_path = module_path.replace("/", ".").replace("\\", ".").lstrip(".")
    
    # Add artifact directory to sys.path
    if artifact_dir not in sys.path:
        sys.path.insert(0, artifact_dir)
    
    # Import the module
    module = importlib.import_module(python_module_path)
    
    # Get the function from the module
    if not hasattr(module, function_name):
        available_functions = [name for name in dir(module) if not name.startswith('_')]
        raise ImportError(
            f"Function '{function_name}' not found in module '{module.__name__}'. "
            f"Available functions: {available_functions}"
        )
    
    handler_function = getattr(module, function_name)
    if not callable(handler_function):
        raise ImportError(
            f"'{function_name}' is not a callable function in module '{module.__name__}'"
        )
    
    return module, handler_function


# Parse the handler from command-line arguments
handler = sys.argv[1]  # Expecting the format 'module.function'
AWS_LAMBDA_RUNTIME_API = f"http://{os.environ['AWS_LAMBDA_RUNTIME_API']}/2018-06-01"

# Get the current working directory (artifact directory)
artifact_dir = os.getcwd()

# Check if PYTHONPATH is set - if so, use simple import (modern layout)
# Otherwise use legacy import from artifact directory
pythonpath_set = 'PYTHONPATH' in os.environ

try:
    if pythonpath_set:
        module, handler_function = resolve_handler_simple(handler)
    else:
        module, handler_function = resolve_handler_legacy(handler, artifact_dir)
    
    log(f"Loaded {handler}")
    
except Exception as ex:
    # Parse handler to show what we tried to import
    module_path = handler.rsplit(".", 1)[0] if "." in handler else handler
    python_module = module_path.replace("/", ".").replace("\\", ".").lstrip(".")
    
    log(f"Failed to load handler: {handler}")
    log(f"  Error: {ex}")
    log(f"  Module: {python_module}")
    log(f"  Working dir: {artifact_dir}")
    log(f"  PYTHONPATH: {os.environ.get('PYTHONPATH', 'not set')}")
    log(f"  sys.path: {sys.path}")
    report_error(ex)
    sys.exit(1)

# Simulating Lambda's event loop
while True:
    try:
        # Get the next event to process
        response = requests.get(f"{AWS_LAMBDA_RUNTIME_API}/runtime/invocation/next")
        response.raise_for_status()

        context = {
            "awsRequestId": response.headers.get("Lambda-Runtime-Aws-Request-Id"),
            "invokedFunctionArn": response.headers.get(
                "Lambda-Runtime-Invoked-Function-Arn"
            ),
            "getRemainingTimeInMillis": lambda: max(
                int(response.headers.get("Lambda-Runtime-Deadline-Ms"))
                - int(time.time() * 1000),
                0,
            ),
            "functionName": os.environ.get("AWS_LAMBDA_FUNCTION_NAME"),
            "functionVersion": os.environ.get("AWS_LAMBDA_FUNCTION_VERSION"),
            "memoryLimitInMB": os.environ.get("AWS_LAMBDA_FUNCTION_MEMORY_SIZE"),
            "logGroupName": os.environ.get("AWS_LAMBDA_LOG_GROUP_NAME"),
            "logStreamName": os.environ.get("AWS_LAMBDA_LOG_STREAM_NAME"),
        }

        event = response.json()

    except Exception as ex:
        log(f"Error getting next invocation: {ex}")
        report_error(ex)
        continue

    # Run the handler function
    try:
        result = handler_function(event, context)
    except Exception as ex:
        log(f"Error running handler: {ex}")
        report_error(ex, context)
        continue

    # Send the response back to Lambda
    while True:
        try:
            requests.post(
                f"{AWS_LAMBDA_RUNTIME_API}/runtime/invocation/{context['awsRequestId']}/response",
                headers={"Content-Type": "application/json"},
                data=json.dumps(result),
            )
            break
        except Exception as _:
            time.sleep(0.5)
            continue

    sys.stdout.flush()
    sys.stderr.flush()
