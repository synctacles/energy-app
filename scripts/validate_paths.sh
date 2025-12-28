#!/usr/bin/env bash
# validate_paths.sh - Check PAD-CONTRACT compliance (SKILL 9)
set -euo pipefail

APP_DIR="${1:-/opt/synctacles/app}"
ERRORS=0

echo "=== PAD-CONTRACT Validator ==="
echo "Checking: $APP_DIR"
echo

# Pattern to search for (avoid self-detection by building pattern dynamically)
FORBIDDEN_PATH="/opt/github"
FORBIDDEN_PATH="${FORBIDDEN_PATH}/"

# Check 1: No /opt/github/ in shell scripts
echo "[CHECK] Shell scripts..."
if grep -rn --exclude="validate_paths.sh" "$FORBIDDEN_PATH" "$APP_DIR/scripts/"*.sh 2>/dev/null; then
    echo "❌ FAIL: Found $FORBIDDEN_PATH in shell scripts"
    ERRORS=$((ERRORS + 1))
else
    echo "✅ PASS: No $FORBIDDEN_PATH in shell scripts"
fi

# Check 2: No /opt/github/ in Python code
echo "[CHECK] Python code..."
if grep -rn "$FORBIDDEN_PATH" "$APP_DIR/sparkcrawler_db/" "$APP_DIR/synctacles_db/" 2>/dev/null; then
    echo "❌ FAIL: Found $FORBIDDEN_PATH in Python code"
    ERRORS=$((ERRORS + 1))
else
    echo "✅ PASS: No $FORBIDDEN_PATH in Python code"
fi

# Check 3: No hardcoded Path(__file__) for logs
echo "[CHECK] Hardcoded log paths..."
if grep -rn 'Path(__file__).*log' "$APP_DIR/sparkcrawler_db/" 2>/dev/null; then
    echo "❌ FAIL: Found hardcoded Path(__file__) for logs"
    ERRORS=$((ERRORS + 1))
else
    echo "✅ PASS: No hardcoded log paths"
fi

# Check 4: SYNCTACLES_LOG_DIR used in collectors
echo "[CHECK] SYNCTACLES_LOG_DIR usage..."
if grep -rq "SYNCTACLES_LOG_DIR" "$APP_DIR/sparkcrawler_db/collectors/"; then
    echo "✅ PASS: SYNCTACLES_LOG_DIR found in collectors"
else
    echo "❌ FAIL: SYNCTACLES_LOG_DIR not used in collectors"
    ERRORS=$((ERRORS + 1))
fi

echo
echo "=== Result ==="
if [[ $ERRORS -eq 0 ]]; then
    echo "✅ All checks passed"
    exit 0
else
    echo "❌ $ERRORS check(s) failed"
    exit 1
fi
