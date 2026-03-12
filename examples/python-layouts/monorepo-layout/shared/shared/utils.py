import json
from datetime import datetime, timezone


def format_response(status_code, body):
    return {
        "statusCode": status_code,
        "headers": {
            "Content-Type": "application/json",
        },
        "body": json.dumps(body),
    }


def get_current_time():
    return datetime.now(timezone.utc)
