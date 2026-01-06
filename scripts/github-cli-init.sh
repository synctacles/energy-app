#!/bin/bash
# Initialize GitHub CLI with stored token
# Usage: source scripts/github-cli-init.sh
# Or: . scripts/github-cli-init.sh

TOKEN_FILE="$HOME/.github_token"

if [ ! -f "$TOKEN_FILE" ]; then
    echo "❌ Error: Token file not found at $TOKEN_FILE"
    echo "Please run this first:"
    echo "  export GITHUB_TOKEN='your-token-here'"
    echo "  cat > ~/.github_token << 'EOF'"
    echo "  export GITHUB_TOKEN=\"\$GITHUB_TOKEN\""
    echo "  EOF"
    echo "  chmod 600 ~/.github_token"
    return 1 2>/dev/null || exit 1
fi

# Load token
source "$TOKEN_FILE"

# Verify authentication
if ! gh auth status &>/dev/null; then
    echo "❌ GitHub CLI authentication failed"
    return 1 2>/dev/null || exit 1
fi

echo "✅ GitHub CLI authenticated successfully"
gh auth status 2>&1 | head -3
