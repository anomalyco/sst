import importlib
import json
import logging
import os
import sys
import traceback
import time
from urllib.request import Request, urlopen
from urllib.error import URLError


# Ensure Python logging goes to stdout so it appears in the dev TUI
logging.basicConfig(
    level=logging.INFO,
    stream=sys.stdout,
    force=True,
)


class LambdaContext:
    """Mimics the real AWS Lambda context object with snake_case attributes."""

    def __init__(self, headers):
        self.aws_request_id = headers.get("Lambda-Runtime-Aws-Request-Id", "")
        self.invoked_function_arn = headers.get("Lambda-Runtime-Invoked-Function-Arn", "")
        self.function_name = os.environ.get("AWS_LAMBDA_FUNCTION_NAME", "")
        self.function_version = os.environ.get("AWS_LAMBDA_FUNCTION_VERSION", "")
        self.memory_limit_in_mb = int(os.environ.get("AWS_LAMBDA_FUNCTION_MEMORY_SIZE", "128"))
        self.log_group_name = os.environ.get("AWS_LAMBDA_LOG_GROUP_NAME", "")
        self.log_stream_name = os.environ.get("AWS_LAMBDA_LOG_STREAM_NAME", "")
        self._deadline_ms = int(headers.get("Lambda-Runtime-Deadline-Ms", "0"))

    def get_remaining_time_in_millis(self):
        return max(self._deadline_ms - int(time.time() * 1000), 0)


def _post(url, body):
    """POST JSON to a URL using stdlib."""
    data = json.dumps(body).encode()
    req = Request(url, data=data, headers={"Content-Type": "application/json"}, method="POST")
    urlopen(req)


def report_error(ex, context=None):
    """Report an error back to the Lambda runtime API."""
    error_response = {
        "errorType": "Error",
        "errorMessage": str(ex),
        "trace": traceback.format_exc().split("\n"),
    }
    if context is None:
        endpoint = f"{AWS_LAMBDA_RUNTIME_API}/runtime/init/error"
    else:
        endpoint = f"{AWS_LAMBDA_RUNTIME_API}/runtime/invocation/{context.aws_request_id}/error"
    _post(endpoint, error_response)


def log(message):
    print(message, flush=True)
    sys.stdout.flush()
    sys.stderr.flush()


def parse_handler_path(handler_path):
    """Parse a handler path like 'module.function' into (module_path, function_name)."""
    if "." not in handler_path:
        raise ImportError(f"Invalid handler format: {handler_path}. Expected 'module.function'")
    module_path, function_name = handler_path.rsplit(".", 1)
    python_module_path = module_path.replace("/", ".").replace("\\", ".").lstrip(".")
    return python_module_path, function_name


def get_handler_function(module, function_name):
    """Get and validate a callable function from a module."""
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
    """Resolve a handler by importing its module and returning the function."""
    python_module_path, function_name = parse_handler_path(handler_path)
    if 'PYTHONPATH' not in os.environ and artifact_dir not in sys.path:
        sys.path.insert(0, artifact_dir)
    module = importlib.import_module(python_module_path)
    return module, get_handler_function(module, function_name)


# --- Main ---

handler = sys.argv[1]
AWS_LAMBDA_RUNTIME_API = f"http://{os.environ['AWS_LAMBDA_RUNTIME_API']}/2018-06-01"
artifact_dir = os.getcwd()

try:
    module, handler_function = resolve_handler(handler, artifact_dir)
    log(f"Loaded {handler}")
except Exception as ex:
    python_module, _ = parse_handler_path(handler) if "." in handler else (handler, "")
    log(f"Failed to load handler: {handler}")
    log(f"  Error: {ex}")
    log(f"  Module: {python_module}")
    log(f"  Working dir: {artifact_dir}")
    log(f"  PYTHONPATH: {os.environ.get('PYTHONPATH', 'not set')}")
    log(f"  sys.path: {sys.path}")
    report_error(ex)
    sys.exit(1)

# Lambda event loop
while True:
    try:
        resp = urlopen(f"{AWS_LAMBDA_RUNTIME_API}/runtime/invocation/next")
        headers = resp.headers
        context = LambdaContext(headers)
        event = json.loads(resp.read())
    except Exception as ex:
        log(f"Error getting next invocation: {ex}")
        report_error(ex)
        continue

    try:
        result = handler_function(event, context)
    except Exception as ex:
        log(f"Error running handler: {ex}")
        report_error(ex, context)
        continue

    # Serialize once, retry the POST up to 3 times
    try:
        response_body = json.dumps(result)
    except Exception as ex:
        log(f"Error serializing response: {ex}")
        report_error(ex, context)
        continue

    response_url = f"{AWS_LAMBDA_RUNTIME_API}/runtime/invocation/{context.aws_request_id}/response"
    for attempt in range(3):
        try:
            req = Request(response_url, data=response_body.encode(), headers={"Content-Type": "application/json"}, method="POST")
            resp = urlopen(req)
            break
        except URLError:
            if attempt < 2:
                time.sleep(0.5)

    sys.stdout.flush()
    sys.stderr.flush()
