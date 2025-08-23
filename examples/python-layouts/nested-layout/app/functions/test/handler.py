"""
Test Lambda handler to validate deployment capabilities.
"""
import json
import os
import sys
from pathlib import Path
from typing import Any, Dict

from shared.utils import format_response, get_current_time


def main(event: Dict[str, Any], context: Any) -> Dict[str, Any]:
    """
    Test Lambda handler that validates various deployment capabilities.
    """
    try:
        test_results = {}
        
        # Test 1: Module imports
        test_results["imports"] = test_imports()
        
        # Test 2: Static file loading
        test_results["static_files"] = test_static_files()
        
        # Test 3: Environment info
        test_results["environment"] = get_environment_info()
        
        # Test 4: Path resolution
        test_results["paths"] = test_path_resolution()
        
        return format_response(200, {
            "message": "Deployment capability tests completed",
            "timestamp": get_current_time().isoformat(),
            "tests": test_results,
            "overall_status": "success" if all(
                result.get("success", False) for result in test_results.values()
            ) else "partial_success"
        })
        
    except Exception as e:
        return format_response(500, {
            "error": "Test execution failed",
            "message": str(e),
            "timestamp": get_current_time().isoformat()
        })


def test_imports() -> Dict[str, Any]:
    """Test various import scenarios."""
    results = {
        "success": True,
        "details": {}
    }
    
    try:
        # Test absolute imports from shared
        from shared.utils import validate_email, sanitize_input
        results["details"]["shared_utils"] = "✅ Success"
        
        # Test nested imports
        from app.functions.auth.handler import main as auth_main
        results["details"]["nested_imports"] = "✅ Success"
        
        # Test function calls
        email_valid = validate_email("test@example.com")
        sanitized = sanitize_input("<script>alert('test')</script>")
        results["details"]["function_calls"] = f"✅ Success (email_valid={email_valid}, sanitized='{sanitized}')"
        
    except Exception as e:
        results["success"] = False
        results["details"]["error"] = f"❌ Import failed: {str(e)}"
    
    return results


def test_static_files() -> Dict[str, Any]:
    """Test static file loading with different strategies."""
    results = {
        "success": False,
        "details": {}
    }
    
    strategies = [
        ("pathlib_relative", lambda: Path(__file__).parent / "../../data/config.json"),
        ("os_path_relative", lambda: os.path.join(os.path.dirname(__file__), "../../data/config.json")),
        ("absolute_from_task", lambda: "/var/task/app/data/config.json"),
    ]
    
    for strategy_name, path_func in strategies:
        try:
            config_path = path_func()
            
            if os.path.exists(str(config_path)):
                with open(config_path, 'r') as f:
                    config = json.load(f)
                results["details"][strategy_name] = f"✅ Success: {config.get('app_name', 'unknown')}"
                results["success"] = True
            else:
                results["details"][strategy_name] = f"❌ File not found: {config_path}"
                
        except Exception as e:
            results["details"][strategy_name] = f"❌ Error: {str(e)}"
    
    return results


def get_environment_info() -> Dict[str, Any]:
    """Get environment information."""
    return {
        "success": True,
        "details": {
            "python_version": sys.version,
            "working_directory": os.getcwd(),
            "handler_file": __file__,
            "python_path_length": len(sys.path),
            "lambda_runtime_dir": "/var/runtime" if os.path.exists("/var/runtime") else "not_lambda",
            "task_dir_contents": list_directory("/var/task") if os.path.exists("/var/task") else "not_lambda"
        }
    }


def test_path_resolution() -> Dict[str, Any]:
    """Test different path resolution strategies."""
    results = {
        "success": True,
        "details": {}
    }
    
    try:
        # Test various path resolution methods
        results["details"]["__file__"] = __file__
        results["details"]["dirname(__file__)"] = os.path.dirname(__file__)
        results["details"]["abspath(__file__)"] = os.path.abspath(__file__)
        results["details"]["cwd"] = os.getcwd()
        
        # Test Path object
        file_path = Path(__file__)
        results["details"]["Path(__file__).parent"] = str(file_path.parent)
        results["details"]["Path(__file__).parent.parent"] = str(file_path.parent.parent)
        
        # Test if we can navigate to expected directories
        expected_dirs = ["app", "shared"]
        for dir_name in expected_dirs:
            dir_path = file_path.parent.parent.parent / dir_name
            results["details"][f"exists_{dir_name}"] = dir_path.exists()
        
    except Exception as e:
        results["success"] = False
        results["details"]["error"] = str(e)
    
    return results


def list_directory(path: str, max_items: int = 10) -> list:
    """List directory contents safely."""
    try:
        items = os.listdir(path)
        return sorted(items)[:max_items]
    except Exception:
        return []