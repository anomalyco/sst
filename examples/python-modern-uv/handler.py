"""
Test: Entry point in root with src/ layout
This tests if [tool.uv] package = true allows imports from src/
"""
from myapp import utils

def lambda_handler(event, context):
    result = utils.process_data({"test": "data"})
    return {
        "statusCode": 200,
        "body": f"Root handler with src/ imports: {result}"
    }
