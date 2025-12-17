"""
Test: Workspace member importing another workspace member
This tests if cross-package imports work with [tool.uv] package = true
"""
from api import models

def lambda_handler(event, context):
    # Test that workspace imports work
    result = models.create_response("Worker importing from API package")
    
    return {
        "statusCode": 200,
        "body": result
    }
