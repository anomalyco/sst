"""
Shared utilities for the workspace example.
"""
import json
from datetime import datetime
from typing import Any, Dict


def format_response(status_code: int, body: Dict[str, Any]) -> Dict[str, Any]:
    """
    Format API Gateway response.
    
    Args:
        status_code: HTTP status code
        body: Response body data
        
    Returns:
        Formatted API Gateway response
    """
    return {
        "statusCode": status_code,
        "headers": {
            "Content-Type": "application/json",
            "Access-Control-Allow-Origin": "*",
            "Access-Control-Allow-Methods": "GET, POST, PUT, DELETE, OPTIONS",
            "Access-Control-Allow-Headers": "Content-Type, Authorization"
        },
        "body": json.dumps(body, default=str)
    }


def get_current_time() -> str:
    """
    Get current timestamp in ISO format.
    
    Returns:
        ISO formatted timestamp
    """
    return datetime.utcnow().isoformat() + "Z"


def validate_required_fields(data: Dict[str, Any], required_fields: list) -> list:
    """
    Validate that required fields are present in data.
    
    Args:
        data: Data dictionary to validate
        required_fields: List of required field names
        
    Returns:
        List of missing field names
    """
    missing_fields = []
    
    for field in required_fields:
        if field not in data or data[field] is None:
            missing_fields.append(field)
    
    return missing_fields


def safe_get(data: Dict[str, Any], key: str, default: Any = None) -> Any:
    """
    Safely get a value from a dictionary.
    
    Args:
        data: Dictionary to get value from
        key: Key to look up (supports dot notation)
        default: Default value if key not found
        
    Returns:
        Value from dictionary or default
    """
    if "." not in key:
        return data.get(key, default)
    
    # Handle nested keys with dot notation
    keys = key.split(".")
    current = data
    
    for k in keys:
        if isinstance(current, dict) and k in current:
            current = current[k]
        else:
            return default
    
    return current