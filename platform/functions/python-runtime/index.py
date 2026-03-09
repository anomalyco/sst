import importlib
import json
import logging
import os
import sys
import traceback
import time
import requests


# Configure Python logging to output to stdout so it appears alongside print statements
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    stream=sys.stdout,
    force=True  # Override any existing logging configuration
)


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


def parse_handler_path(handler_path):
    """
    Parse and validate a handler path like 'module.function'.
    
    Returns:
        tuple: (python_module_path, function_name)
    """
    if "." not in handler_path:
        raise ImportError(f"Invalid handler format: {handler_path}. Expected 'module.function'")
    
    module_path, function_name = handler_path.rsplit(".", 1)
    
    # Convert file path to module path (replace / with .)
    python_module_path = module_path.replace("/", ".").replace("\\", ".").lstrip(".")
    
    return python_module_path, function_name


def get_handler_function(module, function_name):
    """
    Get and validate a callable function from a module.
    
    Returns:
        callable: The handler function
    """
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
    
    return handler_function


def resolve_handler(handler_path, artifact_dir):
    """
    Resolve a handler by importing its module and returning the function.
    
    Uses PYTHONPATH for modern layouts. For legacy layouts (no PYTHONPATH),
    adds the artifact directory to sys.path first.
    
    Args:
        handler_path: Handler path like 'services/api/handler.main'
        artifact_dir: Root directory of the Lambda artifact
        
    Returns:
        tuple: (module, function)
    """
    python_module_path, function_name = parse_handler_path(handler_path)
    
    # For legacy layouts without PYTHONPATH, add artifact dir to sys.path
    if 'PYTHONPATH' not in os.environ and artifact_dir not in sys.path:
        sys.path.insert(0, artifact_dir)
    
    module = importlib.import_module(python_module_path)
    return module, get_handler_function(module, function_name)


# Parse the handler from command-line arguments
handler = sys.argv[1]  # Expecting the format 'module.function'
AWS_LAMBDA_RUNTIME_API = f"http://{os.environ['AWS_LAMBDA_RUNTIME_API']}/2018-06-01"

# Get the current working directory (artifact directory)
artifact_dir = os.getcwd()

try:
    module, handler_function = resolve_handler(handler, artifact_dir)
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
