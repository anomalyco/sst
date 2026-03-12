import boto3
import json
from typing import Dict, Any
from sst import Resource

client = boto3.client('lambda')


def handler(event: Dict[str, Any], context: Any) -> Dict[str, str]:
    client.invoke(
        FunctionName=Resource.Durable.name,
        Qualifier="$LATEST",
        InvocationType="Event",  # Asynchronous invocation
        Payload=json.dumps(event)
    )
    
    return {"message": "Durable function invoked successfully!"}
