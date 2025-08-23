"""
Worker Lambda handler for nested layout example.
"""
import json
import logging
from typing import Any, Dict

from shared.utils import get_current_time

# Configure logging
logger = logging.getLogger(__name__)
logger.setLevel(logging.INFO)


def main(event: Dict[str, Any], context: Any) -> Dict[str, Any]:
    """
    Main worker Lambda handler.
    
    Args:
        event: Lambda event data (SQS, EventBridge, etc.)
        context: Lambda context object
        
    Returns:
        Processing result
    """
    try:
        logger.info("Worker handler invoked")
        
        # Determine event source
        if "Records" in event:
            return handle_sqs_records(event["Records"])
        elif "source" in event and event["source"] == "aws.events":
            return handle_eventbridge_event(event)
        else:
            return handle_direct_invocation(event)
            
    except Exception as e:
        logger.error(f"Worker handler error: {str(e)}")
        raise  # Re-raise for retry logic


def handle_sqs_records(records: list) -> Dict[str, Any]:
    """Handle SQS message records."""
    processed_count = 0
    failed_count = 0
    
    for record in records:
        try:
            message_body = json.loads(record["body"])
            logger.info(f"Processing SQS message: {message_body}")
            
            # Process the message
            result = process_work_item(message_body)
            logger.info(f"SQS message processed: {result}")
            processed_count += 1
            
        except Exception as e:
            logger.error(f"Failed to process SQS message: {str(e)}")
            failed_count += 1
    
    return {
        "statusCode": 200,
        "processedCount": processed_count,
        "failedCount": failed_count,
        "timestamp": get_current_time(),
        "source": "sqs"
    }


def handle_eventbridge_event(event: Dict[str, Any]) -> Dict[str, Any]:
    """Handle EventBridge events."""
    detail_type = event.get("detail-type", "Unknown")
    detail = event.get("detail", {})
    
    logger.info(f"Processing EventBridge event: {detail_type}")
    
    # Process based on event type
    if detail_type == "User Signup":
        result = process_user_signup(detail)
    elif detail_type == "Order Created":
        result = process_order_created(detail)
    elif detail_type == "Scheduled Task":
        result = process_scheduled_task(detail)
    else:
        result = {"status": "skipped", "reason": f"Unknown event type: {detail_type}"}
    
    return {
        "statusCode": 200,
        "eventType": detail_type,
        "result": result,
        "timestamp": get_current_time(),
        "source": "eventbridge"
    }


def handle_direct_invocation(event: Dict[str, Any]) -> Dict[str, Any]:
    """Handle direct Lambda invocations."""
    task_type = event.get("taskType", "default")
    payload = event.get("payload", {})
    
    logger.info(f"Processing direct invocation: {task_type}")
    
    # Process based on task type
    if task_type == "data_processing":
        result = process_data_task(payload)
    elif task_type == "cleanup":
        result = process_cleanup_task(payload)
    elif task_type == "report_generation":
        result = process_report_task(payload)
    else:
        result = process_default_task(payload)
    
    return {
        "statusCode": 200,
        "taskType": task_type,
        "result": result,
        "timestamp": get_current_time(),
        "source": "direct"
    }


def process_work_item(item: Dict[str, Any]) -> Dict[str, Any]:
    """Process a generic work item."""
    item_type = item.get("type", "unknown")
    item_id = item.get("id", "unknown")
    
    # Simulate processing time
    import time
    time.sleep(0.1)
    
    return {
        "status": "completed",
        "item_type": item_type,
        "item_id": item_id,
        "processed_at": get_current_time()
    }


def process_user_signup(detail: Dict[str, Any]) -> Dict[str, Any]:
    """Process user signup events."""
    user_id = detail.get("userId")
    email = detail.get("email")
    
    # Example processing: send welcome email, create profile, etc.
    logger.info(f"Processing user signup: {user_id} ({email})")
    
    return {
        "status": "completed",
        "action": "user_signup",
        "user_id": user_id,
        "actions_taken": ["welcome_email_sent", "profile_created"]
    }


def process_order_created(detail: Dict[str, Any]) -> Dict[str, Any]:
    """Process order created events."""
    order_id = detail.get("orderId")
    user_id = detail.get("userId")
    total = detail.get("total")
    
    # Example processing: update inventory, send confirmation, etc.
    logger.info(f"Processing order: {order_id} for user {user_id} (${total})")
    
    return {
        "status": "completed",
        "action": "order_created",
        "order_id": order_id,
        "actions_taken": ["inventory_updated", "confirmation_sent"]
    }


def process_scheduled_task(detail: Dict[str, Any]) -> Dict[str, Any]:
    """Process scheduled tasks."""
    task_name = detail.get("taskName", "unknown")
    
    logger.info(f"Processing scheduled task: {task_name}")
    
    return {
        "status": "completed",
        "action": "scheduled_task",
        "task_name": task_name,
        "executed_at": get_current_time()
    }


def process_data_task(payload: Dict[str, Any]) -> Dict[str, Any]:
    """Process data processing tasks."""
    data_source = payload.get("source")
    record_count = payload.get("recordCount", 0)
    
    logger.info(f"Processing data from {data_source}: {record_count} records")
    
    return {
        "status": "completed",
        "records_processed": record_count,
        "source": data_source
    }


def process_cleanup_task(payload: Dict[str, Any]) -> Dict[str, Any]:
    """Process cleanup tasks."""
    target = payload.get("target")
    
    logger.info(f"Running cleanup for: {target}")
    
    return {
        "status": "completed",
        "cleanup_target": target,
        "items_cleaned": 42  # Simulated
    }


def process_report_task(payload: Dict[str, Any]) -> Dict[str, Any]:
    """Process report generation tasks."""
    report_type = payload.get("reportType")
    date_range = payload.get("dateRange")
    
    logger.info(f"Generating {report_type} report for {date_range}")
    
    return {
        "status": "completed",
        "report_type": report_type,
        "date_range": date_range,
        "report_url": f"s3://reports/{report_type}-{get_current_time()}.pdf"
    }


def process_default_task(payload: Dict[str, Any]) -> Dict[str, Any]:
    """Process default tasks."""
    logger.info("Processing default task")
    
    return {
        "status": "completed",
        "message": "Default task processed",
        "payload": payload
    }