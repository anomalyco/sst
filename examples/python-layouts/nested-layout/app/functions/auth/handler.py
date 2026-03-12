from shared.utils import format_response


def main(event, context):
    return format_response(200, {"message": "hello from auth"})
