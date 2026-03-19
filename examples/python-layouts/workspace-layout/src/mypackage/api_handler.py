"""
API Lambda handler for the workspace layout example.
"""
import logging
from typing import Any, Dict

import requests

from src.mypackage.utils import format_response, get_current_time

# Configure logging
logger = logging.getLogger(__name__)
logger.setLevel(logging.INFO)


def main(event: Dict[str, Any], context: Any) -> Dict[str, Any]:
    """
    API Lambda handler.
    
    Args:
        event: Lambda event data
        context: Lambda context object
        
    Returns:
        API Gateway response
    """
    try:
        logger.info("API handler invoked")
        
        # Get request data
        http_method = event.get("httpMethod", "GET")
        path = event.get("path", "/")
        
        # Process request
        if http_method == "GET" and path == "/health":
            return format_response(200, {
                "status": "healthy",
                "timestamp": get_current_time(),
                "service": "workspace-example"
            })
        
        elif http_method == "GET" and path == "/external":
            # Example external API call
            response = requests.get("https://httpbin.org/json")
            return format_response(200, {
                "external_data": response.json(),
                "timestamp": get_current_time()
            })
        
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