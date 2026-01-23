#!/bin/bash
# Check SYNCTACLES PROD server status
# Install: cp scripts/deploy/prod-status.sh ~/bin/prod-status && chmod +x ~/bin/prod-status

ssh cc-hub "ssh synct-prod 'echo \"=== API ===\"; systemctl status synctacles-api --no-pager | head -5; echo; echo \"=== Git ===\"; sudo -u synctacles git -C /opt/github/synctacles-api log -1 --oneline'"
