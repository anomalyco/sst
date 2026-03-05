"""
Test: Workspace member importing a shared workspace package
This tests if cross-package imports work via a shared library
Also tests that worker-only dependencies (arrow) are bundled correctly
"""
from shared import models
import arrow


def lambda_handler(event, context):
    result = models.create_response("Worker using shared models")

    now = arrow.utcnow()
    timestamp = now.isoformat()

    return {
        "statusCode": 200,
        "body": result,
        "timestamp": timestamp,
        "arrow_version": arrow.__version__
    }
