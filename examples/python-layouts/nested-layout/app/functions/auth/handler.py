"""
Auth Lambda handler for nested layout example.
"""
import json
import logging
from typing import Any, Dict
import base64
import hashlib
import hmac

from shared.utils import format_response, get_current_time

# Configure logging
logger = logging.getLogger(__name__)
logger.setLevel(logging.INFO)

# Mock secret key (in real app, use AWS Secrets Manager)
SECRET_KEY = "your-secret-key-here"


def main(event: Dict[str, Any], context: Any) -> Dict[str, Any]:
    """
    Main auth Lambda handler.
    
    Args:
        event: Lambda event data
        context: Lambda context object
        
    Returns:
        Auth response
    """
    try:
        logger.info("Auth handler invoked")
        
        # Get request data
        # Handle both API Gateway and Lambda Function URL events
        http_method = event.get("httpMethod") or event.get("requestContext", {}).get("http", {}).get("method", "POST")
        path = event.get("path") or event.get("rawPath", "/")
        
        # Route auth requests
        if http_method == "POST" and path == "/auth/login":
            return handle_login(event)
        elif http_method == "POST" and path == "/auth/register":
            return handle_register(event)
        elif http_method == "POST" and path == "/auth/verify":
            return handle_verify_token(event)
        elif http_method == "POST" and path == "/auth/refresh":
            return handle_refresh_token(event)
        elif http_method == "GET" and path == "/auth/me":
            return handle_get_user_info(event)
        else:
            return format_response(404, {
                "error": "Auth endpoint not found",
                "path": path,
                "method": http_method
            })
            
    except Exception as e:
        logger.error(f"Auth handler error: {str(e)}")
        return format_response(500, {
            "error": "Internal server error",
            "message": str(e)
        })


def handle_login(event: Dict[str, Any]) -> Dict[str, Any]:
    """Handle user login requests."""
    try:
        body = json.loads(event.get("body", "{}"))
        email = body.get("email")
        password = body.get("password")
        
        if not email or not password:
            return format_response(400, {
                "error": "Email and password are required"
            })
        
        # Mock user validation (in real app, check against database)
        if validate_user_credentials(email, password):
            # Generate tokens
            access_token = generate_access_token(email)
            refresh_token = generate_refresh_token(email)
            
            return format_response(200, {
                "message": "Login successful",
                "access_token": access_token,
                "refresh_token": refresh_token,
                "expires_in": 3600,  # 1 hour
                "user": {
                    "email": email,
                    "id": get_user_id(email)
                }
            })
        else:
            return format_response(401, {
                "error": "Invalid credentials"
            })
            
    except json.JSONDecodeError:
        return format_response(400, {
            "error": "Invalid JSON in request body"
        })


def handle_register(event: Dict[str, Any]) -> Dict[str, Any]:
    """Handle user registration requests."""
    try:
        body = json.loads(event.get("body", "{}"))
        email = body.get("email")
        password = body.get("password")
        name = body.get("name")
        
        if not email or not password or not name:
            return format_response(400, {
                "error": "Email, password, and name are required"
            })
        
        # Mock user creation (in real app, save to database)
        if create_user(email, password, name):
            return format_response(201, {
                "message": "User registered successfully",
                "user": {
                    "email": email,
                    "name": name,
                    "id": get_user_id(email)
                }
            })
        else:
            return format_response(409, {
                "error": "User already exists"
            })
            
    except json.JSONDecodeError:
        return format_response(400, {
            "error": "Invalid JSON in request body"
        })


def handle_verify_token(event: Dict[str, Any]) -> Dict[str, Any]:
    """Handle token verification requests."""
    try:
        body = json.loads(event.get("body", "{}"))
        token = body.get("token")
        
        if not token:
            return format_response(400, {
                "error": "Token is required"
            })
        
        # Verify token
        user_info = verify_access_token(token)
        if user_info:
            return format_response(200, {
                "valid": True,
                "user": user_info,
                "expires_at": get_token_expiry(token)
            })
        else:
            return format_response(401, {
                "valid": False,
                "error": "Invalid or expired token"
            })
            
    except json.JSONDecodeError:
        return format_response(400, {
            "error": "Invalid JSON in request body"
        })


def handle_refresh_token(event: Dict[str, Any]) -> Dict[str, Any]:
    """Handle token refresh requests."""
    try:
        body = json.loads(event.get("body", "{}"))
        refresh_token = body.get("refresh_token")
        
        if not refresh_token:
            return format_response(400, {
                "error": "Refresh token is required"
            })
        
        # Verify refresh token and generate new access token
        user_email = verify_refresh_token(refresh_token)
        if user_email:
            new_access_token = generate_access_token(user_email)
            
            return format_response(200, {
                "access_token": new_access_token,
                "expires_in": 3600,
                "token_type": "Bearer"
            })
        else:
            return format_response(401, {
                "error": "Invalid refresh token"
            })
            
    except json.JSONDecodeError:
        return format_response(400, {
            "error": "Invalid JSON in request body"
        })


def handle_get_user_info(event: Dict[str, Any]) -> Dict[str, Any]:
    """Handle get user info requests."""
    # Extract token from Authorization header
    headers = event.get("headers", {})
    auth_header = headers.get("Authorization", "")
    
    if not auth_header.startswith("Bearer "):
        return format_response(401, {
            "error": "Missing or invalid Authorization header"
        })
    
    token = auth_header[7:]  # Remove "Bearer " prefix
    user_info = verify_access_token(token)
    
    if user_info:
        return format_response(200, {
            "user": user_info
        })
    else:
        return format_response(401, {
            "error": "Invalid or expired token"
        })


# Mock authentication functions (replace with real implementations)

def validate_user_credentials(email: str, password: str) -> bool:
    """Validate user credentials (mock implementation)."""
    # In real app, hash password and check against database
    mock_users = {
        "user@example.com": "password123",
        "admin@example.com": "admin123"
    }
    return mock_users.get(email) == password


def create_user(email: str, password: str, name: str) -> bool:
    """Create a new user (mock implementation)."""
    # In real app, save to database
    # For demo, just check if user doesn't exist
    return email not in ["user@example.com", "admin@example.com"]


def get_user_id(email: str) -> str:
    """Get user ID from email (mock implementation)."""
    # In real app, look up from database
    return hashlib.md5(email.encode()).hexdigest()[:8]


def generate_access_token(email: str) -> str:
    """Generate access token (simplified implementation)."""
    payload = {
        "email": email,
        "exp": int(get_current_time().timestamp()) + 3600,  # 1 hour
        "type": "access"
    }
    token_data = json.dumps(payload)
    signature = hmac.new(
        SECRET_KEY.encode(),
        token_data.encode(),
        hashlib.sha256
    ).hexdigest()
    
    return base64.b64encode(f"{token_data}.{signature}".encode()).decode()


def generate_refresh_token(email: str) -> str:
    """Generate refresh token (simplified implementation)."""
    payload = {
        "email": email,
        "exp": int(get_current_time().timestamp()) + 86400 * 7,  # 7 days
        "type": "refresh"
    }
    token_data = json.dumps(payload)
    signature = hmac.new(
        SECRET_KEY.encode(),
        token_data.encode(),
        hashlib.sha256
    ).hexdigest()
    
    return base64.b64encode(f"{token_data}.{signature}".encode()).decode()


def verify_access_token(token: str) -> Dict[str, Any]:
    """Verify access token and return user info."""
    try:
        decoded = base64.b64decode(token.encode()).decode()
        token_data, signature = decoded.rsplit(".", 1)
        
        # Verify signature
        expected_signature = hmac.new(
            SECRET_KEY.encode(),
            token_data.encode(),
            hashlib.sha256
        ).hexdigest()
        
        if signature != expected_signature:
            return None
        
        payload = json.loads(token_data)
        
        # Check expiration
        if payload.get("exp", 0) < int(get_current_time().timestamp()):
            return None
        
        # Check token type
        if payload.get("type") != "access":
            return None
        
        return {
            "email": payload["email"],
            "id": get_user_id(payload["email"])
        }
        
    except Exception:
        return None


def verify_refresh_token(token: str) -> str:
    """Verify refresh token and return user email."""
    try:
        decoded = base64.b64decode(token.encode()).decode()
        token_data, signature = decoded.rsplit(".", 1)
        
        # Verify signature
        expected_signature = hmac.new(
            SECRET_KEY.encode(),
            token_data.encode(),
            hashlib.sha256
        ).hexdigest()
        
        if signature != expected_signature:
            return None
        
        payload = json.loads(token_data)
        
        # Check expiration
        if payload.get("exp", 0) < int(get_current_time().timestamp()):
            return None
        
        # Check token type
        if payload.get("type") != "refresh":
            return None
        
        return payload["email"]
        
    except Exception:
        return None


def get_token_expiry(token: str) -> str:
    """Get token expiry time."""
    try:
        decoded = base64.b64decode(token.encode()).decode()
        token_data, _ = decoded.rsplit(".", 1)
        payload = json.loads(token_data)
        return payload.get("exp", 0)
    except Exception:
        return 0