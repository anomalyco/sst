from typing import Any, Dict
from urllib.parse import urlencode

from aws_durable_execution_sdk_python import (
    DurableContext,
    StepContext,
    durable_execution,
    durable_step,
)
from aws_durable_execution_sdk_python.config import Duration, WaitForCallbackConfig


@durable_step
def start(step_context: StepContext) -> None:
    step_context.logger.info("Workflow started.")


@durable_step
def finish(step_context: StepContext, result: Any) -> None:
    step_context.logger.info(result)


@durable_execution
def handler(event: Dict[str, Any], context: DurableContext) -> None:
    context.step(start())

    def log_callback_url(token: str, step_context: StepContext) -> None:
        callback_url = f"{event['resolverUrl']}?{urlencode({'token': token})}"
        step_context.logger.info(callback_url)

    result = context.wait_for_callback(
        log_callback_url,
        name="callback",
        config=WaitForCallbackConfig(timeout=Duration.from_minutes(5)),
    )

    context.step(finish(result))
