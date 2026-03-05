"""Shared data models used by api and worker"""


def create_response(message: str) -> str:
    """Create a response message"""
    return f"Response: {message}"
