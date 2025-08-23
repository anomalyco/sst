"""
API Lambda handler for nested layout example.
"""
import json
import logging
import time
import os
from typing import Any, Dict

from shared.utils import format_response, get_current_time

# Configure logging
logger = logging.getLogger(__name__)
logger.setLevel(logging.INFO)

# Track module load time
MODULE_LOAD_TIME = time.time()
logger.info(f"Module loaded at {MODULE_LOAD_TIME}")

# Track cold start
COLD_START = True


def main(event: Dict[str, Any], context: Any) -> Dict[str, Any]:
    """
    Main API Lambda handler.
    
    Args:
        event: Lambda event data
        context: Lambda context object
        
    Returns:
        API Gateway response
    """
    try:
        logger.info("API handler invoked")
        
        # Get request data
        # Handle both API Gateway and Lambda Function URL events
        http_method = event.get("httpMethod") or event.get("requestContext", {}).get("http", {}).get("method", "GET")
        path = event.get("path") or event.get("rawPath", "/")
        
        # Route requests
        if http_method == "GET" and path == "/health":
            return handle_health_check()
        elif http_method == "GET" and path.startswith("/api/"):
            return handle_api_request(path, event)
        elif http_method == "POST" and path == "/api/data":
            return handle_post_data(event)
        else:
            return format_response(404, {
                "error": "Not found",
                "path": path,
                "method": http_method
            })
            
    except Exception as e:
        logger.error(f"API handler error: {str(e)}")
        return format_response(500, {
            "error": "Internal server error",
            "message": str(e)
        })


def handle_health_check() -> Dict[str, Any]:
    """Handle health check requests."""
    return format_response(200, {
        "status": "healthy",
        "service": "nested-api",
        "timestamp": get_current_time(),
        "version": "1.0.0"
    })


def handle_api_request(path: str, event: Dict[str, Any]) -> Dict[str, Any]:
    """Handle API requests."""
    # Extract path parameters
    path_parts = path.split("/")
    
    if len(path_parts) >= 3 and path_parts[2] == "users":
        return handle_users_request(path_parts[3:] if len(path_parts) > 3 else [], event)
    elif len(path_parts) >= 3 and path_parts[2] == "orders":
        return handle_orders_request(path_parts[3:] if len(path_parts) > 3 else [], event)
    else:
        return format_response(400, {
            "error": "Invalid API endpoint",
            "path": path
        })


def handle_users_request(path_params: list, event: Dict[str, Any]) -> Dict[str, Any]:
    """Handle user-related requests."""
    if not path_params:
        # GET /api/users
        return format_response(200, {
            "users": [
                {"id": "1", "name": "John Doe", "email": "john@example.com"},
                {"id": "2", "name": "Jane Smith", "email": "jane@example.com"}
            ],
            "timestamp": get_current_time()
        })
    else:
        # GET /api/users/{id}
        user_id = path_params[0]
        return format_response(200, {
            "user": {
                "id": user_id,
                "name": f"User {user_id}",
                "email": f"user{user_id}@example.com"
            },
            "timestamp": get_current_time()
        })


def handle_orders_request(path_params: list, event: Dict[str, Any]) -> Dict[str, Any]:
    """Handle order-related requests."""
    if not path_params:
        # GET /api/orders
        return format_response(200, {
            "orders": [
                {"id": "order-1", "user_id": "1", "total": 99.99},
                {"id": "order-2", "user_id": "2", "total": 149.99}
            ],
            "timestamp": get_current_time()
        })
    else:
        # GET /api/orders/{id}
        order_id = path_params[0]
        return format_response(200, {
            "order": {
                "id": order_id,
                "user_id": "1",
                "total": 99.99,
                "status": "completed"
            },
            "timestamp": get_current_time()
        })


def handle_post_data(event: Dict[str, Any]) -> Dict[str, Any]:
    """Handle POST data requests."""
    try:
        body = json.loads(event.get("body", "{}"))
        
        # Process the data
        processed_data = {
            "received": body,
            "processed_at": get_current_time(),
            "status": "success"
        }
        
        return format_response(201, {
            "message": "Data processed successfully",
            "data": processed_data
        })
        
    except json.JSONDecodeError:
        return format_response(400, {
            "error": "Invalid JSON in request body"
        })