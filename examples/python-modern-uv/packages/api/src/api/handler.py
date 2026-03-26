import json

from shared import models


def lambda_handler(event, context):
    return {
        "statusCode": 200,
        "body": json.dumps(models.response("api package")),
    }
