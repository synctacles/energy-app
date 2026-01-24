LOG="/var/log/nginx/access.log"

echo "=== IP Retention Analysis ==="
echo ""

# Laatste 7 dagen, per dag
for i in {6..0}; do
    DATE=$(date -d "$i days ago" "+%d/%b/%Y")
    
    echo "=== $DATE ==="
    grep "$DATE" "$LOG" \
      | grep -E "/(generation-mix|load|balance|signals)" \
      | grep -vE "(UptimeRobot|bot|135\.181\.255\.83|2a01:4f9:c013:9cdd)" \
      | awk '{print $1}' \
      | sort | uniq -c | sort -rn
    echo ""
done

echo "=== Active IPs (last 7 days) ==="
grep -E "/(generation-mix|load|balance|signals)" "$LOG" \
  | grep -vE "(UptimeRobot|bot|135\.181\.255\.83|2a01:4f9:c013:9cdd)" \
  | awk '{print $1}' \
  | sort -u
```