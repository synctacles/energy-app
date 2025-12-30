# Skills Documentation

This directory contains technical skills and specifications for the SYNCTACLES energy insights platform. These skills document architectural decisions, installation procedures, and deployment workflows.

---

## Skills Overview

### SKILL 9 — SYNCTACLES INSTALLER SPECS

**File:** [SKILL_09_INSTALLER_SPECS.md](./SKILL_09_INSTALLER_SPECS.md)

Server installation script specifications for the brand-free template system. Covers FASE 0-6 installation workflow with ENV-driven configuration that enables multi-tenant deployments without git conflicts.

**Key Topics:**
- Brand-free repository principles
- FASE 0: Interactive brand configuration
- Directory structure (repo vs runtime)
- Python fail-fast patterns
- Validation checklists
- Migration from branded repositories

**Use When:** Setting up a fresh SYNCTACLES installation or understanding the installation architecture.

---

### SKILL 10 — DEPLOYMENT WORKFLOW

**File:** [SKILL_10_DEPLOYMENT_WORKFLOW.md](./SKILL_10_DEPLOYMENT_WORKFLOW.md)

Deployment strategy for moving code from development to production. Defines the complete workflow including pre-deploy validation, backup, file syncing, post-deploy actions, and rollback procedures.

**Key Topics:**
- DEV → PROD deployment process (6 phases)
- Sync manifest format
- Version tracking and semantic versioning
- Emergency procedures (API down, migration failed, service issues)
- Pre-deploy checklist
- Feature deployment workflow

**Use When:** Deploying changes to production or understanding the deployment architecture.

---

### SKILL 12 — BRAND-FREE TEMPLATE ARCHITECTURE

**File:** [SKILL_12_BRAND_FREE_ARCHITECTURE.md](./SKILL_12_BRAND_FREE_ARCHITECTURE.md)

Architectural principles for brand-agnostic repository design enabling multi-tenant deployments. Explains the philosophy, patterns, and best practices for maintaining a single codebase deployed with different branding across multiple servers.

**Key Topics:**
- Core principles (template-based, ENV-driven, fail-fast)
- Template system with {{PLACEHOLDER}} format
- Python fail-fast patterns
- .gitignore strategy
- Multi-tenant deployment patterns (regional, white-label, environment segregation)
- Template generation strategies
- Testing brand-free code
- Security considerations
- Best practices and code review checklist

**Use When:** Designing multi-tenant deployments or understanding the brand-free architecture philosophy.

---

## Skill Dependencies

```
SKILL 12 (Brand-Free Architecture)
    ↓
    ├─→ SKILL 9 (Installer Specs)
    │
    └─→ SKILL 10 (Deployment Workflow)
```

**Recommended Reading Order:**
1. **SKILL 12** - Understand the why and philosophy
2. **SKILL 9** - Implement the installation process
3. **SKILL 10** - Execute the deployment workflow

---

## Key Concepts

### Brand-Free Repository
The repository contains only generic code and templates without specific brand names, domains, or configuration values. All branding is injected at installation time via `.env` configuration.

### Template System
Uses `{{VARIABLE_NAME}}` placeholders in template files that are processed into generated files (`.env`, `manifest.json`) during installation (FASE 0).

### Environment-Driven Configuration
All configuration values come from environment variables defined in `/opt/.env`. The Python codebase uses fail-fast patterns to raise errors when required variables are missing.

### Multi-Tenant Deployment
The same repository code can be deployed to multiple servers/tenants with different branding and configurations, each determined by their own `.env` file.

### Installation Phases (FASE 0-6)

- **FASE 0**: Interactive brand configuration (generates .env and manifest.json)
- **FASE 1**: System updates
- **FASE 2**: Software stack installation
- **FASE 3**: Security configuration
- **FASE 4**: Python environment setup
- **FASE 5**: Production services
- **FASE 6**: Development tools (optional)

---

## Common Tasks

### Fresh Installation
1. Clone the brand-free repository
2. Run `FASE 0` - answer prompts for brand configuration
3. Run `FASE 1-6` - install and configure services
4. Verify installation with validation checklist

### Deploying Changes
1. Commit changes to git
2. Run pre-deploy checks
3. Create backup of current production
4. Sync files according to sync-manifest.txt
5. Run database migrations
6. Restart services
7. Validate with health checks
8. Rollback if validation fails

### Adding a New Tenant
1. Create new .env file with tenant-specific configuration
2. Generate manifest.json from template
3. Deploy using the same installer (FASE 0-6)
4. Each tenant is independently configured and isolated

---

## Architecture Diagram

```
Git Repository (Brand-Free)
├── Code (generic, no hardcoded branding)
├── Templates (.template files)
└── Configuration Templates (.env.example)
         ↓
    FASE 0 (Interactive)
         ↓
    Generated Configuration
    ├── .env (tenant-specific)
    └── manifest.json (generated)
         ↓
    FASE 1-6 (Installation)
         ↓
    Deployed Tenant Instance
    ├── Tenant A (Brand A, Domain A)
    ├── Tenant B (Brand B, Domain B)
    └── Tenant C (Brand C, Domain C)
```

---

## Security Considerations

- **Never commit .env** files to git (use .gitignore)
- **Never commit generated** manifest.json to git
- **Protect .env** file permissions (chmod 600)
- **Use fail-fast patterns** to prevent misconfiguration
- **Validate input** before template processing
- **Store secrets** in .env, never in code

---

## Related Documentation

- Installer specification: See SKILL 9
- Deployment procedures: See SKILL 10
- Brand-free architecture: See SKILL 12
- Coding standards: See SKILL 3
- System architecture: See SKILL 2

---

## Version History

- **v3.0** (2025-12-28): Brand-free template system with FASE 0
- **v2.0** (2025-12-21): Deployment workflow formalization
- **v1.0** (2025-12-21): Initial skill documentation
