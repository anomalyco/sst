"""
Test: Package with [tool.uv] package = true
This tests if the package is importable and can use shared workspace deps
"""
import requests
from shared import models


def lambda_handler(event, context):
    data = models.create_response("API handler works!")

    return {
        "statusCode": 200,
        "body": data
    }
