"""
Shared utilities for the nested layout example.
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


def get_current_time() -> datetime:
    """
    Get current timestamp.
    
    Returns:
        Current datetime object
    """
    return datetime.utcnow()


def validate_email(email: str) -> bool:
    """
    Validate email format.
    
    Args:
        email: Email address to validate
        
    Returns:
        True if email is valid, False otherwise
    """
    import re
    pattern = r'^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$'
    return re.match(pattern, email) is not None


def sanitize_input(data: str) -> str:
    """
    Sanitize user input to prevent injection attacks.
    
    Args:
        data: Input string to sanitize
        
    Returns:
        Sanitized string
    """
    if not isinstance(data, str):
        return str(data)
    
    # Remove potentially dangerous characters
    dangerous_chars = ['<', '>', '"', "'", '&', '\x00']
    sanitized = data
    
    for char in dangerous_chars:
        sanitized = sanitized.replace(char, '')
    
    return sanitized.strip()


def paginate_results(items: list, page: int = 1, per_page: int = 10) -> Dict[str, Any]:
    """
    Paginate a list of items.
    
    Args:
        items: List of items to paginate
        page: Page number (1-based)
        per_page: Items per page
        
    Returns:
        Paginated results with metadata
    """
    total_items = len(items)
    total_pages = (total_items + per_page - 1) // per_page
    
    start_idx = (page - 1) * per_page
    end_idx = start_idx + per_page
    
    paginated_items = items[start_idx:end_idx]
    
    return {
        "items": paginated_items,
        "pagination": {
            "page": page,
            "per_page": per_page,
            "total_items": total_items,
            "total_pages": total_pages,
            "has_next": page < total_pages,
            "has_prev": page > 1
        }
    }


def generate_correlation_id() -> str:
    """
    Generate a correlation ID for request tracking.
    
    Returns:
        Unique correlation ID
    """
    import uuid
    return str(uuid.uuid4())


def log_request(event: Dict[str, Any], correlation_id: str = None) -> None:
    """
    Log incoming request details.
    
    Args:
        event: Lambda event data
        correlation_id: Optional correlation ID
    """
    import logging
    logger = logging.getLogger(__name__)
    
    log_data = {
        "correlation_id": correlation_id or generate_correlation_id(),
        "method": event.get("httpMethod"),
        "path": event.get("path"),
        "user_agent": event.get("headers", {}).get("User-Agent"),
        "source_ip": event.get("requestContext", {}).get("identity", {}).get("sourceIp"),
        "timestamp": get_current_time().isoformat()
    }
    
    logger.info(f"Request received: {json.dumps(log_data)}")


def handle_cors_preflight(event: Dict[str, Any]) -> Dict[str, Any]:
    """
    Handle CORS preflight requests.
    
    Args:
        event: Lambda event data
        
    Returns:
        CORS preflight response
    """
    if event.get("httpMethod") == "OPTIONS":
        return {
            "statusCode": 200,
            "headers": {
                "Access-Control-Allow-Origin": "*",
                "Access-Control-Allow-Methods": "GET, POST, PUT, DELETE, OPTIONS",
                "Access-Control-Allow-Headers": "Content-Type, Authorization, X-Requested-With",
                "Access-Control-Max-Age": "86400"
            },
            "body": ""
        }
    return None