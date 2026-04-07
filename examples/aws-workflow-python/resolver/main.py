import boto3
import json
from typing import Dict, Any, Optional


lambda_client = boto3.client('lambda')


def handler(event: Dict[str, Any], context: Any) -> Dict[str, Any]:
    """Resolves durable execution callbacks."""
    
    # Extract query parameters (assuming API Gateway event structure)
    query_params = event.get('queryStringParameters') or {}
    callback_id = query_params.get('callbackId')
    action = query_params.get('action')
    
    if not callback_id:
        return {
            'statusCode': 400,
            'body': json.dumps({'message': 'Missing callbackId in query parameters'})
        }
    
    try:
        if action == 'failure':
            # Send callback failure
            lambda_client.send_durable_execution_callback_failure(
                CallbackId=callback_id,
                Error={
                    'ErrorData': json.dumps({'message': 'Callback failure!'}),
                    'ErrorType': 'CallbackError',
                    'ErrorMessage': 'An error occurred during the callback execution.'
                }
            )
        elif action == 'heartbeat':
            # Send callback heartbeat
            lambda_client.send_durable_execution_callback_heartbeat(
                CallbackId=callback_id
            )
        else:
            # Default: send callback success
            lambda_client.send_durable_execution_callback_success(
                CallbackId=callback_id,
                Result=json.dumps({'message': 'Callback success!'})
            )
        
        return {
            'statusCode': 200,
            'body': json.dumps({'message': 'Callback sent successfully!'})
        }
        
    except Exception as e:
        return {
            'statusCode': 500,
            'body': json.dumps({'message': f'Error sending callback: {str(e)}'})
        }
