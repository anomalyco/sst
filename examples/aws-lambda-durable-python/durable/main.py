from aws_durable_execution_sdk_python import (
    DurableContext,
    durable_execution,
    durable_step,
    StepContext
)
from aws_durable_execution_sdk_python.config import (
    WaitForCallbackConfig,
    Duration
)
from typing import Dict, Any
import logging
logging.basicConfig(level=logging.INFO)

@durable_step
def step1(step_context: StepContext) -> str:
    """First step of the durable execution."""
    step_context.logger.info("Executing step 1")
    return "Hello"


@durable_step
def step2(step_context: StepContext, step1_result: str) -> str:
    step_context.logger.info("Executing step 2")
    return f"{step1_result} World!"


@durable_execution
def handler(event: Dict[str, Any], context: DurableContext) -> Dict[str, Any]:
    step1_result = context.step(step1())
    
    callback_result = context.wait_for_callback(
        lambda callback_token, context: context.logger.info({"callback_token": callback_token}),
        name="callback",
        config=WaitForCallbackConfig(timeout=Duration.from_minutes(5))
    )
    
    step2_result = context.step(step2(step1_result))
    
    context.logger.info({"step1_result": step1_result, "step2_result": step2_result, "callback_result": callback_result})
    return {
        "step1": step1_result,
        "step2": step2_result,
        "callbackResult": callback_result
    }
