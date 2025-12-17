"""
Test: Workspace member importing another workspace member
This tests if cross-package imports work with [tool.uv] package = true
Also tests that worker-only dependencies (arrow) are bundled only with worker
"""
from api import models
import arrow

def lambda_handler(event, context):
    # Test that workspace imports work
    result = models.create_response("Worker importing from API package")
    
    # Test worker-only dependency (arrow) - should only be bundled with worker
    now = arrow.utcnow()
    timestamp = now.isoformat()
    
    return {
        "statusCode": 200,
        "body": result,
        "timestamp": timestamp,
        "arrow_version": arrow.__version__
    }
