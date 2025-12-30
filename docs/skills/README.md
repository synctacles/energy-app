# Skills Documentation

Comprehensive technical and operational documentation for the SYNCTACLES energy insights platform.

---

## Overview

The SYNCTACLES project is organized around **8 core Skills** that document every aspect of the system: from hard rules and architecture to specific implementation details, deployment procedures, and hardware requirements.

Think of Skills as the "knowledge base" of the project—everything a developer, operator, or contributor needs to understand how the system works and how to work with it.

---

## All Skills

### SKILL 1 — HARD RULES
**File:** [SKILL_01_HARD_RULES.md](./SKILL_01_HARD_RULES.md)
- Fundamental non-negotiable rules
- KISS, Fail-Fast, Brand-Free principles
- Template system with {{PLACEHOLDER}}
- Environment variables and configuration
- Data quality metadata standards
- For: Everyone on the project

### SKILL 2 — SYSTEM ARCHITECTURE  
**File:** [SKILL_02_ARCHITECTURE.md](./SKILL_02_ARCHITECTURE.md)
- 3-layer data pipeline design
- Components: Collectors, Importers, Normalizers, API
- Multi-tenant deployment model
- Database schema with quality metadata
- Fallback strategy and observability
- For: Architects and developers

### SKILL 3 — CODING STANDARDS
**File:** [SKILL_03_CODING_STANDARDS.md](./SKILL_03_CODING_STANDARDS.md)
- Python PEP 8 style guide
- Fail-fast error handling
- Code comments and docstrings
- Configuration management patterns
- Testing and logging standards
- For: Python developers

### SKILL 4 — PRODUCT REQUIREMENTS
**File:** [SKILL_04_PRODUCT_REQUIREMENTS.md](./SKILL_04_PRODUCT_REQUIREMENTS.md)
- What SYNCTACLES does (features, capabilities)
- Real-time generation, load, prices, balance data
- Home Assistant integration
- Roadmap and success metrics
- For: Product managers and users

### SKILL 5 — COMMUNICATION RULES
**File:** [SKILL_05_COMMUNICATION_RULES.md](./SKILL_05_COMMUNICATION_RULES.md)
- Error message structure: [WHAT] - [WHY] - [HOW TO FIX]
- Code comments and documentation style
- Commit message format
- Naming conventions
- Team communication guidelines
- For: Everyone writing code or docs

### SKILL 6 — DATA SOURCES
**File:** [SKILL_06_DATA_SOURCES.md](./SKILL_06_DATA_SOURCES.md)
- ENTSO-E (generation, load, prices)
- TenneT (grid frequency, reserves)
- Energy-Charts (fallback/modeled data)
- API details, rate limits, reliability
- Data quality scoring and fallback strategy
- For: Developers working with data

### SKILL 7 — PERSONAL PROFILE
**File:** [SKILL_07_PERSONAL_PROFILE.md](./SKILL_07_PERSONAL_PROFILE.md)
- ⚠️ **PERSONAL INFORMATION - NOT IN PUBLIC REPO**
- Excluded via .gitignore
- Project lead bio, role, communication style
- For: Direct team members only

### SKILL 8 — HARDWARE PROFILE
**File:** [SKILL_08_HARDWARE_PROFILE.md](./SKILL_08_HARDWARE_PROFILE.md)
- System requirements (CPU, RAM, storage)
- Supported operating systems
- Network configuration and ports
- Performance benchmarks
- Scaling considerations
- Infrastructure as Code (Docker, Terraform)
- For: DevOps and system administrators

### SKILL 9 — INSTALLER SPECS
**File:** [SKILL_09_INSTALLER_SPECS.md](./SKILL_09_INSTALLER_SPECS.md)
- FASE 0-6 installation workflow
- Brand-free template system implementation
- Environment variable configuration
- Python fail-fast patterns
- For: DevOps and installers

### SKILL 10 — DEPLOYMENT WORKFLOW
**File:** [SKILL_10_DEPLOYMENT_WORKFLOW.md](./SKILL_10_DEPLOYMENT_WORKFLOW.md)
- 6-phase deployment process
- Pre-deploy validation and backup
- File syncing and migrations
- Post-deploy validation and rollback
- Emergency procedures
- For: Operations and release managers

### SKILL 12 — BRAND-FREE ARCHITECTURE
**File:** [SKILL_12_BRAND_FREE_ARCHITECTURE.md](./SKILL_12_BRAND_FREE_ARCHITECTURE.md)
- Multi-tenant deployment patterns
- Template system philosophy
- Regional instances, white-label SaaS, environment segregation
- Security and best practices
- Testing and migration strategies
- For: Architects and DevOps

---

## Recommended Reading Order

### New Team Members
1. SKILL 1 - Rules (5 min)
2. SKILL 2 - Architecture (20 min)
3. SKILL 4 - Product (10 min)
4. SKILL 5 - Communication (10 min)
5. Others as needed

### Developers
1. SKILL 1 - Rules (mandatory)
2. SKILL 3 - Coding Standards (mandatory)
3. SKILL 2 - Architecture
4. SKILL 5 - Communication
5. SKILL 6 - Data Sources

### DevOps/Infrastructure
1. SKILL 1 - Rules
2. SKILL 8 - Hardware
3. SKILL 9 - Installer
4. SKILL 10 - Deployment
5. SKILL 12 - Brand-Free

---

## Quick Reference

**Looking for...** → **See SKILL...**
- Code standards → SKILL 3
- Deployment steps → SKILL 9, 10
- Features → SKILL 4
- Architecture → SKILL 2
- Rules → SKILL 1
- Documentation style → SKILL 5
- Data sources → SKILL 6
- Hardware requirements → SKILL 8

---

## Version History

- **v2.0** (2025-12-30): Complete suite (SKILL 1-12)
- **v1.0** (2025-12-30): Initial skills (SKILL 9, 10, 12)
