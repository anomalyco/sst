"""
API service handler.
"""
import json
import logging
from typing import Any, Dict

from shared.utils import format_response, get_current_time

logger = logging.getLogger(__name__)


def main(event: Dict[str, Any], context: Any) -> Dict[str, Any]:
    """Main API service handler."""
    try:
        path = event.get("rawPath") or event.get("path", "/")
        method = event.get("requestContext", {}).get("http", {}).get("method") or event.get("httpMethod", "GET")

        if method == "GET" and path == "/api/health":
            return handle_health()
        elif method == "GET" and path.startswith("/api/users"):
            return handle_users(event)
        elif method == "POST" and path == "/api/data":
            return handle_data(event)
        else:
            return format_response(200, {
                "service": "api",
                "status": "healthy",
                "timestamp": get_current_time().isoformat()
            })

    except Exception as e:
        logger.error(f"API service error: {str(e)}")
        return format_response(500, {"error": "Internal server error"})


def handle_health() -> Dict[str, Any]:
    """Handle health check."""
    return format_response(200, {
        "status": "healthy",
        "service": "api",
        "timestamp": get_current_time().isoformat()
    })


def handle_users(event: Dict[str, Any]) -> Dict[str, Any]:
    """Handle user requests."""
    return format_response(200, {
        "users": [
            {"id": 1, "name": "John Doe"},
            {"id": 2, "name": "Jane Smith"}
        ]
    })


def handle_data(event: Dict[str, Any]) -> Dict[str, Any]:
    """Handle data requests."""
    body = json.loads(event.get("body") or "{}")
    return format_response(201, {
        "message": "Data processed",
        "data": body,
        "processed_at": get_current_time().isoformat()
    })
