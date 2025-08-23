"""
SST Python Runtime Module

This module provides access to SST resources in Python Lambda functions.
It reads encrypted resource data from resource.enc and provides a Resource API.
"""

import os
import json
import base64
from typing import Any, Dict
from cryptography.hazmat.primitives.ciphers import Cipher, algorithms, modes
from cryptography.hazmat.backends import default_backend


class ResourceProxy:
    """Proxy object that provides attribute access to resource properties."""
    
    def __init__(self, data: Dict[str, Any]):
        self._data = data
    
    def __getattr__(self, name: str) -> Any:
        if name in self._data:
            value = self._data[name]
            if isinstance(value, dict):
                return ResourceProxy(value)
            return value
        raise AttributeError(f"Resource has no attribute '{name}'")


class ResourceManager:
    """Manages SST resources by decrypting and parsing resource.enc file."""
    
    def __init__(self):
        self._resources = None
        self._load_resources()
    
    def _load_resources(self):
        """Load and decrypt resources from resource.enc file."""
        try:
            # Get encryption key from environment
            key_b64 = os.environ.get('SST_KEY')
            if not key_b64:
                # In development mode, resources might be provided directly via env vars
                self._load_dev_resources()
                return
            
            # Decode the base64 key
            key = base64.b64decode(key_b64)
            
            # Read the encrypted resource file
            resource_file = os.environ.get('SST_KEY_FILE', 'resource.enc')
            if not os.path.exists(resource_file):
                raise FileNotFoundError(f"Resource file {resource_file} not found")
            
            with open(resource_file, 'rb') as f:
                ciphertext = f.read()
            
            # Decrypt using AES-GCM (matching Go implementation)
            # Go uses 12-byte nonce (all zeros) and no additional data
            nonce = b'\x00' * 12
            cipher = Cipher(algorithms.AES(key), modes.GCM(nonce), backend=default_backend())
            decryptor = cipher.decryptor()
            
            # The ciphertext includes the auth tag at the end (16 bytes for GCM)
            if len(ciphertext) < 16:
                raise ValueError("Invalid ciphertext: too short")
            
            # Split ciphertext and auth tag
            actual_ciphertext = ciphertext[:-16]
            auth_tag = ciphertext[-16:]
            
            # Set the auth tag and decrypt
            decryptor.authenticate_additional_data(b'')
            plaintext = decryptor.update(actual_ciphertext)
            decryptor.finalize_with_tag(auth_tag)
            
            # Parse JSON
            self._resources = json.loads(plaintext.decode('utf-8'))
            
        except Exception as e:
            # Fallback to development mode if decryption fails
            print(f"Warning: Failed to decrypt resources ({e}), falling back to development mode")
            self._load_dev_resources()
    
    def _load_dev_resources(self):
        """Load resources from environment variables (development mode)."""
        self._resources = {}
        
        # Look for SST_RESOURCE_* environment variables
        for key, value in os.environ.items():
            if key.startswith('SST_RESOURCE_') and key != 'SST_RESOURCE_App':
                resource_name = key[13:]  # Remove 'SST_RESOURCE_' prefix
                try:
                    self._resources[resource_name] = json.loads(value)
                except json.JSONDecodeError:
                    self._resources[resource_name] = value
    
    def __getattr__(self, name: str) -> Any:
        if self._resources is None:
            raise RuntimeError("Resources not loaded")
        
        if name in self._resources:
            value = self._resources[name]
            if isinstance(value, dict):
                return ResourceProxy(value)
            return value
        
        raise AttributeError(f"Resource '{name}' not found")


# Global resource manager instance
Resource = ResourceManager()