"""
Shared utilities for the flat layout example.
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
            "Access-Control-Allow-Origin": "*"
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