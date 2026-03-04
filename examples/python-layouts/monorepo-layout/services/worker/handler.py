"""
Worker service handler.

Demonstrates a background worker function in a monorepo layout.
This function is invoked directly (not via URL) and uses the shared
utility package from the workspace.
"""
import json
import logging
from typing import Any, Dict

from shared.utils import get_current_time

logger = logging.getLogger(__name__)


def main(event: Dict[str, Any], context: Any) -> Dict[str, Any]:
    """Main worker service handler."""
    try:
        task_type = event.get("taskType", "default")
        payload = event.get("payload", {})

        logger.info(f"Processing task: {task_type}")

        result = {
            "status": "completed",
            "taskType": task_type,
            "payload": payload,
            "timestamp": get_current_time().isoformat(),
        }

        logger.info(f"Task completed: {result}")
        return result

    except Exception as e:
        logger.error(f"Worker service error: {str(e)}")
        raise
