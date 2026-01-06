# 🚀 Operations & Deployment Documentation

Complete guides for deploying, monitoring, and operating SYNCTACLES.

---

## 📋 Documents in This Folder

### **[DEPLOYMENT_SCRIPTS_GUIDE.md](DEPLOYMENT_SCRIPTS_GUIDE.md)**
Complete reference for all deployment automation scripts:
- Deploy script walkthrough
- Rollback procedures
- Backup strategies
- Health check validation
- Pre-deployment checks

**Read this when:** Deploying to production or setting up new servers

---

## 🔗 Related Documentation

### Deployment & Workflow
- [../skills/SKILL_10_DEPLOYMENT_WORKFLOW.md](../skills/SKILL_10_DEPLOYMENT_WORKFLOW.md) - Deployment workflow procedures
- [../skills/SKILL_09_INSTALLER_SPECS.md](../skills/SKILL_09_INSTALLER_SPECS.md) - Server installation specifications

### Monitoring & Logging
- [../skills/SKILL_13_LOGGING_DIAGNOSTICS_HA_STANDARDS.md](../skills/SKILL_13_LOGGING_DIAGNOSTICS_HA_STANDARDS.md) - Monitoring standards & diagnostics

### Issue Resolution
- [../incidents/](../incidents/) - Past incidents & how they were solved
- [../CC_communication/INDEX.md](../CC_communication/INDEX.md) - Issue tracking system

### Architecture
- [../ARCHITECTURE.md](../ARCHITECTURE.md) - System design (needed to understand what to deploy)

---

## 🎯 Quick Navigation

### **"How do I deploy code?"**
→ Read: [DEPLOYMENT_SCRIPTS_GUIDE.md](DEPLOYMENT_SCRIPTS_GUIDE.md)

### **"What's the deployment workflow?"**
→ Read: [../skills/SKILL_10_DEPLOYMENT_WORKFLOW.md](../skills/SKILL_10_DEPLOYMENT_WORKFLOW.md)

### **"How do I set up a new server?"**
→ Read: [../skills/SKILL_09_INSTALLER_SPECS.md](../skills/SKILL_09_INSTALLER_SPECS.md)

### **"How do I monitor the system?"**
→ Read: [../skills/SKILL_13_LOGGING_DIAGNOSTICS_HA_STANDARDS.md](../skills/SKILL_13_LOGGING_DIAGNOSTICS_HA_STANDARDS.md)

### **"What do I do if something breaks?"**
→ Check: [../incidents/](../incidents/) for past issues
→ Check: [../CC_communication/INDEX.md](../CC_communication/INDEX.md) for known issues

---

## ⚙️ Common Operations

### Deploy Latest Code
```bash
scripts/deploy.sh
```
See: [DEPLOYMENT_SCRIPTS_GUIDE.md](DEPLOYMENT_SCRIPTS_GUIDE.md)

### Rollback to Previous Version
```bash
scripts/rollback.sh
```
See: [DEPLOYMENT_SCRIPTS_GUIDE.md](DEPLOYMENT_SCRIPTS_GUIDE.md)

### Backup Database
```bash
scripts/maintenance/backup_database.sh
```
See: [DEPLOYMENT_SCRIPTS_GUIDE.md](DEPLOYMENT_SCRIPTS_GUIDE.md)

### Health Check
```bash
curl http://localhost:8000/health
```
See: [DEPLOYMENT_SCRIPTS_GUIDE.md](DEPLOYMENT_SCRIPTS_GUIDE.md)

---

## 📊 Key Operational Files

| File | Purpose |
|------|---------|
| `/scripts/deploy.sh` | Automated deployment pipeline |
| `/scripts/rollback.sh` | Rollback to previous version |
| `/scripts/maintenance/backup_database.sh` | Database backup |
| `/scripts/maintenance/health_check.sh` | System health validation |
| `/deployment/sync/*/DEPLOY.md` | Specific deployment procedures |

---

## 🔍 System Status & Monitoring

**Health Endpoint:**
```bash
GET /health
```

**Monitoring:**
- Prometheus metrics at `/metrics`
- Application logs in `/var/log/synctacles-api/`
- Structured logging for all components

See: [../skills/SKILL_13_LOGGING_DIAGNOSTICS_HA_STANDARDS.md](../skills/SKILL_13_LOGGING_DIAGNOSTICS_HA_STANDARDS.md)

---

## 🚨 Incident Response

### If System is Down:
1. Check `/health` endpoint
2. Review logs: `/var/log/synctacles-api/`
3. Check database connectivity
4. See [../incidents/](../incidents/) for similar past issues
5. Rollback if needed: `scripts/rollback.sh`

### If Data is Stale:
1. Check collector logs
2. Check normalizer logs
3. Verify external API availability (ENTSO-E, Energy-Charts)
4. See [../CC_communication/INDEX.md](../CC_communication/INDEX.md) for data gap issues

### If Performance is Slow:
1. Check database query performance
2. Check cache hit rates
3. Review [../reports/LOAD_TEST_REPORT.md](../reports/LOAD_TEST_REPORT.md) for baselines
4. See [../reports/OPTIMIZATION_RESULTS.md](../reports/OPTIMIZATION_RESULTS.md) for optimizations

---

## 📞 For More Information

- **Deployment procedures:** [DEPLOYMENT_SCRIPTS_GUIDE.md](DEPLOYMENT_SCRIPTS_GUIDE.md)
- **Deployment workflow:** [../skills/SKILL_10_DEPLOYMENT_WORKFLOW.md](../skills/SKILL_10_DEPLOYMENT_WORKFLOW.md)
- **Server setup:** [../skills/SKILL_09_INSTALLER_SPECS.md](../skills/SKILL_09_INSTALLER_SPECS.md)
- **Monitoring:** [../skills/SKILL_13_LOGGING_DIAGNOSTICS_HA_STANDARDS.md](../skills/SKILL_13_LOGGING_DIAGNOSTICS_HA_STANDARDS.md)
- **Past incidents:** [../incidents/](../incidents/)

---

**Last Updated:** January 6, 2026
**Status:** Complete & Current
