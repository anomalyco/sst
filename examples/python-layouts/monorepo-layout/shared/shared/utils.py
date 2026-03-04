"""
Shared utility functions for the monorepo.
"""
import json
from datetime import datetime, timezone
from typing import Any, Dict


def format_response(status_code: int, body: Dict[str, Any]) -> Dict[str, Any]:
    """
    Format a standard API Gateway response.
    
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
        "body": json.dumps(body)
    }


def get_current_time() -> datetime:
    """
    Get the current UTC time.
    
    Returns:
        Current datetime in UTC
    """
    return datetime.now(timezone.utc)


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


def sanitize_input(data: str, max_length: int = 1000) -> str:
    """
    Basic input sanitization.
    
    Args:
        data: Input string to sanitize
        max_length: Maximum allowed length
        
    Returns:
        Sanitized string
    """
    if not isinstance(data, str):
        return str(data)
    
    # Truncate if too long
    if len(data) > max_length:
        data = data[:max_length]
    
    # Basic sanitization (remove null bytes, control characters)
    data = data.replace('\x00', '').replace('\r', '').replace('\n', ' ')
    
    return data.strip()