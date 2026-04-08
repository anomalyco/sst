import boto3
import json
from typing import Any, Dict


lambda_client = boto3.client("lambda")


def handler(event: Dict[str, Any], _context: Any) -> str:
    query_params = event.get("queryStringParameters") or {}
    token = query_params.get("token")

    if not token:
        return "Missing token in query parameters"

    lambda_client.send_durable_execution_callback_success(
        CallbackId=token, Result=json.dumps({"message": "Sent from the resolver."})
    )

    return "Workflow callback sent. Check the logs to see the workflow succeed."
