def main(event, context):
    return {
        "layout": "nested",
        "job": event.get("job", "sync-users"),
        "status": "completed",
    }
