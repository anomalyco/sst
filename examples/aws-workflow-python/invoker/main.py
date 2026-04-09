import boto3
import json
from typing import Any, Dict

from sst import Resource

client = boto3.client("lambda")


def handler(_event: Dict[str, Any], _context: Any) -> str:
    client.invoke(
        FunctionName=Resource.Workflow.name,
        Qualifier=Resource.Workflow.qualifier,
        InvocationType="Event",
        Payload=json.dumps({"resolverUrl": Resource.Resolver.url}),
    )

    return "Workflow started. Check the logs for the callback URL."
