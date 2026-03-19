from shared.utils import format_response, get_current_time


def main(event, context):
    return format_response(200, {
        "service": "auth",
        "timestamp": get_current_time().isoformat(),
    })
