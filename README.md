# Synctacles Platform

Shared infrastructure and authentication for all Synctacles products.

## Products

This repository contains platform-level infrastructure. Product-specific code has been extracted to dedicated repositories:

- **Energy API:** [synctacles/energy](https://github.com/synctacles/energy) - Dutch electricity price API
- **Care (Support Bot):** [synctacles/care](https://github.com/synctacles/care) - Customer support automation
- **Brains (AI/ML):** [synctacles/brains](https://github.com/synctacles/brains) - Embedding generation and ML infrastructure

## This Repository

**Platform** contains shared infrastructure and future authentication services:

- Shared libraries (future: `platform/shared/`)
- Authentication service (future: `platform/auth-service/`)
- Infrastructure as Code
- System-level documentation
- Cross-product deployment scripts

## Repository Migration

Energy code was extracted from this repository in **February 2025** (Issue #143).

### What Moved

- `synctacles_db/` → `energy_api/` in [synctacles/energy](https://github.com/synctacles/energy)
- Energy-specific scripts, docs, tests, and configuration
- Database migrations (Alembic)
- Systemd service files

### What Remains

- Platform-level documentation (`docs/skills/infrastructure/`, `docs/skills/core/`)
- Deployment and monitoring scripts (`scripts/deploy/`, `scripts/monitoring/`)
- Shared configuration templates
- Cross-product CI/CD workflows

## Future Structure

```
platform/
├── auth-service/       # Centralized authentication (Issue #121)
├── shared/             # Shared Python libraries
├── infrastructure/     # IaC, deploy scripts
└── docs/              # System-level documentation
```

## Related Documentation

See [CLAUDE.md](CLAUDE.md) for Claude Code operating guidelines.

For product-specific documentation, see respective repositories.

## License

MIT - See LICENSE file
