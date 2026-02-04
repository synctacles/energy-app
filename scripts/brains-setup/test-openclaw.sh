#!/bin/bash
set -euo pipefail

# ============================================
# OpenClaw Test Script
# ============================================

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

TESTS_PASSED=0
TESTS_FAILED=0

test_pass() { echo -e "${GREEN}✓${NC} $1"; ((TESTS_PASSED++)); }
test_fail() { echo -e "${RED}✗${NC} $1"; ((TESTS_FAILED++)); }
test_warn() { echo -e "${YELLOW}⚠ ${NC} $1"; }

echo "============================================"
echo "OpenClaw Installation Tests"
echo "============================================"
echo ""

# ============================================
# Service Tests
# ============================================
echo "--- Service Status ---"

# PostgreSQL
if systemctl is-active --quiet postgresql; then
    test_pass "PostgreSQL is running"
else
    test_fail "PostgreSQL is NOT running"
fi

# Ollama
if systemctl is-active --quiet ollama; then
    test_pass "Ollama is running"
else
    test_fail "Ollama is NOT running"
fi

# OpenClaw
if systemctl is-active --quiet openclaw; then
    test_pass "OpenClaw is running"
else
    test_fail "OpenClaw is NOT running"
fi

echo ""

# ============================================
# Database Tests
# ============================================
echo "--- Database Tests ---"

# Database exists
if sudo -u postgres psql -lqt | cut -d \| -f 1 | grep -qw brains_kb; then
    test_pass "Database 'brains_kb' exists"
else
    test_fail "Database 'brains_kb' does NOT exist"
fi

# Schema exists
if sudo -u postgres psql -d brains_kb -c '\dn' | grep -q kb; then
    test_pass "Schema 'kb' exists"
else
    test_fail "Schema 'kb' does NOT exist"
fi

# Tables exist
TABLES=$(sudo -u postgres psql -d brains_kb -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'kb';")
if [[ $TABLES -ge 2 ]]; then
    test_pass "KB tables exist (${TABLES} tables)"
else
    test_fail "KB tables missing (found ${TABLES})"
fi

# Read-only user test
if sudo -u postgres psql -d brains_kb -c "SET ROLE openclaw_reader; SELECT 1 FROM kb.entries LIMIT 1;" &>/dev/null; then
    test_pass "openclaw_reader can SELECT from kb.entries"
else
    test_warn "openclaw_reader SELECT test failed (might be empty table)"
fi

# Write protection test
if ! sudo -u postgres psql -d brains_kb -c "SET ROLE openclaw_reader; INSERT INTO kb.entries (title, content) VALUES ('test', 'test');" &>/dev/null; then
    test_pass "openclaw_reader CANNOT INSERT into kb.entries (correct!)"
else
    test_fail "openclaw_reader CAN INSERT - SECURITY ISSUE!"
fi

echo ""

# ============================================
# Ollama Tests
# ============================================
echo "--- Ollama Tests ---"

# API responsive
if curl -s http://localhost:11434/api/tags &>/dev/null; then
    test_pass "Ollama API is responsive"
else
    test_fail "Ollama API is NOT responsive"
fi

# Models available
for model in "phi3:mini" "nomic-embed-text"; do
    if ollama list | grep -q "$model"; then
        test_pass "Model '$model' is available"
    else
        test_fail "Model '$model' is NOT available"
    fi
done

echo ""

# ============================================
# OpenClaw Tests
# ============================================
echo "--- OpenClaw Tests ---"

# Config exists
if [[ -f /etc/openclaw/openclaw.json ]]; then
    test_pass "OpenClaw config exists"
else
    test_fail "OpenClaw config missing"
fi

# Gateway port
if ss -tlnp 2>/dev/null | grep -q ':18789'; then
    test_pass "OpenClaw gateway listening on port 18789"
else
    test_fail "OpenClaw gateway NOT listening"
fi

# MCP server exists
if [[ -f /opt/openclaw/mcp/kb-search.js ]]; then
    test_pass "KB Search MCP server exists"
else
    test_fail "KB Search MCP server missing"
fi

echo ""

# ============================================
# Summary
# ============================================
echo "============================================"
echo "Test Summary"
echo "============================================"
echo -e "Passed: ${GREEN}${TESTS_PASSED}${NC}"
echo -e "Failed: ${RED}${TESTS_FAILED}${NC}"
echo ""

if [[ $TESTS_FAILED -eq 0 ]]; then
    echo -e "${GREEN}All tests passed! ✓${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed. Review above output.${NC}"
    exit 1
fi
