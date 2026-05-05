from utils import json_response


def main(event, context):
    return json_response(
        {
            "layout": "flat",
            "message": "Hello from the project root",
        }
    )


def worker(event, context):
    return {
        "layout": "flat",
        "job": event.get("job", "send-email"),
        "status": "completed",
    }
