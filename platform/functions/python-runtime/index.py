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


def discover_search_paths(artifact_dir):
    """
    Discover all possible Python module search paths in the artifact directory.
    
    Returns:
        List[str]: List of directories to search for Python modules
    """
    search_paths = [artifact_dir]
    
    # Common patterns in SST artifacts
    patterns = [
        "*/src",
        "*/src/*",
        "*",
        "*/app",
        "*/app/*",
        "*/services",
        "*/services/*",
        "*/functions",
        "*/functions/*"
    ]
    
    for pattern in patterns:
        if "*" in pattern:
            # Handle glob patterns
            parts = pattern.split("/")
            current_paths = [artifact_dir]
            
            for part in parts:
                if part == "*":
                    # Expand wildcard
                    new_paths = []
                    for path in current_paths:
                        try:
                            for item in os.listdir(path):
                                item_path = os.path.join(path, item)
                                if os.path.isdir(item_path):
                                    new_paths.append(item_path)
                        except (OSError, PermissionError):
                            continue
                    current_paths = new_paths
                else:
                    # Regular directory
                    current_paths = [os.path.join(path, part) for path in current_paths]
            
            search_paths.extend(current_paths)
        else:
            # Direct path
            search_paths.append(os.path.join(artifact_dir, pattern))
    
    # Filter to existing directories and remove duplicates
    valid_paths = []
    seen = set()
    for path in search_paths:
        if os.path.isdir(path) and path not in seen:
            valid_paths.append(path)
            seen.add(path)
    
    return valid_paths


def resolve_handler(handler_path, artifact_dir):
    """
    Resolve handler path to importable module and function.
    
    Args:
        handler_path: Handler path like 'src/mypackage/handler.api_handler'
        artifact_dir: Root directory of the Lambda artifact
        
    Returns:
        tuple: (module, function) or raises ImportError
    """
    log(f"Resolving handler: {handler_path}")
    log(f"Artifact directory: {artifact_dir}")
    
    # Parse handler path
    if "." not in handler_path:
        raise ImportError(f"Invalid handler format: {handler_path}. Expected 'module.function'")
    
    module_path, function_name = handler_path.rsplit(".", 1)
    log(f"Parsed - module_path: {module_path}, function_name: {function_name}")
    
    # If the module_path is absolute, make it relative to artifact_dir
    if os.path.isabs(module_path):
        # Remove the artifact_dir prefix to get relative path
        if module_path.startswith(artifact_dir):
            module_path = os.path.relpath(module_path, artifact_dir)
            log(f"Converted absolute path to relative: {module_path}")
        else:
            # This shouldn't happen, but handle it gracefully
            log(f"Warning: absolute path doesn't start with artifact_dir")
    
    # Convert file path to module path (replace / with .)
    python_module_path = module_path.replace("/", ".").replace("\\", ".")
    log(f"Python module path: {python_module_path}")
    
    # Discover search paths
    search_paths = discover_search_paths(artifact_dir)
    log(f"Search paths: {search_paths}")
    
    # Try different import strategies
    strategies = [
        # Strategy 1: Direct import with original path
        lambda: try_import_direct(python_module_path, search_paths),
        
        # Strategy 2: Import from each search path
        lambda: try_import_from_paths(module_path, search_paths),
        
        # Strategy 3: Try as relative import from common bases
        lambda: try_import_relative(module_path, search_paths),
        
        # Strategy 4: Search for module file directly
        lambda: try_import_by_file_search(module_path, search_paths)
    ]
    
    last_error = None
    for i, strategy in enumerate(strategies, 1):
        try:
            module = strategy()
            if module:
                log(f"Successfully imported module using strategy {i}")
                
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
                
        except Exception as e:
            last_error = e
            continue
    
    # All strategies failed
    raise ImportError(
        f"Could not import handler '{handler_path}'. "
        f"Searched in paths: {search_paths}. "
        f"Last error: {last_error}"
    )


def try_import_direct(python_module_path, search_paths):
    """Try importing the module path directly."""
    for search_path in search_paths:
        original_path = sys.path[:]
        try:
            sys.path.insert(0, search_path)
            return importlib.import_module(python_module_path)
        except ImportError:
            continue
        finally:
            sys.path[:] = original_path
    return None


def try_import_from_paths(module_path, search_paths):
    """Try importing by adding each search path and importing the basename."""
    module_parts = module_path.split("/")
    module_name = module_parts[-1]
    
    for search_path in search_paths:
        # Try importing just the module name
        original_path = sys.path[:]
        try:
            # Add the directory that should contain the module
            module_dir = search_path
            for part in module_parts[:-1]:
                module_dir = os.path.join(module_dir, part)
            
            if os.path.isdir(module_dir):
                sys.path.insert(0, module_dir)
                return importlib.import_module(module_name)
        except ImportError:
            continue
        finally:
            sys.path[:] = original_path
    return None


def try_import_relative(module_path, search_paths):
    """Try importing as relative module from different base paths."""
    module_parts = module_path.split("/")
    
    for search_path in search_paths:
        original_path = sys.path[:]
        try:
            sys.path.insert(0, search_path)
            
            # Try different combinations of the module path
            for i in range(len(module_parts)):
                try_path = ".".join(module_parts[i:])
                try:
                    return importlib.import_module(try_path)
                except ImportError:
                    continue
        except ImportError:
            continue
        finally:
            sys.path[:] = original_path
    return None


def try_import_by_file_search(module_path, search_paths):
    """Search for the module file directly and import it."""
    module_parts = module_path.split("/")
    module_name = module_parts[-1]
    
    # Look for the module file
    for search_path in search_paths:
        # Build the expected file path
        file_path = search_path
        for part in module_parts:
            file_path = os.path.join(file_path, part)
        
        # Try .py extension
        py_file = file_path + ".py"
        if os.path.isfile(py_file):
            # Found the file, try to import it
            original_path = sys.path[:]
            try:
                parent_dir = os.path.dirname(py_file)
                sys.path.insert(0, parent_dir)
                return importlib.import_module(module_name)
            except ImportError:
                continue
            finally:
                sys.path[:] = original_path
        
        # Try as package directory
        if os.path.isdir(file_path):
            init_file = os.path.join(file_path, "__init__.py")
            if os.path.isfile(init_file):
                original_path = sys.path[:]
                try:
                    parent_dir = os.path.dirname(file_path)
                    sys.path.insert(0, parent_dir)
                    return importlib.import_module(module_name)
                except ImportError:
                    continue
                finally:
                    sys.path[:] = original_path
    
    return None


# Parse the handler from command-line arguments
handler = sys.argv[1]  # Expecting the format 'module.function'
AWS_LAMBDA_RUNTIME_API = f"http://{os.environ['AWS_LAMBDA_RUNTIME_API']}/2018-06-01"

# Get the current working directory (artifact directory)
artifact_dir = os.getcwd()

try:
    # Resolve the handler using the new logic
    module, handler_function = resolve_handler(handler, artifact_dir)
    log(f"Successfully resolved handler: {handler}")
    
except Exception as ex:
    log(f"Failed to resolve handler: {ex}")
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
