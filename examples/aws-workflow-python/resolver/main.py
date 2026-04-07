import boto3
import json
from typing import Any, Dict


lambda_client = boto3.client("lambda")


def handler(event: Dict[str, Any], _context: Any) -> Dict[str, Any]:
    query_params = event.get("queryStringParameters") or {}
    token = query_params.get("token")

    if not token:
        return {
            "statusCode": 400,
            "body": json.dumps({"message": "Missing token in query parameters"}),
        }

    lambda_client.send_durable_execution_callback_success(
        CallbackId=token, Result=json.dumps({"message": "Callback received"})
    )

    return {
        "statusCode": 200,
        "body": json.dumps({"message": "Workflow callback sent."}),
    }
