from shared.utils import get_current_time


def main(event, context):
    return {
        "status": "completed",
        "taskType": event.get("taskType", "default"),
        "timestamp": get_current_time().isoformat(),
    }
