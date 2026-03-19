"""
Simple Lambda handler for flat layout example.
"""
import logging
from typing import Any, Dict

import requests
from utils import format_response, get_current_time

# Configure logging
logger = logging.getLogger(__name__)
logger.setLevel(logging.INFO)


def main(event: Dict[str, Any], context: Any) -> Dict[str, Any]:
    """
    Main Lambda handler.
    
    Args:
        event: Lambda event data
        context: Lambda context object
        
    Returns:
        API Gateway response
    """
    try:
        logger.info("Handler invoked")
        
        # Get request data
        # Handle both API Gateway and Lambda Function URL events
        http_method = event.get("httpMethod") or event.get("requestContext", {}).get("http", {}).get("method", "GET")
        path = event.get("path") or event.get("rawPath", "/")
        
        # Process request
        if http_method == "GET" and path == "/":
            return format_response(200, {
                "message": "Hello from flat layout!",
                "timestamp": get_current_time(),
                "layout": "flat"
            })
        
        elif http_method == "GET" and path == "/health":
            return format_response(200, {
                "status": "healthy",
                "timestamp": get_current_time()
            })
        
        elif http_method == "GET" and path == "/data":
            # Example external API call
            response = requests.get("https://httpbin.org/json")
            return format_response(200, {
                "data": response.json(),
                "timestamp": get_current_time()
            })
        
        else:
            return format_response(404, {
                "error": "Not found",
                "path": path,
                "method": http_method
            })
            
    except Exception as e:
        logger.error(f"Handler error: {str(e)}")
        return format_response(500, {
            "error": "Internal server error",
            "message": str(e)
        })


def worker(event: Dict[str, Any], context: Any) -> Dict[str, Any]:
    """
    Simple worker handler.
    
    Args:
        event: Lambda event data
        context: Lambda context object
        
    Returns:
        Processing result
    """
    try:
        logger.info("Worker invoked")
        
        # Simple processing
        message = event.get("message", "No message")
        
        # Simulate some work
        result = {
            "processed": True,
            "message": message,
            "timestamp": get_current_time()
        }
        
        logger.info(f"Work completed: {result}")
        return result
        
    except Exception as e:
        logger.error(f"Worker error: {str(e)}")
        raise