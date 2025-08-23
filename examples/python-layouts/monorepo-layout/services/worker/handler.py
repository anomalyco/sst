"""
Worker service handler.
"""
import json
import logging
from typing import Any, Dict

from shared.utils import get_current_time

logger = logging.getLogger(__name__)


def main(event: Dict[str, Any], context: Any) -> Dict[str, Any]:
    """Main worker service handler."""
    try:
        logger.info("Worker service invoked")
        
        # Handle different event sources
        if "Records" in event:
            return handle_sqs_messages(event["Records"])
        elif "source" in event and event["source"] == "aws.events":
            return handle_scheduled_event(event)
        else:
            return handle_direct_invocation(event)
            
    except Exception as e:
        logger.error(f"Worker service error: {str(e)}")
        raise  # Re-raise for retry logic


def handle_sqs_messages(records: list) -> Dict[str, Any]:
    """Handle SQS message processing."""
    processed = 0
    failed = 0
    
    for record in records:
        try:
            message = json.loads(record["body"])
            logger.info(f"Processing message: {message}")
            
            # Process the message
            process_work_item(message)
            processed += 1
            
        except Exception as e:
            logger.error(f"Failed to process message: {str(e)}")
            failed += 1
    
    return {
        "statusCode": 200,
        "processed": processed,
        "failed": failed,
        "timestamp": get_current_time().isoformat()
    }


def handle_scheduled_event(event: Dict[str, Any]) -> Dict[str, Any]:
    """Handle scheduled events."""
    detail_type = event.get("detail-type", "Unknown")
    detail = event.get("detail", {})
    
    logger.info(f"Processing scheduled event: {detail_type}")
    
    # Process based on event type
    if detail_type == "Daily Cleanup":
        result = perform_cleanup()
    elif detail_type == "Report Generation":
        result = generate_report(detail)
    else:
        result = {"status": "skipped", "reason": f"Unknown event: {detail_type}"}
    
    return {
        "statusCode": 200,
        "eventType": detail_type,
        "result": result,
        "timestamp": get_current_time().isoformat()
    }


def handle_direct_invocation(event: Dict[str, Any]) -> Dict[str, Any]:
    """Handle direct Lambda invocations."""
    task_type = event.get("taskType", "default")
    payload = event.get("payload", {})
    
    logger.info(f"Processing direct task: {task_type}")
    
    if task_type == "data_processing":
        result = process_data(payload)
    elif task_type == "notification":
        result = send_notification(payload)
    else:
        result = {"status": "completed", "message": "Default task processed"}
    
    return {
        "statusCode": 200,
        "taskType": task_type,
        "result": result,
        "timestamp": get_current_time().isoformat()
    }


def process_work_item(item: Dict[str, Any]) -> Dict[str, Any]:
    """Process a work item."""
    item_type = item.get("type", "unknown")
    item_id = item.get("id", "unknown")
    
    logger.info(f"Processing work item: {item_type} ({item_id})")
    
    # Simulate processing
    import time
    time.sleep(0.1)
    
    return {
        "status": "completed",
        "item_type": item_type,
        "item_id": item_id
    }


def perform_cleanup() -> Dict[str, Any]:
    """Perform cleanup tasks."""
    logger.info("Performing daily cleanup")
    
    # Simulate cleanup work
    cleaned_items = 25
    
    return {
        "status": "completed",
        "items_cleaned": cleaned_items,
        "cleanup_type": "daily"
    }


def generate_report(detail: Dict[str, Any]) -> Dict[str, Any]:
    """Generate reports."""
    report_type = detail.get("reportType", "default")
    
    logger.info(f"Generating report: {report_type}")
    
    return {
        "status": "completed",
        "report_type": report_type,
        "report_url": f"s3://reports/{report_type}-{get_current_time().isoformat()}.pdf"
    }


def process_data(payload: Dict[str, Any]) -> Dict[str, Any]:
    """Process data."""
    data_source = payload.get("source", "unknown")
    record_count = payload.get("recordCount", 0)
    
    logger.info(f"Processing data from {data_source}: {record_count} records")
    
    return {
        "status": "completed",
        "source": data_source,
        "records_processed": record_count
    }


def send_notification(payload: Dict[str, Any]) -> Dict[str, Any]:
    """Send notifications."""
    notification_type = payload.get("type", "email")
    recipient = payload.get("recipient", "unknown")
    
    logger.info(f"Sending {notification_type} notification to {recipient}")
    
    return {
        "status": "sent",
        "type": notification_type,
        "recipient": recipient
    }