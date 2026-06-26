import os
import json


def handler(event, context):
    """
    Handler that returns the build argument value passed during Docker build.
    This demonstrates that buildArgs are correctly passed to the container build.
    """
    build_arg_value = os.environ.get("MY_BUILD_ARG", "NOT_SET")

    return {
        "statusCode": 200,
        "body": json.dumps({
            "message": "Hello from build_args example!",
            "build_arg_value": build_arg_value,
        }),
    }
