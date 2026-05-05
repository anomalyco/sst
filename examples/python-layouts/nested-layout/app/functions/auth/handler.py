from shared.utils import json_response


def main(event, context):
    return json_response({"layout": "nested", "service": "auth"})
