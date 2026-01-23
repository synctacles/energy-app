#!/usr/bin/env python3
"""Test that development environment is properly configured"""

import sys


def test_imports():
    """Test all critical imports"""
    print("Testing Python imports...")

    packages = [
        ("entsoe", "entsoe-py"),
        ("pandas", "pandas"),
        ("fastapi", "fastapi"),
        ("sqlalchemy", "sqlalchemy"),
    ]

    failed = False
    for module, name in packages:
        try:
            __import__(module)
            print(f"✅ {name}")
        except ImportError as e:
            print(f"❌ {name}: {e}")
            failed = True

    return not failed


def main():
    print("=" * 60)
    print("SYNCTACLES Development Environment Test")
    print("=" * 60)
    print()

    if not test_imports():
        print("\n❌ Import test failed!")
        sys.exit(1)

    print()
    print("=" * 60)
    print("✅ Development environment is ready!")
    print("=" * 60)


if __name__ == "__main__":
    main()
