from mypackage.utils import json_response


def api_handler(event, context):
    return json_response(
        {
            "layout": "workspace",
            "package": "mypackage",
        }
    )


def worker_handler(event, context):
    return {
        "layout": "workspace",
        "job": event.get("job", "resize-image"),
        "status": "completed",
    }
