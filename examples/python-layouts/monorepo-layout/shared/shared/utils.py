import json


def api_response(service):
    return {
        "statusCode": 200,
        "headers": {"Content-Type": "application/json"},
        "body": json.dumps({"layout": "monorepo", "service": service}),
    }


def worker_result(job):
    return {"layout": "monorepo", "job": job, "status": "completed"}
