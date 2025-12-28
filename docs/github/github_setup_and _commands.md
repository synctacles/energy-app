# GitHub Configuration & Setup Guide

**Purpose:** Complete reference for GitHub setup, configuration, and essential git commands  
**Target:** SYNCTACLES development team and contributors  
**Repository:** https://github.com/DATADIO/synctacles-repo  
**Last Updated:** December 5, 2025  

---

## Table of Contents

1. [Overview](#overview)
2. [Prerequisites](#prerequisites)
3. [Initial Setup](#initial-setup)
4. [Git Configuration](#git-configuration)
5. [Essential Commands](#essential-commands)
6. [Workflow](#workflow)
7. [Troubleshooting](#troubleshooting)
8. [Repository Structure](#repository-structure)

---

## Overview

### What This Guide Covers

This guide documents the complete GitHub setup for the SYNCTACLES project repository, including:
- SSH authentication configuration
- Git author identity setup
- Repository folder structure
- Essential git commands with explanations
- Standard development workflows

### About SYNCTACLES Repository

```
Repository Name:  synctacles-repo
GitHub Account:   DATADIO
URL:             git@github.com:DATADIO/synctacles-repo.git
Branch:          main
Access:          SSH (recommended)
```

---

## Prerequisites

### Required Software

Before starting, ensure you have:

1. **Git installed** (version 2.x or higher)
   ```bash
   git --version
   # Expected: git version 2.x.x or higher
   ```

2. **SSH key configured** on your machine
   ```bash
   ls -la ~/.ssh/id_rsa
   # Should show: -rw------- 1 user user ... id_rsa
   ```

3. **SSH key added to GitHub**
   - Visit: https://github.com/settings/keys
   - Should show: "DATADIO Server" (or similar)

### Verify SSH Access

```bash
ssh -T git@github.com

# Expected output:
# Hi DATADIO! You've successfully authenticated, but GitHub does not provide shell access.
```

If you see this message, SSH is working correctly! ✅

---

## Initial Setup

### First-Time Repository Setup

If you're cloning the repository for the first time:

#### Step 1: Clone Repository

```bash
# Clone using SSH (recommended)
git clone git@github.com:DATADIO/synctacles-repo.git

# Navigate into repository
cd synctacles-repo

# Verify successful clone
ls -la
# Should show: config/, docker/, docs/, scripts/, tests/, .github/, .gitignore
```

#### Step 2: Verify Repository Status

```bash
# Check git status
git status
# Expected: On branch main, nothing to commit, working tree clean

# Check commit history
git log --oneline
# Expected: Shows initial commit

# Check remote connection
git remote -v
# Expected: origin git@github.com:DATADIO/synctacles-repo.git (fetch)
#          origin git@github.com:DATADIO/synctacles-repo.git (push)
```

#### Step 3: Verify Git Configuration

```bash
# Check git author
git config user.name
# Expected: DATADIO

git config user.email
# Expected: admin@synctacles.nl

# If not set or incorrect, configure (see Git Configuration section)
```

---

## Git Configuration

### Configure Git Author (IMPORTANT!)

**Why:** Git tracks who made each commit. Author identity is separate from SSH authentication!

### Local Machine Setup

```bash
# Set global configuration (applies to all repositories)
git config --global user.name "DATADIO"
git config --global user.email "admin@synctacles.nl"

# Verify configuration
git config --global --list | grep user
# Expected: user.name=DATADIO
#          user.email=admin@synctacles.nl
```

### Repository-Specific Setup (Optional)

If you need different identity for this repository only:

```bash
cd /opt/github/synctacles-repo

# Set local configuration (only this repository)
git config user.name "DATADIO"
git config user.email "admin@synctacles.nl"

# Verify
git config user.name
# Expected: DATADIO
```

### Important Notes

- **Global vs Local:** Global settings apply to all repositories, local settings override global for specific repo
- **Author Identity:** This is who created the commit (not SSH authentication)
- **Attribution:** Your name appears in commit history on GitHub
- **Standard Practice:** Use your real name or account username

---

## Essential Commands

### 1. Check Repository Status

```bash
git status

# Output explanation:
# On branch main                          ← Your current branch
# Your branch is up to date with 'origin/main'.  ← No unpushed commits
# nothing to commit, working tree clean   ← No uncommitted changes

# If changes exist, output shows:
# Changes not staged for commit:
#   modified:   filename.py
#   deleted:    other_file.py
```

**When to use:** Before making changes, before pushing, or when unsure of state  
**Important:** Always check status before pushing!

---

### 2. View Commit History

```bash
# Show recent commits (one line each)
git log --oneline

# Output example:
# 9944a43 (HEAD -> main, origin/main) Initial: SYNCTACLES project structure
# ab12cd4 Previous commit
# ef34gh5 Earlier commit

# Show detailed commit information
git log -1
# Shows: Author, Date, Full message for most recent commit

# Show commits for specific file
git log -- path/to/file.py

# Show commits in graph format (useful for branches)
git log --oneline --graph --all
```

**When to use:** Reviewing project history, understanding what changed when, verifying commits  
**Explanation:**
- `--oneline`: Compact format (one line per commit)
- `-1`: Show only 1 commit (use `-5` for 5, etc.)
- `--graph`: Visual branch representation
- `--all`: Show all branches

---

### 3. Create and Commit Changes

```bash
# Stage all changes for commit
git add -A

# Or stage specific file
git add path/to/file.py

# Check what's staged
git status
# Should show: Changes to be committed: modified/new/deleted files

# Create commit with message
git commit -m "Short description (50 chars)"

# Or create commit with detailed message
git commit -m "Short description

- Detailed point 1
- Detailed point 2
- Detailed point 3"
```

**When to use:** After making changes to code  
**Best Practices:**
- Write clear, descriptive commit messages
- Commit related changes together
- Use imperative mood: "Add feature" not "Added feature"
- Reference issues if applicable: "Fix #123"

---

### 4. Push Changes to GitHub

```bash
# Push current branch to GitHub
git push origin main

# Or push all branches
git push origin --all

# Push with upstream tracking (first push of branch)
git push -u origin main
# After this, simple 'git push' works
```

**When to use:** After committing, to backup code on GitHub and make available to team  
**Important:**
- Always push to `origin main` unless using feature branches
- Check status before push: `git status`
- Verify no errors in output

**Output explanation:**
```
Enumerating objects: 5, done.         ← Preparing data
Counting objects: 100% (5/5), done.   ← Counting files
Delta compression using up to 4 threads  ← Compressing
Compressing objects: 100% (3/3), done.
Writing objects: 100% (5/5), 653 bytes
To github.com:DATADIO/synctacles-repo.git
 * [new branch]      main -> main     ← Successfully pushed
```

---

### 5. Pull Latest Changes

```bash
# Pull latest changes from GitHub
git pull origin main

# Or simply (if upstream tracking set)
git pull

# Check what was pulled
git log --oneline -1
```

**When to use:** Before starting work, to ensure you have latest code  
**Important:** Pulling before coding prevents conflicts

**Output explanation:**
```
Updating abc1234..def5678            ← Fetching changes
Fast-forward                          ← Type of merge
 file1.py | 10 ++++++++++
 file2.py |  5 +----
 2 files changed, 13 insertions(+), 4 deletions(-)
```

---

### 6. Create Feature Branch

```bash
# Create new branch
git branch feature/my-feature

# Switch to branch
git checkout feature/my-feature

# Or create and switch in one command
git checkout -b feature/my-feature

# Verify current branch
git branch
# Expected: * feature/my-feature

# Push new branch to GitHub
git push -u origin feature/my-feature
```

**When to use:** For major features or bug fixes (optional for small changes)  
**Naming convention:**
- `feature/description` - new features
- `bugfix/description` - bug fixes
- `hotfix/description` - urgent production fixes

---

### 7. Merge Feature Branch

```bash
# Switch to main
git checkout main

# Pull latest
git pull origin main

# Merge feature branch
git merge feature/my-feature

# Push merged changes
git push origin main

# Delete feature branch (optional)
git branch -d feature/my-feature
git push origin --delete feature/my-feature
```

**When to use:** After feature is complete and tested  
**Important:** Always merge to main, test before pushing

---

### 8. Revert Changes

```bash
# Undo uncommitted changes to file
git checkout -- path/to/file.py

# Undo all uncommitted changes
git checkout -- .

# Undo last commit (keep changes)
git reset HEAD~1

# Undo last commit (discard changes)
git reset --hard HEAD~1

# Create new commit that undoes previous commit
git revert HEAD
```

**When to use:** When you made a mistake  
**Important:** Use `reset` for local changes, `revert` for pushed commits

---

## Workflow

### Standard Development Workflow

#### For New Features or Bug Fixes:

```bash
# 1. Start with clean main
git checkout main
git pull origin main

# 2. Create feature branch
git checkout -b feature/new-feature

# 3. Make changes and commit
# ... edit files ...
git add -A
git commit -m "Add new feature

- Description of changes
- What was added
- How it works"

# 4. Push branch
git push -u origin feature/new-feature

# 5. Create Pull Request on GitHub
# (Or notify team)

# 6. After review and approval, merge
git checkout main
git pull origin main
git merge feature/new-feature
git push origin main

# 7. Delete feature branch
git branch -d feature/new-feature
git push origin --delete feature/new-feature
```

#### For Direct Commits to Main (Small Changes):

```bash
# 1. Ensure main is up to date
git checkout main
git pull origin main

# 2. Make changes
# ... edit files ...

# 3. Stage and commit
git add -A
git commit -m "Fix typo in README"

# 4. Push to main
git push origin main

# 5. Verify on GitHub
# Visit: https://github.com/DATADIO/synctacles-repo
```

---

### Best Practices

#### Commit Messages

Good commit messages help with code review and history understanding:

```bash
# ✅ GOOD - Clear and descriptive
git commit -m "Add D3 sparkcrawler data collection

- Implemented ENTSO-E API client
- Added data ingestion scheduler (5-minute intervals)
- Created raw data storage layer"

# ❌ BAD - Too vague
git commit -m "update code"

# ❌ BAD - Too long
git commit -m "Add a lot of functionality to the system including the new API endpoints and the database layer and refactored some stuff"
```

#### Frequency of Commits

```bash
# ✅ GOOD - Logical grouping
# Commit 1: "Implement sparkcrawler API client"
# Commit 2: "Add error handling and retry logic"
# Commit 3: "Write tests for sparkcrawler"

# ❌ BAD - Too many tiny commits
# Commit 1: "Add import"
# Commit 2: "Add class definition"
# Commit 3: "Add method"
```

#### Before Pushing

```bash
# Always check before pushing
git status           # No uncommitted changes?
git log -1           # Is commit message clear?
git diff origin/main # What exactly changed?
```

---

## Troubleshooting

### Issue 1: "fatal: detected dubious ownership in repository"

**Cause:** Git security check (different user created folder vs running git)

**Solution:**
```bash
git config --global --add safe.directory /opt/github/synctacles-repo
```

**Explanation:** Tells git to trust this directory (safe for your use case)

---

### Issue 2: "Permission denied (publickey)"

**Cause:** SSH key not configured or not added to GitHub

**Solution:**
```bash
# Verify SSH key exists
ls ~/.ssh/id_rsa

# If not, generate new key
ssh-keygen -t rsa -b 4096 -f ~/.ssh/id_rsa -N ""

# Get public key
cat ~/.ssh/id_rsa.pub

# Add to GitHub: https://github.com/settings/keys
# New SSH key → Paste output above → Add SSH key
```

**Verification:**
```bash
ssh -T git@github.com
# Expected: Hi DATADIO! You've successfully authenticated...
```

---

### Issue 3: "Author identity unknown"

**Cause:** Git author not configured

**Solution:**
```bash
git config --global user.name "DATADIO"
git config --global user.email "admin@synctacles.nl"

# Verify
git config --global --list | grep user
```

---

### Issue 4: "error: src refspec main does not match any"

**Cause:** Trying to push but no commits exist on branch

**Solution:**
```bash
# Create commits first
git add -A
git commit -m "Initial commit"

# Then push
git push -u origin main
```

---

### Issue 5: "Your branch is ahead of 'origin/main'"

**Cause:** Local commits not yet pushed to GitHub

**Solution:**
```bash
# Push commits
git push origin main

# Verify
git status
# Expected: Your branch is up to date with 'origin/main'
```

---

## Repository Structure

### Current Structure

```
synctacles-repo/                    ← Repository root
├── .github/                        ← GitHub workflows
│   └── workflows/                  ← CI/CD pipelines (future)
│
├── sparkcrawler_db/                ← Phase D3: Raw data collection
│   ├── __init__.py
│   ├── main.py
│   ├── clients/
│   │   ├── entsoe.py             ← ENTSO-E API client
│   │   └── tennet.py             ← TenneT API client
│   ├── tasks/
│   │   └── ingest_data.py        ← Ingestion scheduler
│   └── storage/
│       └── raw_data.py           ← Raw data storage
│
├── synctacles_db/                  ← Phase D4: API & Normalization
│   ├── __init__.py
│   ├── main.py                    ← FastAPI app entry
│   ├── routes/
│   │   ├── generation_mix.py      ← /v1/nl/generation_mix
│   │   ├── load.py                ← /v1/nl/load
│   │   └── prices.py              ← /v1/nl/prices
│   ├── models/
│   │   ├── orm.py                 ← SQLAlchemy ORM
│   │   └── schemas.py             ← Pydantic schemas
│   └── db/
│       ├── queries.py             ← Database queries
│       └── normalization.py       ← Data transformation
│
├── docker/                         ← Container configuration
│   ├── Dockerfile
│   ├── docker-compose.yml
│   └── entrypoint.sh
│
├── scripts/                        ← Automation scripts
│   ├── setup/
│   │   └── setup_synctacles_server_v1.9.sh
│   ├── deploy/
│   │   ├── deploy_d3.sh           ← Deploy D3 only
│   │   ├── deploy_d4.sh           ← Deploy D4 only
│   │   └── deploy_all.sh          ← Deploy both
│   ├── test/
│   │   ├── test_api.sh
│   │   └── validate_setup.sh
│   └── tools/
│       ├── backup_db.sh
│       ├── health_check.sh
│       └── cleanup.sh
│
├── tests/                          ← Test suite
│   ├── __init__.py
│   ├── test_api.py                ← D4 API tests
│   ├── test_crawler.py            ← D3 crawler tests
│   └── conftest.py                ← Pytest configuration
│
├── config/                         ← Configuration templates
│   ├── .env.example               ← Environment variables
│   └── settings.template.py       ← Settings template
│
├── docs/                           ← Documentation
│   ├── README.md
│   ├── GITHUB.md                  ← This file
│   ├── API.md                     ← API reference
│   ├── ARCHITECTURE.md            ← System design
│   ├── SETUP.md                   ← Installation guide
│   ├── DEPLOYMENT.md              ← Deployment procedures
│   └── TROUBLESHOOTING.md         ← Common issues
│
├── .gitignore                      ← Git ignore rules
├── README.md                       ← Project overview
├── requirements.txt                ← Python dependencies
└── LICENSE                         ← License (if applicable)
```

### Folder Purposes

| Folder | Purpose | D3/D4 |
|--------|---------|-------|
| `sparkcrawler_db/` | Raw data ingestion from ENTSO-E/TenneT | D3 |
| `synctacles_db/` | API and data normalization layer | D4 |
| `docker/` | Container configuration and orchestration | Both |
| `scripts/` | Deployment and utility automation | Both |
| `tests/` | Test suite for both layers | Both |
| `docs/` | Documentation (this file!) | Reference |
| `config/` | Configuration templates | Reference |

---

## Quick Reference

### Most Common Commands

```bash
# Check status
git status

# Make changes and commit
git add -A
git commit -m "Description"

# Push to GitHub
git push origin main

# Pull latest changes
git pull origin main

# View history
git log --oneline

# Create feature branch
git checkout -b feature/name

# Switch branches
git checkout main
git checkout feature/name

# Merge branch to main
git merge feature/name
```

---

## Summary

### What You Need to Know

1. **Repository URL:** `git@github.com:DATADIO/synctacles-repo.git`
2. **Default branch:** `main`
3. **Git author:** DATADIO
4. **Access method:** SSH
5. **Main folder structure:** sparkcrawler_db (D3), synctacles_db (D4), docker, scripts, tests

### Four Basic Commands to Remember

```bash
git pull        # Get latest code
git add -A      # Stage changes
git commit -m   # Save changes locally
git push        # Send to GitHub
```

### Before Each Work Session

```bash
# 1. Pull latest
git pull origin main

# 2. Make changes
# ... edit files ...

# 3. Check what changed
git status

# 4. Commit and push
git add -A
git commit -m "Description of changes"
git push origin main
```

---

## Additional Resources

- **Git Documentation:** https://git-scm.com/doc
- **GitHub Help:** https://docs.github.com/
- **Conventional Commits:** https://www.conventionalcommits.org/
- **GitHub SSH Setup:** https://docs.github.com/en/authentication/connecting-to-github-with-ssh

---

**Last Updated:** December 5, 2025  
**Repository:** https://github.com/DATADIO/synctacles-repo  
**Questions?** Check Troubleshooting section or documentation

