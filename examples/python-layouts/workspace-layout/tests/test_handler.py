"""
Tests for the workspace layout example handlers.
"""
import json
import pytest
from unittest.mock import patch, MagicMock

from src.mypackage.handler import api_handler, worker_handler, process_message


class TestApiHandler:
    """Test cases for the API handler."""
    
    def test_health_endpoint(self):
        """Test the health check endpoint."""
        event = {
            "httpMethod": "GET",
            "path": "/health"
        }
        context = MagicMock()
        
        response = api_handler(event, context)
        
        assert response["statusCode"] == 200
        body = json.loads(response["body"])
        assert body["status"] == "healthy"
        assert "timestamp" in body
    
    @patch('src.mypackage.handler.requests.get')
    def test_external_endpoint(self, mock_get):
        """Test the external API endpoint."""
        # Mock external API response
        mock_response = MagicMock()
        mock_response.json.return_value = {"test": "data"}
        mock_get.return_value = mock_response
        
        event = {
            "httpMethod": "GET",
            "path": "/external"
        }
        context = MagicMock()
        
        response = api_handler(event, context)
        
        assert response["statusCode"] == 200
        body = json.loads(response["body"])
        assert "external_data" in body
        assert body["external_data"]["test"] == "data"
    
    def test_not_found(self):
        """Test 404 response for unknown paths."""
        event = {
            "httpMethod": "GET",
            "path": "/unknown"
        }
        context = MagicMock()
        
        response = api_handler(event, context)
        
        assert response["statusCode"] == 404
        body = json.loads(response["body"])
        assert body["error"] == "Not found"


class TestWorkerHandler:
    """Test cases for the worker handler."""
    
    def test_empty_records(self):
        """Test worker with no records."""
        event = {"Records": []}
        context = MagicMock()
        
        response = worker_handler(event, context)
        
        assert response["statusCode"] == 200
        assert response["processedCount"] == 0
    
    def test_sqs_message_processing(self):
        """Test processing SQS messages."""
        event = {
            "Records": [
                {
                    "body": json.dumps({
                        "type": "user_signup",
                        "user_id": "123",
                        "email": "test@example.com"
                    })
                }
            ]
        }
        context = MagicMock()
        
        response = worker_handler(event, context)
        
        assert response["statusCode"] == 200
        assert response["processedCount"] == 1


class TestProcessMessage:
    """Test cases for message processing."""
    
    def test_user_signup_message(self):
        """Test user signup message processing."""
        message = {
            "type": "user_signup",
            "user_id": "123",
            "email": "test@example.com"
        }
        
        result = process_message(message)
        
        assert result["status"] == "completed"
        assert result["action"] == "user_signup"
        assert result["user_id"] == "123"
    
    def test_order_created_message(self):
        """Test order created message processing."""
        message = {
            "type": "order_created",
            "order_id": "order-456",
            "user_id": "123"
        }
        
        result = process_message(message)
        
        assert result["status"] == "completed"
        assert result["action"] == "order_created"
        assert result["order_id"] == "order-456"
    
    def test_unknown_message_type(self):
        """Test unknown message type handling."""
        message = {
            "type": "unknown_type",
            "data": "some data"
        }
        
        result = process_message(message)
        
        assert result["status"] == "skipped"
        assert result["reason"] == "unknown_type"


# Run tests
if __name__ == "__main__":
    pytest.main([__file__])