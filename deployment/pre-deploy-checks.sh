#!/usr/bin/env bash
# pre-deploy-checks.sh
# Pre-deployment validation for SYNCTACLES
# Version: 1.0 (2025-12-21)

set -euo pipefail

RED="\e[31m"; GREEN="\e[32m"; YELLOW="\e[33m"; NC="\e[0m"

FAILED=0

check() {
    local name="$1"
    shift
    if "$@" >/dev/null 2>&1; then
        echo -e "${GREEN}✓${NC} $name"
    else
        echo -e "${RED}✗${NC} $name"
        ((FAILED++))
    fi
}

header() {
    echo
    echo "=========================================="
    echo "$1"
    echo "=========================================="
}

header "Pre-Deploy Checks"

# 1. Git status
check "Git: No uncommitted changes" git diff-index --quiet HEAD --
check "Git: On main branch" test "$(git rev-parse --abbrev-ref HEAD)" = "main"

# 2. VERSION file exists
check "VERSION file exists" test -f VERSION

# 3. Database migrations valid
if [[ -d alembic/versions ]]; then
    check "Alembic migrations exist" test -n "$(ls -A alembic/versions/*.py 2>/dev/null)"
else
    echo -e "${YELLOW}⚠${NC} No alembic/versions directory"
fi


# 7. Python syntax check (sample)
if command -v python3 >/dev/null 2>&1; then
    if [[ -d synctacles_db ]]; then
        check "Python syntax valid" python3 -m py_compile synctacles_db/**/*.py 2>/dev/null || true
    fi
fi

echo
if [[ $FAILED -eq 0 ]]; then
    echo -e "${GREEN}✓ All checks passed${NC}"
    exit 0
else
    echo -e "${RED}✗ $FAILED checks failed${NC}"
    echo "Fix issues before deploying"
    exit 1
fi
