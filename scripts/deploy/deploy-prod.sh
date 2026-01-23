#!/bin/bash
# Deploy to SYNCTACLES PROD server
# Install: cp scripts/deploy/deploy-prod.sh ~/bin/deploy-prod && chmod +x ~/bin/deploy-prod

echo "🚀 Deploying to PROD..."
ssh cc-hub "ssh synct-prod 'sudo /opt/synctacles/auto-update.sh'"
echo ""
echo "✅ Done!"
