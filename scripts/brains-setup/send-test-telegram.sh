#!/bin/bash
# ============================================
# Send Test Messages to All Telegram Channels
# ============================================

# Load secrets
if [[ -f /etc/openclaw/secrets.env ]]; then
    source /etc/openclaw/secrets.env
else
    echo "Error: /etc/openclaw/secrets.env not found"
    exit 1
fi

# Chat IDs (vul in na migratie)
CHAT_IDS=(
    # "chat_id_1"
    # "chat_id_2"
    # Voeg toe na inventarisatie van DEV
)

if [[ ${#CHAT_IDS[@]} -eq 0 ]]; then
    echo "Warning: No chat IDs configured in this script"
    echo "Edit this script and add chat IDs to the CHAT_IDS array"
    exit 1
fi

MESSAGE="🤖 *OpenClaw Test*

De nieuwe Brains KB Bot is online!

✅ Server: BRAINS
✅ LLM: Ollama (Phi-3 Mini)
✅ Database: Connected
✅ Status: Operationeel

Dit is een automatische test message.
Timestamp: $(date)"

for chat_id in "${CHAT_IDS[@]}"; do
    echo "Sending to chat: ${chat_id}..."

    curl -s -X POST "https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/sendMessage" \
        -d "chat_id=${chat_id}" \
        -d "text=${MESSAGE}" \
        -d "parse_mode=Markdown"

    echo ""
done

echo "Test messages sent!"
