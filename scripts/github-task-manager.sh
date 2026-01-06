#!/bin/bash

# GitHub Task Manager - Automated task management via Claude AI
# This script enables Claude to manage GitHub issues automatically
# Usage:
#   ./github-task-manager.sh status              # Show all open issues
#   ./github-task-manager.sh comment <issue> "msg"  # Add comment to issue
#   ./github-task-manager.sh update-status <issue> "in-progress|done|blocked"
#   ./github-task-manager.sh daily-report         # Generate daily standup report

set -e

# Configuration
REPO="${REPO:-DATADIO/synctacles-api}"
TOKEN="${GITHUB_PAT:-}"
BASE_URL="https://api.github.com/repos/$REPO"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Functions

function check_token() {
    if [ -z "$TOKEN" ]; then
        echo -e "${RED}❌ Error: GITHUB_PAT environment variable not set${NC}"
        echo "Please set: export GITHUB_PAT='your_token_here'"
        exit 1
    fi
}

function api_call() {
    local method=$1
    local endpoint=$2
    local data=$3

    if [ -z "$data" ]; then
        curl -s -X "$method" \
            -H "Authorization: token $TOKEN" \
            -H "Accept: application/vnd.github.v3+json" \
            "$BASE_URL$endpoint"
    else
        curl -s -X "$method" \
            -H "Authorization: token $TOKEN" \
            -H "Accept: application/vnd.github.v3+json" \
            -d "$data" \
            "$BASE_URL$endpoint"
    fi
}

function list_issues() {
    echo -e "${BLUE}📋 Open Issues:${NC}"
    api_call GET "/issues?state=open&per_page=100" | jq -r '.[] | "\(.number) [\(.labels[0].name // "no-label")] \(.title)"' | sort -n
}

function get_issue_details() {
    local issue=$1
    api_call GET "/issues/$issue"
}

function add_comment() {
    local issue=$1
    local comment=$2

    api_call POST "/issues/$issue/comments" "{\"body\":\"$comment\"}" > /dev/null
    echo -e "${GREEN}✅ Comment added to issue #$issue${NC}"
}

function update_issue_status() {
    local issue=$1
    local status=$2

    case $status in
        "in-progress")
            api_call PATCH "/issues/$issue/labels" '["in-progress"]' > /dev/null
            echo -e "${GREEN}✅ Issue #$issue marked as in-progress${NC}"
            ;;
        "done")
            api_call PATCH "/issues/$issue" '{"state":"closed"}' > /dev/null
            echo -e "${GREEN}✅ Issue #$issue closed${NC}"
            ;;
        "blocked")
            api_call PATCH "/issues/$issue/labels" '["blocked"]' > /dev/null
            echo -e "${YELLOW}⚠️ Issue #$issue marked as blocked${NC}"
            ;;
        *)
            echo -e "${RED}❌ Unknown status: $status${NC}"
            exit 1
            ;;
    esac
}

function daily_report() {
    echo -e "${BLUE}📊 Daily Standup Report${NC}"
    echo ""

    local issues=$(api_call GET "/issues?state=open&per_page=100")

    # Count by priority
    local critical=$(echo "$issues" | jq '.[] | select(.labels[].name == "critical") | .number' | wc -l)
    local high=$(echo "$issues" | jq '.[] | select(.labels[].name == "high") | .number' | wc -l)
    local medium=$(echo "$issues" | jq '.[] | select(.labels[].name == "medium") | .number' | wc -l)

    echo -e "${RED}🔴 CRITICAL: $critical${NC}"
    echo "$issues" | jq -r '.[] | select(.labels[].name == "critical") | "   #\(.number) - \(.title)"'
    echo ""

    echo -e "${YELLOW}🟠 HIGH: $high${NC}"
    echo "$issues" | jq -r '.[] | select(.labels[].name == "high") | "   #\(.number) - \(.title)"'
    echo ""

    echo -e "${BLUE}🟡 MEDIUM: $medium${NC}"
    echo "$issues" | jq -r '.[] | select(.labels[].name == "medium") | "   #\(.number) - \(.title)"'
    echo ""

    local total=$((critical + high + medium))
    echo -e "${GREEN}Total open: $total${NC}"
}

function progress_summary() {
    local issues=$(api_call GET "/issues?state=all&per_page=100")
    local open=$(echo "$issues" | jq '.[] | select(.state == "open") | .number' | wc -l)
    local closed=$(echo "$issues" | jq '.[] | select(.state == "closed") | .number' | wc -l)
    local total=$((open + closed))

    local percentage=$((closed * 100 / total))

    echo -e "${BLUE}📈 Project Progress${NC}"
    echo "Completed: $closed/$total ($percentage%)"
    echo ""
    echo "Open by priority:"

    local critical=$(api_call GET "/issues?state=open&labels=critical&per_page=100" | jq 'length')
    local high=$(api_call GET "/issues?state=open&labels=high&per_page=100" | jq 'length')
    local medium=$(api_call GET "/issues?state=open&labels=medium&per_page=100" | jq 'length')

    echo "  🔴 Critical: $critical"
    echo "  🟠 High: $high"
    echo "  🟡 Medium: $medium"
}

# Main
check_token

case "${1:-status}" in
    status)
        list_issues
        ;;
    comment)
        if [ -z "$2" ] || [ -z "$3" ]; then
            echo "Usage: $0 comment <issue-number> \"<comment>\""
            exit 1
        fi
        add_comment "$2" "$3"
        ;;
    update-status)
        if [ -z "$2" ] || [ -z "$3" ]; then
            echo "Usage: $0 update-status <issue-number> <in-progress|done|blocked>"
            exit 1
        fi
        update_issue_status "$2" "$3"
        ;;
    daily-report)
        daily_report
        ;;
    progress)
        progress_summary
        ;;
    *)
        echo "Usage: $0 {status|comment|update-status|daily-report|progress}"
        echo ""
        echo "Examples:"
        echo "  $0 status                          # List all open issues"
        echo "  $0 comment 5 'Working on this'     # Add comment to issue #5"
        echo "  $0 update-status 5 in-progress     # Mark issue #5 as in-progress"
        echo "  $0 update-status 5 done            # Close issue #5"
        echo "  $0 daily-report                    # Show daily standup report"
        echo "  $0 progress                        # Show project progress"
        exit 1
        ;;
esac
