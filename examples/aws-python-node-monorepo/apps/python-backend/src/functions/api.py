from lib.ping import ping
from sst import Resource


def handler(event, context):
    response_code = ping()
    print(f"Response code: {response_code}")

    return {
        "statusCode": 200,
        "body": f"Hello from Python! - Linkable value: {Resource.MyLinkableValue.foo}",
    }
