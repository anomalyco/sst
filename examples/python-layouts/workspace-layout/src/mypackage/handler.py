"""
Lambda handlers for the workspace layout example.
"""
import json
import logging
from typing import Any, Dict

import requests

from src.mypackage.utils import format_response, get_current_time

# Configure logging
logger = logging.getLogger(__name__)
logger.setLevel(logging.INFO)


def api_handler(event: Dict[str, Any], context: Any) -> Dict[str, Any]:
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
        # Handle both API Gateway and Lambda Function URL events
        http_method = event.get("httpMethod") or event.get("requestContext", {}).get("http", {}).get("method", "GET")
        path = event.get("path") or event.get("rawPath", "/")
        
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


def worker_handler(event: Dict[str, Any], context: Any) -> Dict[str, Any]:
    """
    Worker Lambda handler for background processing.
    
    Args:
        event: Lambda event data (SQS, EventBridge, etc.)
        context: Lambda context object
        
    Returns:
        Processing result
    """
    try:
        logger.info("Worker handler invoked")
        
        # Process event data
        records = event.get("Records", [])
        processed_count = 0
        
        for record in records:
            # Example: Process SQS message
            if "body" in record:
                message_body = json.loads(record["body"])
                logger.info(f"Processing message: {message_body}")
                
                # Simulate work
                result = process_message(message_body)
                logger.info(f"Message processed: {result}")
                processed_count += 1
        
        return {
            "statusCode": 200,
            "processedCount": processed_count,
            "timestamp": get_current_time()
        }
        
    except Exception as e:
        logger.error(f"Worker handler error: {str(e)}")
        raise  # Re-raise for SQS retry logic


def process_message(message: Dict[str, Any]) -> Dict[str, Any]:
    """
    Process a single message.
    
    Args:
        message: Message data to process
        
    Returns:
        Processing result
    """
    # Example processing logic
    message_type = message.get("type", "unknown")
    
    if message_type == "user_signup":
        return process_user_signup(message)
    elif message_type == "order_created":
        return process_order_created(message)
    else:
        logger.warning(f"Unknown message type: {message_type}")
        return {"status": "skipped", "reason": "unknown_type"}


def process_user_signup(message: Dict[str, Any]) -> Dict[str, Any]:
    """Process user signup message."""
    user_id = message.get("user_id")
    email = message.get("email")
    
    # Example: Send welcome email, create user profile, etc.
    logger.info(f"Processing user signup: {user_id} ({email})")
    
    return {
        "status": "completed",
        "action": "user_signup",
        "user_id": user_id
    }


def process_order_created(message: Dict[str, Any]) -> Dict[str, Any]:
    """Process order created message."""
    order_id = message.get("order_id")
    user_id = message.get("user_id")
    
    # Example: Update inventory, send confirmation, etc.
    logger.info(f"Processing order: {order_id} for user {user_id}")
    
    return {
        "status": "completed", 
        "action": "order_created",
        "order_id": order_id
    }