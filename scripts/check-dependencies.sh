#!/bin/bash
# Automated Dependency Scanning Script
# Checks for known vulnerabilities in Python dependencies
# Used by: pre-commit hook, CI/CD pipeline, local development
#
# Exit codes:
#   0 = No vulnerabilities found
#   1 = Vulnerabilities found
#   2 = Tool not available or configuration error

set -e

# Colors for output
RED='\033[0;31m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
REPORT_DIR="${PROJECT_ROOT}/.dependency-reports"

echo -e "${BLUE}🔍 Starting Dependency Scanning...${NC}\n"

# Create report directory
mkdir -p "$REPORT_DIR"

# ============================================================================
# 1. Check if pip-audit is installed
# ============================================================================
if ! command -v pip-audit &> /dev/null; then
    echo -e "${YELLOW}⚠️  pip-audit not found. Installing...${NC}"
    pip install pip-audit
fi

# ============================================================================
# 2. Run pip-audit on requirements.txt (main)
# ============================================================================
echo -e "${BLUE}📦 Scanning requirements.txt...${NC}"
if pip-audit -r "$PROJECT_ROOT/requirements.txt" \
    --desc \
    --format json > "$REPORT_DIR/pip-audit-main.json" 2>&1; then
    echo -e "${GREEN}✅ requirements.txt: No vulnerabilities found${NC}\n"
    MAIN_SAFE=true
else
    echo -e "${RED}❌ requirements.txt: Vulnerabilities detected${NC}\n"
    MAIN_SAFE=false
    # Show summary
    echo -e "${YELLOW}Vulnerability Summary:${NC}"
    pip-audit -r "$PROJECT_ROOT/requirements.txt" --desc || true
fi

# ============================================================================
# 3. Run pip-audit on requirements-frozen.txt (pinned)
# ============================================================================
if [ -f "$PROJECT_ROOT/requirements-frozen.txt" ]; then
    echo -e "${BLUE}📦 Scanning requirements-frozen.txt...${NC}"
    if pip-audit -r "$PROJECT_ROOT/requirements-frozen.txt" \
        --desc \
        --format json > "$REPORT_DIR/pip-audit-frozen.json" 2>&1; then
        echo -e "${GREEN}✅ requirements-frozen.txt: No vulnerabilities found${NC}\n"
        FROZEN_SAFE=true
    else
        echo -e "${RED}❌ requirements-frozen.txt: Vulnerabilities detected${NC}\n"
        FROZEN_SAFE=false
        echo -e "${YELLOW}Vulnerability Summary:${NC}"
        pip-audit -r "$PROJECT_ROOT/requirements-frozen.txt" --desc || true
    fi
else
    FROZEN_SAFE=true
fi

# ============================================================================
# 4. Generate HTML report for GitOps/review
# ============================================================================
echo -e "${BLUE}📊 Generating reports...${NC}"

# Create summary report
cat > "$REPORT_DIR/SCAN_SUMMARY.md" << 'SUMMARY_EOF'
# Dependency Scan Summary

Generated: $(date -u +"%Y-%m-%d %H:%M:%S UTC")

## Status
- requirements.txt: MAIN_STATUS
- requirements-frozen.txt: FROZEN_STATUS

## Details
See json reports for full vulnerability details:
- pip-audit-main.json
- pip-audit-frozen.json

## Remediation
Run: pip-audit --fix -r requirements.txt

SUMMARY_EOF

sed -i "s/MAIN_STATUS/${MAIN_SAFE:-unknown}/" "$REPORT_DIR/SCAN_SUMMARY.md"
sed -i "s/FROZEN_STATUS/${FROZEN_SAFE:-unknown}/" "$REPORT_DIR/SCAN_SUMMARY.md"

echo -e "${GREEN}✅ Reports generated in ${REPORT_DIR}${NC}\n"

# ============================================================================
# 5. Exit with appropriate status
# ============================================================================
if [ "$MAIN_SAFE" = true ] && [ "$FROZEN_SAFE" = true ]; then
    echo -e "${GREEN}✅ All dependency scans passed${NC}"
    exit 0
else
    echo -e "${RED}❌ Dependency vulnerabilities found${NC}"
    echo -e "${YELLOW}Run 'pip-audit --fix -r requirements.txt' to attempt auto-remediation${NC}"
    exit 1
fi
