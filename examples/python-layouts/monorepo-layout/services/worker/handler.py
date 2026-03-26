from shared.utils import worker_result


def main(event, context):
    return worker_result(event.get("job", "daily-report"))
