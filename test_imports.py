#!/usr/bin/env python3
"""
Test script to check various import scenarios and file path issues
"""

import os
import sys

def test_imports():
    """Test different import scenarios"""
    print("=== Import Tests ===")
    
    # Test 1: Absolute imports
    try:
        from shared.utils import format_response
        print("✅ Absolute import (shared.utils) works")
    except ImportError as e:
        print(f"❌ Absolute import failed: {e}")
    
    # Test 2: Relative imports (if we're in a package)
    try:
        from .utils import something  # This would fail in a script context
        print("✅ Relative import works")
    except ImportError as e:
        print(f"❌ Relative import failed: {e}")
    
    # Test 3: Deep nested imports
    try:
        from app.functions.api.handler import main
        print("✅ Deep nested import works")
    except ImportError as e:
        print(f"❌ Deep nested import failed: {e}")

def test_file_paths():
    """Test file path scenarios"""
    print("\n=== File Path Tests ===")
    
    print(f"Current working directory: {os.getcwd()}")
    print(f"Script location: {__file__}")
    print(f"Python path: {sys.path}")
    
    # Test 1: Relative file path
    try:
        with open('data/config.json', 'r') as f:
            print("✅ Relative file path works")
    except FileNotFoundError:
        print("❌ Relative file path failed - file not found")
    
    # Test 2: Absolute file path based on script location
    try:
        script_dir = os.path.dirname(os.path.abspath(__file__))
        config_path = os.path.join(script_dir, 'data', 'config.json')
        with open(config_path, 'r') as f:
            print("✅ Absolute file path works")
    except FileNotFoundError:
        print("❌ Absolute file path failed - file not found")

if __name__ == "__main__":
    test_imports()
    test_file_paths()