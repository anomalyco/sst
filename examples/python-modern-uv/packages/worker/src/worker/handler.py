import json

import arrow
from shared import models


def lambda_handler(event, context):
    body = models.response("worker package")
    body["timestamp"] = arrow.utcnow().isoformat()

    return {
        "statusCode": 200,
        "body": json.dumps(body),
    }
