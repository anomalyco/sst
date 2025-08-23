"""
Authentication service handler.
"""
import json
import logging
from typing import Any, Dict

from shared.utils import format_response, get_current_time

logger = logging.getLogger(__name__)


def main(event: Dict[str, Any], context: Any) -> Dict[str, Any]:
    """Main auth service handler."""
    try:
        # Handle both API Gateway and Lambda Function URL events
        path = event.get("path") or event.get("rawPath", "/")
        method = event.get("httpMethod") or event.get("requestContext", {}).get("http", {}).get("method", "GET")
        
        if method == "POST" and path == "/auth/login":
            return handle_login(event)
        elif method == "POST" and path == "/auth/verify":
            return handle_verify(event)
        else:
            return format_response(404, {"error": "Not found"})
            
    except Exception as e:
        logger.error(f"Auth service error: {str(e)}")
        return format_response(500, {"error": "Internal server error"})


def handle_login(event: Dict[str, Any]) -> Dict[str, Any]:
    """Handle login requests."""
    body = json.loads(event.get("body", "{}"))
    username = body.get("username")
    password = body.get("password")
    
    # Mock authentication
    if username == "admin" and password == "password":
        return format_response(200, {
            "token": "mock-jwt-token",
            "expires_in": 3600,
            "user": {"username": username, "role": "admin"}
        })
    else:
        return format_response(401, {"error": "Invalid credentials"})


def handle_verify(event: Dict[str, Any]) -> Dict[str, Any]:
    """Handle token verification."""
    body = json.loads(event.get("body", "{}"))
    token = body.get("token")
    
    # Mock verification
    if token == "mock-jwt-token":
        return format_response(200, {
            "valid": True,
            "user": {"username": "admin", "role": "admin"}
        })
    else:
        return format_response(401, {"valid": False})