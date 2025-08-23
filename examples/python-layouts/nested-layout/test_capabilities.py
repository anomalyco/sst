#!/usr/bin/env python3
"""
Test script to validate Python deployment capabilities.
Tests module imports, static file loading, and path resolution.
"""

import os
import sys
import json
from pathlib import Path

def test_absolute_imports():
    """Test absolute imports work correctly."""
    print("=== Testing Absolute Imports ===")
    
    try:
        # Test importing from shared module
        from shared.utils import format_response, get_current_time
        print("✅ Absolute import from shared.utils works")
        
        # Test using the imported functions
        response = format_response(200, {"test": "data"})
        timestamp = get_current_time()
        print(f"✅ Functions work: response type={type(response)}, timestamp={timestamp}")
        
    except ImportError as e:
        print(f"❌ Absolute import failed: {e}")
        return False
    
    try:
        # Test importing from nested app modules
        from app.functions.auth.handler import main as auth_main
        print("✅ Absolute import from app.functions.auth.handler works")
        
    except ImportError as e:
        print(f"❌ Nested absolute import failed: {e}")
        return False
    
    return True


def test_relative_imports():
    """Test relative imports (these might not work)."""
    print("\n=== Testing Relative Imports ===")
    
    # This test needs to be run from within a package context
    # We'll create a simple test case
    try:
        # This would only work if we're running as part of a package
        # In Lambda, this might fail
        print("⚠️  Relative imports are not recommended in Lambda functions")
        print("   Use absolute imports instead: from shared.utils import ...")
        return True
        
    except Exception as e:
        print(f"❌ Relative import test failed: {e}")
        return False


def test_static_file_loading():
    """Test loading static files with different path strategies."""
    print("\n=== Testing Static File Loading ===")
    
    # Strategy 1: Relative to current file
    try:
        current_dir = Path(__file__).parent
        config_path = current_dir / "app" / "data" / "config.json"
        
        if config_path.exists():
            with open(config_path, 'r') as f:
                config = json.load(f)
            print(f"✅ Static file loaded with Path: {config}")
        else:
            print(f"❌ Config file not found at: {config_path}")
            
    except Exception as e:
        print(f"❌ Path-based file loading failed: {e}")
    
    # Strategy 2: Using os.path relative to __file__
    try:
        script_dir = os.path.dirname(os.path.abspath(__file__))
        config_path = os.path.join(script_dir, "app", "data", "config.json")
        
        if os.path.exists(config_path):
            with open(config_path, 'r') as f:
                config = json.load(f)
            print(f"✅ Static file loaded with os.path: {config}")
        else:
            print(f"❌ Config file not found at: {config_path}")
            
    except Exception as e:
        print(f"❌ os.path-based file loading failed: {e}")
    
    # Strategy 3: Working directory relative (not recommended)
    try:
        cwd = os.getcwd()
        config_path = os.path.join(cwd, "app", "data", "config.json")
        
        if os.path.exists(config_path):
            with open(config_path, 'r') as f:
                config = json.load(f)
            print(f"✅ Static file loaded from CWD: {config}")
        else:
            print(f"⚠️  Config file not found in CWD: {config_path}")
            print(f"   Current working directory: {cwd}")
            
    except Exception as e:
        print(f"❌ CWD-based file loading failed: {e}")
    
    return True


def test_environment_info():
    """Print environment information for debugging."""
    print("\n=== Environment Information ===")
    print(f"Python version: {sys.version}")
    print(f"Current working directory: {os.getcwd()}")
    print(f"Script location: {__file__}")
    print(f"Python path: {sys.path[:3]}...")  # First 3 entries
    
    # List contents of current directory
    print(f"\nContents of current directory:")
    try:
        for item in sorted(os.listdir(".")):
            item_path = os.path.join(".", item)
            if os.path.isdir(item_path):
                print(f"  📁 {item}/")
            else:
                print(f"  📄 {item}")
    except Exception as e:
        print(f"❌ Could not list directory: {e}")


def main():
    """Run all tests."""
    print("Python Deployment Capabilities Test")
    print("=" * 50)
    
    test_environment_info()
    
    success = True
    success &= test_absolute_imports()
    success &= test_relative_imports()
    success &= test_static_file_loading()
    
    print("\n" + "=" * 50)
    if success:
        print("✅ All tests completed (some may have warnings)")
    else:
        print("❌ Some tests failed")
    
    return 0 if success else 1


if __name__ == "__main__":
    sys.exit(main())