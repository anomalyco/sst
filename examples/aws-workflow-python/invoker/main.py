import boto3
import json
from typing import Any, Dict

from sst import Resource

client = boto3.client("lambda")


def handler(_event: Dict[str, Any], _context: Any) -> Dict[str, Any]:
    client.invoke(
        FunctionName=Resource.Workflow.name,
        Qualifier=Resource.Workflow.qualifier,
        InvocationType="Event",  # Asynchronous invocation
        Payload=json.dumps({"resolverUrl": Resource.Resolver.url}),
    )

    return {
        "statusCode": 200,
        "body": json.dumps(
            {
                "message": "Workflow started. Check the workflow logs for the callback URL."
            }
        ),
    }
