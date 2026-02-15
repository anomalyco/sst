"""
Minimal AI Agent with AWS Bedrock integration using Strands
Based on: https://strandsagents.com/latest/documentation/docs/user-guide/deploy/deploy_to_bedrock_agentcore/
"""

import os
import logging
from strands import Agent, tool
from strands.models import BedrockModel
from strands_tools import calculator, current_time
from bedrock_agentcore import BedrockAgentCoreApp

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Environment configuration
MODEL_ID = os.getenv("MODEL_NAME", "us.amazon.nova-micro-v1:0")
TEMPERATURE = float(os.getenv("TEMPERATURE", "0.7"))
AWS_REGION = os.getenv("AWS_REGION", "us-east-1")


# Define a custom tool using the @tool decorator
@tool
def letter_counter(word: str, letter: str) -> int:
    """
    Count occurrences of a specific letter in a word.

    Args:
        word (str): The input word to search in
        letter (str): The specific letter to count

    Returns:
        int: The number of occurrences of the letter in the word
    """
    if not isinstance(word, str) or not isinstance(letter, str):
        return 0

    if len(letter) != 1:
        raise ValueError("The 'letter' parameter must be a single character")

    return word.lower().count(letter.lower())


# Create a Bedrock model instance
bedrock_model = BedrockModel(
    model_id=MODEL_ID,
    temperature=TEMPERATURE,
    region_name=AWS_REGION,
)

# Create an agent with tools from the community-driven strands-tools package
# as well as our custom letter_counter tool
agent = Agent(
    model=bedrock_model,
    tools=[calculator, current_time, letter_counter]
)

# Create AgentCore app
app = BedrockAgentCoreApp()
logger.info("🚀 Starting Strands Agent with BedrockAgentCore...")
logger.info(f"   Model: {MODEL_ID}")
logger.info(f"   Region: {AWS_REGION}")
logger.info(f"   Temperature: {TEMPERATURE}")


@app.entrypoint
async def invoke(payload):
    """
    Agent entrypoint for AWS Bedrock AgentCore Runtime.
    Handles user requests and streams agent responses.
    """
    logger.info(f"📨 Received payload: {payload}")
    
    # Extract the user message from the payload
    user_message = payload.get("message") or payload.get("prompt", "Hello")
    logger.info(f"💬 User message: {user_message}")
    
    result = agent(user_message)
    return {"result": result.message}


if __name__ == "__main__":
    app.run()
