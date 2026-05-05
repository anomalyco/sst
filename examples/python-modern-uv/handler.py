from myapp import utils


def lambda_handler(event, context):
    return {
        "statusCode": 200,
        "body": utils.greeting("root package"),
    }
