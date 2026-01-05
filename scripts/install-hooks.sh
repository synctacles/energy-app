#!/bin/bash
# Install git hooks for credential protection
# Run this after cloning the repository

set -e

REPO_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
HOOKS_SOURCE="$REPO_ROOT/scripts/hooks"
HOOKS_TARGET="$REPO_ROOT/.git/hooks"

echo "Installing git hooks from $HOOKS_SOURCE..."

if [ ! -d "$HOOKS_TARGET" ]; then
    echo "Error: .git/hooks directory not found at $HOOKS_TARGET"
    echo "Are you running this from the repository root?"
    exit 1
fi

# Copy pre-commit hook
if [ -f "$HOOKS_SOURCE/pre-commit" ]; then
    cp "$HOOKS_SOURCE/pre-commit" "$HOOKS_TARGET/pre-commit"
    chmod +x "$HOOKS_TARGET/pre-commit"
    echo "✓ Installed pre-commit hook"
else
    echo "Warning: pre-commit hook not found at $HOOKS_SOURCE/pre-commit"
fi

echo ""
echo "Git hooks installation complete!"
echo ""
echo "Hooks installed:"
echo "  - pre-commit: Blocks hardcoded database credentials"
echo ""
echo "To test the hook:"
echo "  echo 'synctacles@localhost' > test.txt"
echo "  git add test.txt"
echo "  git commit -m 'test' # Should BLOCK"
echo ""
