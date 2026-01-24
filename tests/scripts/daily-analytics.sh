#!/bin/bash
# /opt/synctacles/scripts/daily-analytics.sh

DATE=$(date +%Y-%m-%d)
YESTERDAY=$(date -d "1 day ago" +%Y-%m-%d)
LOG_DIR="/opt/synctacles/logs/analytics"
TODAY_LOG="$LOG_DIR/ips-$DATE.txt"
YESTERDAY_LOG="$LOG_DIR/ips-$YESTERDAY.txt"
REPORT="$LOG_DIR/report-$DATE.txt"

NGINX_LOG="/var/log/nginx/access.log"
TODAY_NGINX=$(date +%d/%b/%Y)

# Create log directory
mkdir -p "$LOG_DIR"

echo "=== SYNCTACLES Daily Analytics: $DATE ===" | tee "$REPORT"
echo "" | tee -a "$REPORT"

# Extract today's unique IPs
grep "$TODAY_NGINX" "$NGINX_LOG" \
  | grep -E "/(generation-mix|load|balance|signals)" \
  | grep -vE "(UptimeRobot|bot|135\.181\.255\.83|2a01:4f9:c013:9cdd)" \
  | awk '{print $1}' \
  | sort -u > "$TODAY_LOG"

# Total active users today
TOTAL=$(wc -l < "$TODAY_LOG")
echo "Active users today: $TOTAL" | tee -a "$REPORT"
echo "" | tee -a "$REPORT"

# Heavy vs Light users
echo "=== Usage Intensity ===" | tee -a "$REPORT"
echo "Heavy users (>50 req/day):" | tee -a "$REPORT"
grep "$TODAY_NGINX" "$NGINX_LOG" \
  | grep -E "/(generation-mix|load|balance|signals)" \
  | grep -vE "(UptimeRobot|bot|135\.181\.255\.83|2a01:4f9:c013:9cdd)" \
  | awk '{print $1}' \
  | sort | uniq -c | sort -rn \
  | awk '$1 > 50 {print "  " $1 " requests - " $2}' | tee -a "$REPORT"

echo "" | tee -a "$REPORT"
echo "Light users (10-50 req/day):" | tee -a "$REPORT"
grep "$TODAY_NGINX" "$NGINX_LOG" \
  | grep -E "/(generation-mix|load|balance|signals)" \
  | grep -vE "(UptimeRobot|bot|135\.181\.255\.83|2a01:4f9:c013:9cdd)" \
  | awk '{print $1}' \
  | sort | uniq -c | sort -rn \
  | awk '$1 >= 10 && $1 <= 50 {print "  " $1 " requests - " $2}' | tee -a "$REPORT"

echo "" | tee -a "$REPORT"
echo "Testers (<10 req/day):" | tee -a "$REPORT"
grep "$TODAY_NGINX" "$NGINX_LOG" \
  | grep -E "/(generation-mix|load|balance|signals)" \
  | grep -vE "(UptimeRobot|bot|135\.181\.255\.83|2a01:4f9:c013:9cdd)" \
  | awk '{print $1}' \
  | sort | uniq -c | sort -rn \
  | awk '$1 < 10 {print "  " $1 " requests - " $2}' | tee -a "$REPORT"

echo "" | tee -a "$REPORT"

# New vs Returning (if yesterday's log exists)
if [ -f "$YESTERDAY_LOG" ]; then
    echo "=== Retention Analysis ===" | tee -a "$REPORT"
    
    YESTERDAY_COUNT=$(wc -l < "$YESTERDAY_LOG")
    
    # New users (in today, not in yesterday)
    NEW=$(comm -13 "$YESTERDAY_LOG" "$TODAY_LOG")
    NEW_COUNT=$(echo "$NEW" | grep -c .)
    
    # Returning users (in both)
    RETURNING=$(comm -12 "$YESTERDAY_LOG" "$TODAY_LOG")
    RETURNING_COUNT=$(echo "$RETURNING" | grep -c .)
    
    # Churned users (in yesterday, not in today)
    CHURNED=$(comm -23 "$YESTERDAY_LOG" "$TODAY_LOG")
    CHURNED_COUNT=$(echo "$CHURNED" | grep -c .)
    
    # Retention rate
    if [ $YESTERDAY_COUNT -gt 0 ]; then
        RETENTION=$((RETURNING_COUNT * 100 / YESTERDAY_COUNT))
        echo "Retention rate: $RETENTION% ($RETURNING_COUNT/$YESTERDAY_COUNT)" | tee -a "$REPORT"
    fi
    
    echo "" | tee -a "$REPORT"
    echo "New users today: $NEW_COUNT" | tee -a "$REPORT"
    if [ $NEW_COUNT -gt 0 ]; then
        echo "$NEW" | sed 's/^/  /' | tee -a "$REPORT"
    fi
    
    echo "" | tee -a "$REPORT"
    echo "Returning users: $RETURNING_COUNT" | tee -a "$REPORT"
    if [ $RETURNING_COUNT -gt 0 ]; then
        echo "$RETURNING" | sed 's/^/  /' | tee -a "$REPORT"
    fi
    
    echo "" | tee -a "$REPORT"
    echo "Churned (lost): $CHURNED_COUNT" | tee -a "$REPORT"
    if [ $CHURNED_COUNT -gt 0 ]; then
        echo "$CHURNED" | sed 's/^/  /' | tee -a "$REPORT"
    fi
else
    echo "No yesterday data - first run" | tee -a "$REPORT"
fi

echo "" | tee -a "$REPORT"
echo "=== 7-Day Trend ===" | tee -a "$REPORT"
for i in {6..0}; do
    CHECK_DATE=$(date -d "$i days ago" +%Y-%m-%d)
    CHECK_LOG="$LOG_DIR/ips-$CHECK_DATE.txt"
    if [ -f "$CHECK_LOG" ]; then
        COUNT=$(wc -l < "$CHECK_LOG")
        echo "$CHECK_DATE: $COUNT users" | tee -a "$REPORT"
    fi
done

echo "" | tee -a "$REPORT"
echo "Report saved: $REPORT"
echo "IP list saved: $TODAY_LOG"