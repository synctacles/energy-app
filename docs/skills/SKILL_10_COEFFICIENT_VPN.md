# SKILL 10 — COEFFICIENT SERVER VPN CONFIGURATION

WireGuard Split Tunneling for Geo-Restricted API Access
Version: 1.0 (2026-01-10)

---

## PURPOSE

This skill documents the VPN configuration and Enever proxy functionality for the coefficient engine server. The server requires a Dutch IP address to access Enever consumer price data while maintaining SSH accessibility and normal service operation.

**Server Roles:**
1. WireGuard VPN split tunnel for Dutch IP
2. Enever API proxy endpoint for main API server
3. Historical Enever data collection and storage

**Related:**
- SKILL_15: Consumer Price Engine (how prices are calculated)
- ADR-002: VPN Split Tunneling decision rationale
- HANDOFF_CC_CAI_ENEVER_IMPLEMENTATION_COMPLETE.md: Implementation details

---

## OVERVIEW

### Problem

The coefficient server (91.99.150.36) is hosted in Germany but needs to access Enever, a Dutch energy provider API that may implement geo-restrictions.

### Solution

WireGuard split tunnel that:
- ✅ Routes ONLY Enever traffic (84.46.252.107) through Dutch VPN
- ✅ Keeps SSH and all other traffic on normal routes
- ✅ Survives server reboots
- ✅ Requires no application-level proxy configuration

### Architecture

```
┌─────────────────────────────────────────────────┐
│  Coefficient Server (91.99.150.36)             │
│  Location: Hetzner Germany                     │
│                                                 │
│  ┌───────────────────────────────────────────┐ │
│  │ Routing Table                             │ │
│  │  • SSH (port 22)        → eth0 (direct)   │ │
│  │  • PostgreSQL (5432)    → eth0 (direct)   │ │
│  │  • Coefficient API      → eth0 (direct)   │ │
│  │  • 84.46.252.107/32     → pia-split (VPN) │ │ ← Enever only
│  │  • Default              → eth0 (direct)   │ │
│  └───────────────────────────────────────────┘ │
└─────────────────────────────────────────────────┘
                    │
                    │ (Only Enever traffic)
                    ▼
         ┌──────────────────────┐
         │  WireGuard Tunnel    │
         │  Interface: pia-split│
         └──────────────────────┘
                    │
                    ▼
         ┌──────────────────────┐
         │  PIA VPN Exit        │
         │  Amsterdam, NL       │
         │  158.173.21.230      │
         └──────────────────────┘
                    │
                    ▼
         ┌──────────────────────┐
         │  Enever API          │
         │  84.46.252.107       │
         │  (sees Dutch IP)     │
         └──────────────────────┘
```

---

## ENEVER PROXY FUNCTIONALITY

### Architecture

```
Main API (135.181.255.83)
│
└── GET /internal/enever/prices
            │
            ▼
    Coefficient Server (91.99.150.36)
    ├── FastAPI Proxy (port 8080)
    │   └── IP whitelist: 135.181.255.83
    │
    ├── VPN Split Tunnel
    │   └── Routes 84.46.252.107 → pia-split
    │
    └── Enever API
        └── https://enever.nl/api/
```

### Proxy Endpoint

**URL:** `http://91.99.150.36:8080/internal/enever/prices`

**Security:** IP whitelist (only main API server)

**Response:**
```json
{
  "timestamp": "2026-01-11T15:30:00Z",
  "source": "enever",
  "prices_today": {
    "Frank Energie": [
      {"hour": 0, "price": 0.2463},
      {"hour": 1, "price": 0.2412}
    ],
    "Tibber": [...]
  },
  "prices_tomorrow": null
}
```

**Health Endpoint:** `GET /internal/enever/health`

### Data Collection

**Schedule:** 2x daily via systemd timer
- 00:30 UTC — Fetch today's final prices
- 15:30 UTC — Fetch today + tomorrow

**Storage:** PostgreSQL on coefficient server

```sql
CREATE TABLE enever_prices (
    id SERIAL PRIMARY KEY,
    timestamp TIMESTAMPTZ NOT NULL,
    provider VARCHAR(50) NOT NULL,
    hour INTEGER NOT NULL,
    price_total DECIMAL(8,5) NOT NULL,
    collected_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(timestamp, provider, hour)
);
```

**Volume:** ~1200 records/day (25 providers × 24 hours × 2 collections)

### SystemD Services

**coefficient-api.service** — FastAPI proxy
```bash
sudo systemctl status coefficient-api.service
# Port 8080, auto-restart
```

**enever-collector.timer** — Data collection
```bash
sudo systemctl status enever-collector.timer
# Runs at 00:30 and 15:30 UTC
```

### Files

```
/opt/github/coefficient-engine/
├── api/main.py
├── routes/enever.py
├── services/enever_client.py
└── collectors/enever_collector.py

/opt/coefficient/
├── .enever_token
└── .env

/etc/systemd/system/
├── coefficient-api.service
├── enever-collector.service
└── enever-collector.timer
```

### Verification

```bash
# Test proxy from main API
ssh root@135.181.255.83 'curl -s http://91.99.150.36:8080/internal/enever/prices | jq ".prices_today | keys | length"'
# Expected: 25

# Check collector logs
ssh coefficient@91.99.150.36 'sudo journalctl -u enever-collector.service -n 20'

# Verify database records
ssh coefficient@91.99.150.36 'psql -U coefficient -d coefficient -c "SELECT COUNT(*) FROM enever_prices WHERE collected_at > NOW() - INTERVAL '\''1 day'\'';"'
# Expected: 1200+
```

---

## INITIAL SETUP

### Prerequisites

1. **PIA Account**
   - Username: `p3110379` (Leo's account)
   - Password: Stored in `/home/coefficient/pia_credentials` (not in git)
   - Subscription active

2. **Server Requirements**
   - WireGuard kernel module installed
   - `wg-quick` utility available
   - `curl`, `jq` for API calls
   - Coefficient user with sudo access

3. **DNS Resolution**
   ```bash
   # Verify Enever IP
   dig +short enever.nl
   # Should return: 84.46.252.107
   ```

### Installation Steps

#### 1. Clone PIA Manual Connections

```bash
ssh coefficient@91.99.150.36

cd ~
git clone https://github.com/pia-foss/manual-connections.git
cd manual-connections
```

#### 2. Run Setup Script

The setup script is at `~/setup_pia_split.sh`:

```bash
cd ~
sudo ~/setup_pia_split.sh
```

**What it does:**
1. Gets PIA authentication token (24h validity)
2. Connects to NL Amsterdam server via PIA API
3. Generates WireGuard key pair
4. Creates `/etc/wireguard/pia-split.conf`
5. Brings up tunnel with `wg-quick up pia-split`
6. Verifies routing

#### 3. Enable Auto-Start

```bash
# Enable service to start on boot
sudo systemctl enable wg-quick@pia-split

# Check status
sudo systemctl status wg-quick@pia-split
```

#### 4. Verification

```bash
# Test script
python3 ~/test_enever_vpn.py
```

Expected output:
```
✓ Traffic will go through VPN
✓ VPN tunnel is active
✓ Successfully connected to Enever
✓ HTTPS connection successful
✓ Content suggests Dutch access
```

---

## CONFIGURATION FILES

### WireGuard Config

Location: `/etc/wireguard/pia-split.conf`

```ini
[Interface]
PrivateKey = <generated-private-key>
Address = 10.32.189.185  # Example, will vary

[Peer]
PublicKey = IahOocCJ09Dky+9zgOP6qZUqg/Yntbmz3V/hwb4oFWA=  # PIA server key
Endpoint = 158.173.21.230:1337  # Amsterdam server
AllowedIPs = 84.46.252.107/32   # ONLY Enever
PersistentKeepalive = 25
```

**Critical:** `AllowedIPs` must be ONLY `84.46.252.107/32`
- ❌ DO NOT use `0.0.0.0/0` (routes all traffic, breaks SSH)
- ✅ ONLY use specific Enever IP

### Setup Script

Location: `~/setup_pia_split.sh`

Key variables:
```bash
PIA_USER="p3110379"
PIA_PASS="CW6bM8ohYh"  # Should be in env var, not hardcoded
WG_SERVER_IP="158.173.21.230"  # Amsterdam NL
WG_SERVER_CN="amsterdam448"
ENEVER_IP="84.46.252.107"
```

**Security Note:** Credentials should be stored securely, not committed to git.

---

## OPERATIONS

### Start/Stop VPN

```bash
# Start
sudo wg-quick up pia-split

# Stop
sudo wg-quick down pia-split

# Restart
sudo systemctl restart wg-quick@pia-split
```

### Check Status

```bash
# WireGuard status
sudo wg show pia-split

# Systemd service status
sudo systemctl status wg-quick@pia-split

# Routing verification
ip route get 84.46.252.107
# Should show: dev pia-split

# Tunnel health
sudo wg show pia-split latest-handshakes
# Recent timestamp = healthy
```

### Monitor Traffic

```bash
# Data transfer
sudo wg show pia-split transfer

# Live packet capture on VPN interface
sudo tcpdump -i pia-split -n

# Test Enever connectivity
curl -4 http://84.46.252.107
# Should return Enever homepage
```

---

## TROUBLESHOOTING

### Problem: SSH Connection Lost After VPN Setup

**Symptoms:**
- SSH connection times out
- Cannot reconnect to server
- Need Hetzner console access

**Cause:** VPN configured with `AllowedIPs = 0.0.0.0/0` (full tunnel)

**Solution:**
```bash
# Via Hetzner console:
sudo wg-quick down pia-split
sudo rm /etc/wireguard/pia-split.conf

# Re-run setup script with correct config
sudo ~/setup_pia_split.sh
```

**Prevention:** Always use split tunnel config (only Enever IP in AllowedIPs)

---

### Problem: VPN Tunnel Down

**Symptoms:**
```bash
sudo wg show pia-split
# No output or "interface not found"
```

**Diagnosis:**
```bash
# Check if interface exists
ip link show pia-split

# Check systemd service
sudo systemctl status wg-quick@pia-split

# Check logs
sudo journalctl -u wg-quick@pia-split -n 50
```

**Solutions:**

1. **Restart tunnel:**
   ```bash
   sudo systemctl restart wg-quick@pia-split
   ```

2. **Re-run setup (if config corrupted):**
   ```bash
   sudo wg-quick down pia-split
   sudo ~/setup_pia_split.sh
   ```

3. **Check PIA server availability:**
   ```bash
   ping -c 3 158.173.21.230
   ```

---

### Problem: Handshake Failing

**Symptoms:**
```bash
sudo wg show pia-split latest-handshakes
# Shows old timestamp (> 2 minutes ago)
```

**Causes:**
- PIA server unreachable
- Firewall blocking UDP 1337
- Token expired (>24 hours)

**Solutions:**

1. **Check connectivity to PIA:**
   ```bash
   nc -zvu 158.173.21.230 1337
   ```

2. **Refresh token and re-setup:**
   ```bash
   sudo wg-quick down pia-split
   sudo ~/setup_pia_split.sh
   ```

3. **Try different PIA server:**
   Edit `setup_pia_split.sh`, change to streaming server:
   ```bash
   WG_SERVER_IP="158.173.3.235"  # Amsterdam streaming
   WG_SERVER_CN="amsterdam404"
   ```

---

### Problem: Enever Traffic Not Going Through VPN

**Symptoms:**
```bash
ip route get 84.46.252.107
# Shows: dev eth0 (not pia-split)
```

**Diagnosis:**
```bash
# Check WireGuard config
sudo cat /etc/wireguard/pia-split.conf | grep AllowedIPs
# Should show: AllowedIPs = 84.46.252.107/32

# Check if tunnel is up
sudo wg show pia-split
```

**Solutions:**

1. **Verify routing:**
   ```bash
   # Manually add route (temporary test)
   sudo ip route add 84.46.252.107/32 dev pia-split

   # Test
   ip route get 84.46.252.107
   ```

2. **Fix WireGuard config:**
   ```bash
   sudo nano /etc/wireguard/pia-split.conf
   # Ensure AllowedIPs = 84.46.252.107/32

   # Restart
   sudo wg-quick down pia-split
   sudo wg-quick up pia-split
   ```

---

### Problem: Enever IP Changed

**Symptoms:**
- DNS shows new IP for enever.nl
- VPN traffic still routes to old IP
- Enever API requests fail

**Solution:**

1. **Check current Enever IP:**
   ```bash
   dig +short enever.nl
   # Note new IP
   ```

2. **Update setup script:**
   ```bash
   nano ~/setup_pia_split.sh
   # Update ENEVER_IP variable
   ```

3. **Update WireGuard config:**
   ```bash
   sudo nano /etc/wireguard/pia-split.conf
   # Update AllowedIPs to new IP
   ```

4. **Restart tunnel:**
   ```bash
   sudo systemctl restart wg-quick@pia-split
   ```

5. **Verify:**
   ```bash
   ip route get <new-enever-ip>
   # Should show: dev pia-split
   ```

---

### Problem: PIA Token Expired

**Symptoms:**
- Setup script fails with authentication error
- "Token expired" message

**Cause:** PIA tokens expire after 24 hours

**Solution:**

Tokens expire but established WireGuard sessions persist. Only needed for:
- Initial setup
- Re-establishing connection after disconnect

**Re-authenticate:**
```bash
sudo ~/setup_pia_split.sh
# Gets fresh token and reconnects
```

**Automate token refresh (optional):**
```bash
# Cron job to refresh weekly (before expiry becomes issue)
0 3 * * 0 /home/coefficient/setup_pia_split.sh >> /var/log/pia_refresh.log 2>&1
```

---

### Problem: High Latency Through VPN

**Symptoms:**
- Enever API requests slow (>500ms)
- Timeout errors

**Diagnosis:**
```bash
# Test latency to Enever via VPN
ping -I pia-split 84.46.252.107

# Test latency to PIA server
ping 158.173.21.230

# Check PIA server load
curl -s https://serverlist.piaservers.net/vpninfo/servers/v6 | \
  jq '.regions[] | select(.name == "Netherlands")'
```

**Solutions:**

1. **Switch to streaming optimized server:**
   ```bash
   # Edit setup_pia_split.sh
   WG_SERVER_IP="158.173.3.235"  # Streaming server
   WG_SERVER_CN="amsterdam404"

   # Re-run setup
   sudo ~/setup_pia_split.sh
   ```

2. **Try different NL server:**
   - Check PIA server list for alternative NL endpoints
   - Test latency before switching

---

## MONITORING

### Health Checks

Add to monitoring system (Grafana/Prometheus):

```bash
#!/bin/bash
# /opt/coefficient/scripts/check_vpn_health.sh

# Check 1: Tunnel exists
if ! sudo wg show pia-split &> /dev/null; then
    echo "CRITICAL: VPN tunnel down"
    exit 2
fi

# Check 2: Recent handshake
LAST_HANDSHAKE=$(sudo wg show pia-split latest-handshakes | awk '{print $2}')
NOW=$(date +%s)
AGE=$((NOW - LAST_HANDSHAKE))

if [ $AGE -gt 120 ]; then  # 2 minutes
    echo "WARNING: Handshake old ($AGE seconds)"
    exit 1
fi

# Check 3: Routing correct
ROUTE=$(ip route get 84.46.252.107 | grep -o "dev [^ ]*" | awk '{print $2}')
if [ "$ROUTE" != "pia-split" ]; then
    echo "CRITICAL: Enever not routed through VPN"
    exit 2
fi

# Check 4: Data transfer happening (if collection active)
# (Optional: check if bytes transferred recently)

echo "OK: VPN healthy"
exit 0
```

### Metrics to Track

1. **Tunnel uptime**
   ```bash
   systemctl show wg-quick@pia-split --property=ActiveEnterTimestamp
   ```

2. **Handshake freshness**
   ```bash
   sudo wg show pia-split latest-handshakes
   ```

3. **Data transfer**
   ```bash
   sudo wg show pia-split transfer
   ```

4. **Latency to Enever**
   ```bash
   ping -I pia-split -c 3 84.46.252.107 | tail -1
   ```

### Alerts

Configure alerts for:
- ❌ VPN tunnel down for > 5 minutes
- ⚠️ Handshake age > 2 minutes
- ⚠️ No data transfer for > 15 minutes (during collection)
- ❌ Enever routing not via pia-split

---

## SECURITY

### Credentials Management

**Current (simple):**
```bash
# Hardcoded in setup script (not ideal)
PIA_USER="p3110379"
PIA_PASS="CW6bM8ohYh"
```

**Better:**
```bash
# Environment variables
export PIA_USER="p3110379"
export PIA_PASS="$(cat /opt/coefficient/.pia_password)"

# In setup script
PIA_USER="${PIA_USER}"
PIA_PASS="${PIA_PASS}"
```

**Best:**
```bash
# Secrets manager (e.g., HashiCorp Vault)
PIA_USER=$(vault kv get -field=username secret/pia)
PIA_PASS=$(vault kv get -field=password secret/pia)
```

### WireGuard Keys

- Private key: `/etc/wireguard/pia-split.conf` (mode 600, root only)
- Public key: Shared with PIA (safe to expose)
- Keys rotated on each setup run (new key pair generated)

### IP Whitelisting

Coefficient API already uses IP whitelist:
- Only 135.181.255.83 (SYNCTACLES) can access coefficient API
- VPN doesn't change this security model
- Enever traffic routed via VPN is outbound only

---

## ALTERNATIVE VPN PROVIDERS

If PIA service degrades or discontinues NL servers:

### Mullvad

```bash
# Install Mullvad CLI
wget https://mullvad.net/download/app/deb/latest
sudo apt install ./mullvad-*.deb

# Login
mullvad account login <account-number>

# Connect to NL
mullvad relay set location nl
mullvad connect

# Split tunnel config (Mullvad supports this natively)
mullvad split-tunnel add 84.46.252.107
```

### ProtonVPN

```bash
# Install ProtonVPN CLI
sudo apt install protonvpn-cli

# Login
protonvpn-cli login

# Connect to NL with split tunnel
protonvpn-cli connect --cc NL
# Manual split tunnel via iptables rules
```

### Migration Effort

- PIA → Mullvad: ~30 minutes
- PIA → ProtonVPN: ~1 hour
- Similar WireGuard-based setup
- Need to update server endpoints and credentials

---

## TESTING PROCEDURES

### Before Deployment

```bash
# 1. Verify SSH access maintained
# (In separate terminal, keep SSH session open)
ssh coefficient@91.99.150.36
# Keep this running during VPN setup

# 2. In another terminal, setup VPN
ssh coefficient@91.99.150.36
sudo ~/setup_pia_split.sh

# 3. Verify first terminal still responsive
# If yes: split tunnel working
# If no: VPN broke SSH, reboot required
```

### After Deployment

```bash
# Run full test suite
python3 ~/test_enever_vpn.py

# Manual verification
curl -4 http://84.46.252.107
# Should return Enever content

# Routing check
traceroute -n -I 84.46.252.107
# Should show PIA server in path
```

### Regression Testing

After any VPN config change:

1. ✅ SSH still works
2. ✅ Enever routes via VPN
3. ✅ PostgreSQL accessible (direct route)
4. ✅ Coefficient API accessible (direct route)
5. ✅ Internet connectivity works (direct route)

---

## MAINTENANCE

### Regular Tasks

**Monthly:**
- Check PIA server list for better NL endpoints
- Review VPN logs for anomalies
- Verify WireGuard keys rotated (setup script does this)

**Quarterly:**
- Test failover to alternative PIA NL server
- Review latency metrics
- Update setup scripts if PIA API changes

**Annually:**
- Review VPN provider options
- Consider cost vs. performance
- Evaluate alternative approaches (e.g., Hetzner NL datacenter)

### Updates

**WireGuard Updates:**
```bash
# Kernel module updates via system updates
sudo apt update && sudo apt upgrade

# Restart tunnel after kernel update
sudo systemctl restart wg-quick@pia-split
```

**PIA API Changes:**
- Monitor PIA manual-connections repo for updates
- Test setup script after PIA infrastructure changes
- Update this skill doc with new procedures

---

## FUTURE ENHANCEMENTS

### Multi-Country Support

If coefficient engine expands to other countries:

```bash
# Belgium
AllowedIPs = 84.46.252.107/32, <belgium-api-ip>/32

# Or separate interfaces
wg-quick up pia-nl      # Netherlands
wg-quick up pia-be      # Belgium
wg-quick up pia-de      # Germany
```

### Automated Failover

```python
# Monitor VPN health
# If primary NL server down, switch to secondary
# Implemented in future version
```

### Dynamic IP Routing

```python
# Resolve enever.nl dynamically
# Update WireGuard AllowedIPs automatically
# Requires wg-quick post-up hooks
```

---

## REFERENCES

- **ADR-002**: VPN Split Tunneling decision
- **PIA Manual Connections**: https://github.com/pia-foss/manual-connections
- **WireGuard Docs**: https://www.wireguard.com/
- **Hetzner Network**: https://docs.hetzner.com/cloud/networks/

---

## CHANGELOG

- **2026-01-11 v1.1**: Added Enever proxy functionality (Claude Code)
  - Documented proxy endpoint architecture
  - Added data collection schedule
  - Included SystemD services documentation
  - Added verification commands

- **2026-01-10 v1.0**: Initial skill created (Claude Code)
  - Documented split tunnel setup
  - Added troubleshooting procedures
  - Included monitoring and security guidelines
  - Tested and verified on coefficient server
