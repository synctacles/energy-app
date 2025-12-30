#!/bin/bash
# TCP Kernel Tuning voor SYNCTACLES
# Voorkomt TIME_WAIT socket exhaustion bij high concurrency

set -euo pipefail

echo "=== TCP Kernel Tuning ==="

# Check if already applied
if grep -q "SYNCTACLES Performance Tuning" /etc/sysctl.conf 2>/dev/null; then
    echo "✅ TCP tuning already applied"
    exit 0
fi

# Apply tuning
cat >> /etc/sysctl.conf << 'SYSCTL'

# SYNCTACLES Performance Tuning
# Toegevoegd door setup script
# Docs: SKILL_08_HARDWARE_PROFILE.md

# Reuse TIME_WAIT sockets for new connections
net.ipv4.tcp_tw_reuse = 1

# Reduce TIME_WAIT duration (60s -> 30s)
net.ipv4.tcp_fin_timeout = 30

# Increase connection backlog
net.core.somaxconn = 4096
net.ipv4.tcp_max_syn_backlog = 4096
SYSCTL

# Apply immediately
sysctl -p

echo "✅ TCP tuning applied"
sysctl net.ipv4.tcp_tw_reuse net.ipv4.tcp_fin_timeout net.core.somaxconn
