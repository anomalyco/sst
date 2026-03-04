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
        path = event.get("rawPath") or event.get("path", "/")
        method = event.get("requestContext", {}).get("http", {}).get("method") or event.get("httpMethod", "GET")

        if method == "POST" and path == "/auth/login":
            return handle_login(event)
        elif method == "POST" and path == "/auth/verify":
            return handle_verify(event)
        else:
            return format_response(200, {
                "service": "auth",
                "status": "healthy",
                "timestamp": get_current_time().isoformat()
            })

    except Exception as e:
        logger.error(f"Auth service error: {str(e)}")
        return format_response(500, {"error": "Internal server error"})


def handle_login(event: Dict[str, Any]) -> Dict[str, Any]:
    """Handle login requests."""
    body = json.loads(event.get("body") or "{}")
    username = body.get("username")
    password = body.get("password")

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
    body = json.loads(event.get("body") or "{}")
    token = body.get("token")

    if token == "mock-jwt-token":
        return format_response(200, {
            "valid": True,
            "user": {"username": "admin", "role": "admin"}
        })
    else:
        return format_response(401, {"valid": False})
