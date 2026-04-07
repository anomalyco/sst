import boto3
import json
from typing import Dict, Any
from sst import Resource

client = boto3.client("lambda")


def handler(event: Dict[str, Any], context: Any) -> Dict[str, str]:
    client.invoke(
        FunctionName=Resource.Workflow.name,
        Qualifier=Resource.Workflow.qualifier,
        InvocationType="Event",  # Asynchronous invocation
        Payload=json.dumps(event),
    )

    return {"message": "Workflow invoked successfully!"}
