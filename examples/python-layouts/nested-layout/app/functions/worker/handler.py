def main(event, context):
    return {
        "status": "completed",
        "taskType": event.get("taskType", "default"),
    }
