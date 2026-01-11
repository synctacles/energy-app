# ADR-002: VPN Split Tunneling for Coefficient Server

**Date:** 2026-01-10
**Status:** Accepted
**Deciders:** Leo, Claude Code
**Related:** HANDOFF_CC_COEFFICIENT_ENGINE.md, SKILL_10_COEFFICIENT_VPN.md
**Scope:** Coefficient Engine Infrastructure

---

## Context

The coefficient engine server (91.99.150.36) is hosted on Hetzner in Germany. It needs to collect consumer price data from Enever, a Dutch energy provider. Enever may implement geo-restrictions that block non-Dutch IP addresses.

Key constraints:
- Server is in Germany (DE) but needs Dutch (NL) IP for Enever access
- Server must remain accessible via SSH for management
- Other services (PostgreSQL, coefficient API) must remain reachable
- VPN connection must survive reboots
- Leo has an existing Private Internet Access (PIA) VPN subscription

Initial attempts with full VPN tunnel resulted in SSH connection timeouts, requiring server reboot via Hetzner console.

---

## Decision

**Implement WireGuard split tunneling that routes ONLY Enever traffic through a Dutch VPN exit point.**

Configuration:
- VPN Provider: Private Internet Access (PIA)
- Protocol: WireGuard
- Exit Location: Netherlands (Amsterdam)
- Routing: Split tunnel (not full tunnel)
- Target: Only traffic to 84.46.252.107 (enever.nl) goes via VPN
- All other traffic: Direct routing (normal path)

Implementation:
```bash
# WireGuard interface: pia-split
# AllowedIPs: 84.46.252.107/32 (Enever only)
# No default route (0.0.0.0/0) installed
```

---

## Alternatives Considered

### 1. Full VPN Tunnel (REJECTED)

**Approach:** Route all traffic through Dutch VPN exit.

**Pros:**
- Simple configuration
- All traffic appears from NL

**Cons:**
- ❌ Breaks SSH access (existing connections timeout)
- ❌ Requires complex routing rules to exclude SSH
- ❌ All services appear to come from VPN IP (unnecessary)
- ❌ Higher latency for non-Enever traffic

**Why rejected:** SSH access is critical. First implementation attempt resulted in connection timeout requiring manual reboot via Hetzner console. Not acceptable for production.

### 2. SOCKS5 Proxy (REJECTED)

**Approach:** Use PIA's SOCKS5 proxy service to route specific application traffic.

**Pros:**
- Application-level control (Python requests can use proxy)
- No system-wide routing changes
- SSH remains unaffected

**Cons:**
- ❌ PIA no longer offers SOCKS5 proxies (verified 2026-01-10)
- ❌ Requires application-level proxy configuration
- ❌ Less transparent than network-level routing

**Why rejected:** PIA removed SOCKS5 support from their entire server fleet. Investigation of their server list showed 0 regions with proxysocks servers.

### 3. No VPN - Direct Access (REJECTED)

**Approach:** Access Enever directly from German IP.

**Pros:**
- Simple, no VPN needed
- No configuration overhead
- No dependency on VPN provider

**Cons:**
- ❌ May trigger geo-blocking from Enever
- ❌ May appear suspicious (DE server accessing NL consumer API)
- ❌ Could result in API access denial

**Why rejected:** Enever is a Dutch consumer service. A German server repeatedly accessing their API may trigger security alerts or geo-restrictions. Professional appearance requires Dutch IP.

### 4. VPN with SSH Port Exclusion (CONSIDERED)

**Approach:** Full VPN tunnel but exclude port 22 (SSH) from VPN routing.

**Pros:**
- All HTTP(S) traffic goes via VPN
- SSH explicitly excluded

**Cons:**
- More complex routing rules
- Potential for misconfiguration
- Still routes unnecessary traffic through VPN

**Why not chosen:** Split tunneling by destination IP is simpler and more maintainable. We only need Enever routed, not all HTTP(S) traffic.

---

## Implementation

### WireGuard Configuration

```conf
# /etc/wireguard/pia-split.conf
[Interface]
PrivateKey = <generated>
Address = <PIA-assigned-IP>

[Peer]
PublicKey = <PIA-server-key>
Endpoint = 158.173.21.230:1337  # NL Amsterdam
AllowedIPs = 84.46.252.107/32   # Enever only
PersistentKeepalive = 25
```

### Routing Verification

```bash
# Verify Enever routes through VPN
ip route get 84.46.252.107
# Output: 84.46.252.107 dev pia-split src <VPN-IP>

# Verify SSH uses normal route
ip route get <server-IP>
# Output: <server-IP> dev eth0 src <server-IP>
```

### Service Persistence

```bash
# Enable auto-start on boot
systemctl enable wg-quick@pia-split

# Status check
systemctl status wg-quick@pia-split
wg show pia-split
```

---

## Consequences

### Positive

✅ **SSH Access Preserved**: Management access remains stable
✅ **Selective Routing**: Only Enever traffic goes via VPN
✅ **Low Latency**: Other services (PostgreSQL, API) use direct routing
✅ **Transparent**: Python code doesn't need proxy configuration
✅ **Persistent**: Automatically starts on boot
✅ **Verifiable**: Easy to test routing with `ip route get`

### Negative

⚠️ **PIA Token Expiry**: PIA tokens expire every 24 hours
- **Mitigation**: WireGuard session remains active once established
- **Impact**: Only affects new connection establishment
- **Resolution**: Automated token refresh can be implemented if needed

⚠️ **Single Point of Failure**: If PIA Amsterdam server goes down, Enever access fails
- **Mitigation**: PIA has multiple NL servers, can switch endpoints
- **Impact**: Coefficient data collection paused until VPN restored
- **Resolution**: Fallback to historical coefficients in SYNCTACLES API

⚠️ **IP Address Dependency**: Hardcoded Enever IP (84.46.252.107)
- **Mitigation**: DNS changes would require config update
- **Impact**: Low (Enever unlikely to change IP frequently)
- **Resolution**: Could be enhanced to resolve domain dynamically

⚠️ **Maintenance Overhead**: VPN configuration needs occasional updates
- **Monitoring**: Check VPN tunnel health in regular health checks
- **Documentation**: SKILL_10 provides troubleshooting procedures

### Neutral

ℹ️ **Geo-location Complexity**: Server appears to be in both DE and NL simultaneously
- For Enever: Appears to come from Netherlands (intended)
- For everything else: Appears to come from Germany (correct)
- This is expected behavior for split tunneling

---

## Verification

Test performed 2026-01-10:

```python
# Test 1: Routing verification
ip route get 84.46.252.107
# ✓ Output: dev pia-split (correct)

# Test 2: Connection test
curl http://84.46.252.107
# ✓ Status: 200 (Enever homepage)

# Test 3: HTTPS test
curl https://enever.nl
# ✓ Status: 200 (content suggests Dutch access)

# Test 4: SSH preservation
ssh coefficient@91.99.150.36 'hostname'
# ✓ Output: coefficient (connection maintained)

# Test 5: Packet capture
tcpdump -i pia-split dst 84.46.252.107
# ✓ Packets visible on VPN interface
```

All tests passed. Split tunnel configuration confirmed working.

---

## Security Considerations

1. **VPN Provider Trust**: We trust PIA not to log or interfere with traffic
   - PIA has no-logging policy (independently audited)
   - Only Enever traffic exposed to PIA

2. **Credentials**: PIA username/password stored in setup scripts
   - Not committed to git (in .gitignore)
   - Only accessible to coefficient user
   - Token-based auth after initial setup

3. **IP Whitelisting**: Coefficient API already uses IP whitelist
   - Only SYNCTACLES server (135.181.255.83) can access
   - VPN doesn't change this security model

---

## Monitoring

Health checks should verify:
1. WireGuard tunnel is up: `wg show pia-split`
2. Route exists: `ip route get 84.46.252.107 | grep pia-split`
3. Handshake is recent: `wg show pia-split latest-handshakes`
4. Data transfer occurring: `wg show pia-split transfer`

Alert if:
- Tunnel down for > 5 minutes
- No handshake in > 2 minutes
- No data transfer in > 15 minutes (during active collection)

---

## Future Considerations

### If Enever Blocks VPN IPs

If Enever implements VPN detection:
1. Try residential proxy service (more expensive)
2. Use cloud provider with NL region (e.g., Hetzner NL datacenter)
3. Request API access directly from Enever as business partner

### If PIA Service Degrades

Alternative VPN providers:
- Mullvad (supports WireGuard, NL servers)
- ProtonVPN (NL servers available)
- IVPN (privacy-focused, NL available)

Migration effort: ~30 minutes (similar WireGuard setup)

### Multi-Country Expansion

If coefficient engine expands to other countries:
- DE: No VPN needed (server already in DE)
- BE: Add Belgium VPN endpoint
- FR: Add France VPN endpoint
- Same split tunnel approach, multiple WireGuard interfaces

---

## References

- PIA Manual Connections: https://github.com/pia-foss/manual-connections
- WireGuard Documentation: https://www.wireguard.com/
- SKILL_10_COEFFICIENT_VPN.md: Setup and troubleshooting procedures
- HANDOFF_CC_COEFFICIENT_ENGINE.md: Overall coefficient engine architecture

---

## Changelog

- **2026-01-10**: Initial decision (Claude Code)
  - Implemented split tunneling with WireGuard
  - Tested and verified working
  - SSH access preserved throughout setup
