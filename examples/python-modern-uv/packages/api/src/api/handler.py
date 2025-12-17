"""
Test: Package with [tool.uv] package = true
This tests if the package is importable without explicit uv sync
"""
import requests
from api import models

def lambda_handler(event, context):
    # Test that package imports work
    data = models.create_response("API handler works!")
    
    return {
        "statusCode": 200,
        "body": data
    }
