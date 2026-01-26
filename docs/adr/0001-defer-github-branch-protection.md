# ADR-0001: Defer GitHub Branch Protection Until Traction Achieved

**Status:** Accepted
**Date:** 2026-01-26
**Deciders:** Leo (Product Owner)
**Technical context:** Claude Code (CC)

---

## Context

Na implementatie van comprehensive security hardening (commit 9cdb166) met:
- Pre-commit hooks (SQL injection, secrets, direct main commits)
- CI/CD security pipeline (Bandit, pip-audit, Safety)
- Release checklist met security gate
- SKILL_05_SECURITY documentation

...wilden we GitHub branch protection configureren als extra laag van defense-in-depth.

### Discovery

GitHub branch protection rules vereisen GitHub Team ($4/user/maand) om gehandhaafd te worden op private repositories. Beide implementaties:
- **Rulesets** (moderne methode) - "won't be enforced on this private repository until you upgrade"
- **Branch protection rules** (klassieke methode) - "Not enforced" label

### Current Security Posture

Zonder GitHub branch protection hebben we nog steeds:

**Layer 1: Pre-commit (Lokaal)**
- ✅ Blokkeert SQL injection patterns
- ✅ Blokkeert hardcoded secrets
- ✅ Blokkeert direct commits to main
- ✅ Enforces code formatting (Ruff)

**Layer 2: CI/CD (GitHub Actions)**
- ✅ Bandit static security analysis
- ✅ pip-audit dependency vulnerabilities
- ✅ Safety vulnerability database check
- ✅ pytest test suite

**Layer 3: Release Gate (Handmatig)**
- ✅ Security checklist in docs/RELEASE_CHECKLIST.md
- ✅ Manual review vereist voor deploy

**Layer 4: Pre-push (Care add-on)**
- ✅ Automated test execution before push
- ✅ Blocks push if tests fail

### Risk Assessment

**Zonder GitHub branch protection:**
- ⚠️ Developer kan pre-commit hooks bypassen met `git commit --no-verify`
- ⚠️ Developer kan direct naar main pushen als local hooks disabled zijn
- ✅ CI/CD draait ALTIJD, ook bij bypass van lokale checks
- ✅ Solo developer (Leo) met discipline = laag risico

**Met GitHub branch protection ($48/jaar):**
- ✅ Technische enforcement van PR requirement
- ✅ Status checks MOETEN passen voor merge
- ✅ Extra laag tegen menselijke fouten
- ❌ Kosten voor feature met beperkte ROI in solo context

---

## Decision

**Defer GitHub Team subscription** (en dus branch protection) tot het product voldoende traction heeft.

**Rationale:**
1. **Huidige security posture is adequaat** - Pre-commit + CI/CD biedt degelijke borging
2. **Solo developer context** - Geen teamleden die onbedoeld rules kunnen bypassen
3. **Cost/benefit** - $48/jaar is niet veel, maar ROI is laag zonder team
4. **Traction milestone** - Bij groei (team, klanten, revenue) wordt GitHub Team sowieso aantrekkelijk
5. **Reversible decision** - Branch protection rules zijn al voorbereid, alleen subscription nodig

**Compenserende controls:**
- SKILL_01_HARD_RULES: "NEVER use --no-verify" is gedocumenteerd
- Pre-commit hooks blijven actief en worden getest
- CI/CD pipeline blijft mandatory gate
- Security audit quarterly review (Q2 2026)

---

## Consequences

### Positive

- ✅ Geen maandelijkse kosten ($4/maand bespaard)
- ✅ Focus op product development ipv infrastructure overhead
- ✅ Security borging is al structureel via pre-commit + CI/CD
- ✅ Branch protection rules zijn voorbereid voor toekomstige activatie

### Negative

- ⚠️ Geen technische enforcement van PR requirement
- ⚠️ Developer kan bewust pre-commit hooks bypassen
- ⚠️ Menselijke discipline vereist (geen technical safeguard)

### Mitigation

- 📋 GitHub issue aangemaakt: "Evaluate GitHub Team for branch protection"
- 📋 Review trigger: Bij eerste externe contributor OF 100 users OR €500 MRR
- 📋 Quarterly security review checklist: Verify no --no-verify commits in history
- 📋 CI/CD failure = immediate investigation (treated as P0)

---

## Alternatives Considered

### A) Maak repositories public
- ✅ Branch protection gratis beschikbaar
- ❌ Proprietary code publiek zichtbaar
- ❌ Competitive disadvantage
- **Rejected:** Privacy > free branch protection

### B) Neem GitHub Team subscription nu
- ✅ Maximale technical enforcement
- ✅ Future-proof voor team groei
- ❌ Overkill voor solo developer
- ❌ Premature optimization
- **Rejected:** Wait for traction

### C) Accepteer status quo zonder compensatie
- ✅ Geen extra werk
- ❌ Geen monitoring van --no-verify usage
- ❌ Geen trigger voor re-evaluation
- **Rejected:** Need compensating controls

---

## Review Triggers

Re-evaluate this decision when ANY of these occurs:

1. **Team growth:** Eerste externe developer/contributor
2. **Traction:** 100+ active users OF €500 MRR
3. **Security incident:** Pre-commit bypass leidt tot productie issue
4. **Quarterly review:** Q2 2026 (April 1, 2026)
5. **GitHub policy change:** Branch protection gratis voor private repos

---

## Related

- **Implementation:** Commit 9cdb166 (security hardening)
- **Documentation:** [docs/skills/core/SKILL_05_SECURITY.md](../skills/core/SKILL_05_SECURITY.md)
- **GitHub Issue:** [#93](https://github.com/synctacles/backend/issues/93) - Evaluate GitHub Team subscription
- **Release checklist:** [docs/RELEASE_CHECKLIST.md](../RELEASE_CHECKLIST.md)

---

## Notes

**Pre-configured branch protection settings (ready to activate):**
```yaml
Branch: main
Rules:
  - Require pull request before merging: true
  - Require status checks: ["security"]
  - Require branches to be up to date: true
  - Block force pushes: true
  - Do not allow bypassing: true
```

**Activation command when ready:**
```bash
# 1. Subscribe to GitHub Team ($4/user/month)
# 2. Go to Settings → Branches → Add rule
# 3. Apply above settings
# 4. Test with: git push origin main (should fail)
# 5. Test with: PR → security check must pass
```

---

*"Perfect is the enemy of good. Ship with adequate security, iterate when needed."*
