# SKILL 5 — COMMUNICATION RULES

How to Structure Messages, Documentation, and Team Communication
Version: 1.0 (2025-12-30)

---

## PURPOSE

Define how we communicate about SYNCTACLES—in code comments, documentation, error messages, team discussions, and external communications. Consistent communication prevents misunderstandings and makes knowledge transferable.

---

## ERROR MESSAGES

### Structure

Every error message must follow this structure:

```
[WHAT] - [WHY] - [HOW TO FIX]
```

**[WHAT]:** Describe the problem in user terms
**[WHY]:** Explain why this is a problem
**[HOW TO FIX]:** Provide actionable steps

### Examples

✅ **Good:**
```python
raise ValueError(
    "BRAND_NAME environment variable not set.\n"
    "This is required to identify your deployment.\n"
    "Set it in /opt/.env or run 'sudo ./scripts/setup/setup.sh fase0'"
)
```

✅ **Good:**
```python
raise FileNotFoundError(
    f"Configuration file not found: {config_path}\n"
    f"Expected .env file in installation directory.\n"
    f"Create it by running FASE 0 setup script with: sudo ./setup.sh fase0"
)
```

❌ **Bad:**
```python
raise ValueError("Invalid config")
raise Exception("Something went wrong")
raise RuntimeError("Error in import")
```

---

## CODE COMMENTS

### Comment Style

Comments explain **WHY**, not **WHAT**.

✅ **Good (explains why):**
```python
# Reject data older than 4 hours: ENTSO-E publishes every 15 min,
# so > 4 hours indicates a collection failure. Better to use fallback.
if age_minutes > 240:
    return fallback_data()
```

❌ **Bad (explains what the code does):**
```python
# Calculate age in minutes
age_minutes = (datetime.now() - timestamp).total_seconds() / 60
# Check if older than 240 minutes
if age_minutes > 240:
    # Return fallback data
    return fallback_data()
```

### When to Comment

Comment **non-obvious decisions**:

- Why we chose a specific threshold (240 vs 300 minutes)
- Why we use an algorithm instead of a simpler approach
- Why we catch a specific exception type
- Why something looks wrong but is actually correct

Don't comment **obvious code**:

- `x = x + 1` doesn't need "increment x"
- Loop iteration doesn't need "iterate through items"
- Variable assignment doesn't need "set y to value"

---

## DOCUMENTATION

### Document Structure

Every document should have:

1. **Title & Version**
   ```markdown
   # SKILL 5 — COMMUNICATION RULES

   Version: 1.0 (2025-12-30)
   ```

2. **Purpose** (1-2 sentences explaining why this doc exists)
   ```markdown
   ## PURPOSE

   Define how we communicate about SYNCTACLES...
   ```

3. **Table of Contents** (for long documents)
   ```markdown
   1. [Error Messages](#error-messages)
   2. [Code Comments](#code-comments)
   ```

4. **Content** (organized by topic)
   - Use clear headings (##, ###)
   - Use examples (Good/Bad)
   - Use code blocks with language specified

5. **Related Skills** (link to relevant documentation)
   ```markdown
   ## RELATED SKILLS
   - SKILL 1: Hard Rules
   - SKILL 3: Coding Standards
   ```

### Markdown Formatting

Use consistent markdown:

```markdown
# H1 - Document Title
## H2 - Major Section
### H3 - Subsection
#### H4 - Details

**Bold** for emphasis
`code` for inline code

[Link text](url) for links

- Bullet lists
  - Nested items
  - More items

1. Numbered lists
2. Second item
```

### Code Examples

Always label examples as Good/Bad:

```markdown
✅ **Good:**
[example code]

❌ **Bad:**
[counter-example]
```

Include language specification in code blocks:

````markdown
```python
# Python code
def my_function():
    pass
```

```bash
# Shell commands
git commit -m "message"
```
````

---

## COMMIT MESSAGES

### Message Format

Use conventional commits:

```
<type>: <description>

<body>

<footer>
```

**Type:** One of:
- `feat` - New feature
- `fix` - Bug fix
- `docs` - Documentation
- `refactor` - Code reorganization
- `test` - Test changes
- `chore` - Build, dependencies, etc.

### Examples

✅ **Good:**
```
feat: Add fallback to Energy-Charts when ENTSO-E unavailable

When ENTSO-E API is down or slow, use Energy-Charts modeled data.
This prevents gaps in generation data during API outages.

Implements automatic fallback in normalizers layer with quality
scoring to indicate data reliability.

Closes #42
```

✅ **Good:**
```
fix: Fail-fast when BRAND_NAME environment variable missing

Previously, system would run with default brand name, causing
misconfiguration to go undetected until runtime.

Now raises ValueError with clear instructions at startup.
```

❌ **Bad:**
```
Fixed stuff
Update code
more improvements
```

---

## TECHNICAL DOCUMENTATION

### Architecture Document Style

When documenting architecture:

1. **Start with the big picture**
   ```
   What is this system? What problem does it solve?
   ```

2. **Show component diagram**
   ```
   ASCII diagram showing how pieces fit together
   ```

3. **Explain each layer/component**
   ```
   - What it does
   - Why it exists
   - How it fits in the bigger picture
   ```

4. **Provide concrete examples**
   ```
   Example data flow through the system
   ```

5. **Document trade-offs**
   ```
   Why we chose this approach over alternatives
   ```

### API Documentation

For REST APIs:

- **Endpoint:** `GET /v1/generation/current`
- **Description:** Current electricity generation by fuel type
- **Response:**
  ```json
  {
    "timestamp": "...",
    "mix": { ... },
    "quality": 0.95
  }
  ```
- **Error responses:**
  ```json
  {
    "error": "Database unavailable",
    "status": 503
  }
  ```

---

## TERMINAL OUTPUT

### User-Facing Messages

Be clear and friendly:

```bash
# Good: Clear progress, actionable next steps
Setting up brand configuration...
  ✓ BRAND_NAME: Energy Insights NL
  ✓ DATABASE: synctacles_ei_nl
  ✓ SERVICE USER: ei-nl

Next steps:
  1. Run: sudo ./setup.sh fase1
  2. Wait for installation to complete
  3. Check status: sudo systemctl status ei-nl-api
```

```bash
# Bad: Cryptic output
setting up
doing thing 1
doing thing 2
done
```

### Logging Output

Use structured logging:

```bash
# Good: timestamp, level, component, message
2025-12-30 10:15:23 [INFO] collectors.entso_e: Fetched 150 records from A75
2025-12-30 10:15:24 [INFO] importers.entso_e_a75: Inserted 150 records to raw table
2025-12-30 10:15:25 [DEBUG] normalizers.generation: Calculated quality score: 0.98

# Bad: unclear timestamps and components
Fetched data
OK
Success
```

---

## TEAM COMMUNICATION

### Status Updates

When reporting status or issues:

1. **What is the situation?**
2. **What caused it?**
3. **What are the impacts?**
4. **What are we doing about it?**
5. **When will it be fixed?**

**Example:**
```
ISSUE: API response times degraded (p95: 500ms, target: 100ms)

CAUSE: Database query optimization issue in generation endpoint.
Normalizer is running full table scan instead of indexed lookup.

IMPACT: Users reporting slow dashboard loads. Automation triggers
delayed by 1-2 seconds.

ACTION: Investigating database indexes. Estimated fix: 2 hours.

WORKAROUND: Restart API service to clear query cache (temporary fix).
```

### Pull Request Descriptions

Describe changes clearly:

```markdown
## Summary
Add fallback to Energy-Charts when ENTSO-E is down

## Changes
- New `energy_charts_client.py` in collectors/
- Updated normalizers to try Energy-Charts if ENTSO-E fails
- Added quality scoring to distinguish primary vs fallback data

## Testing
- Added unit tests for fallback logic
- Tested with ENTSO-E API mocked as unavailable
- Verified quality scores are < 0.8 for fallback data

## Related Issues
Closes #42

## Deployment Notes
No database migrations. No new environment variables.
Backward compatible.
```

---

## NAMING CONVENTIONS

### Variables & Functions

Use clear, descriptive names:

```python
# Good: clear what it is
generation_mw = 2500
collect_entso_e_data()
is_data_stale()

# Bad: abbreviated or vague
gen = 2500
get_data()
check()
```

### Files & Directories

Use snake_case:

```
good_file_name.py
bad_FileName.py
bad-file-name.py
```

### Constants

Use UPPER_CASE:

```python
MAX_AGE_MINUTES = 240
API_TIMEOUT_SECONDS = 30
DEFAULT_LOG_LEVEL = 'INFO'
```

---

## DOCUMENTATION STANDARDS

### For New Features

Every new feature needs:

1. **Feature Description** - What does it do?
2. **Why It Matters** - Why is it useful?
3. **How to Use** - User guide with examples
4. **How It Works** - Technical explanation
5. **Troubleshooting** - Common issues and fixes

### For Architecture Changes

When changing architecture:

1. **Problem** - What's the issue?
2. **Options** - What alternatives exist?
3. **Decision** - Which option did we choose?
4. **Rationale** - Why did we choose it?
5. **Impacts** - What changes as a result?

**Example (ADR format):**
```markdown
# ADR-008: Use PostgreSQL for Time-Series Data

## Problem
Do we need a specialized time-series database (InfluxDB, TimescaleDB)
or is PostgreSQL sufficient?

## Options
1. PostgreSQL with proper indexing (current)
2. InfluxDB (specialized, complex)
3. TimescaleDB (PostgreSQL extension)

## Decision
Stick with PostgreSQL.

## Rationale
- KISS: PostgreSQL is familiar, well-supported
- Performance: Sufficient for 15-minute granularity
- Complexity: No need for specialized tool

## Impacts
- Data retention limited by disk space
- Queries might be slower than specialized DB
- Easier for team to maintain
```

---

## FEEDBACK & CRITICISM

### How to Give Feedback

Be specific and constructive:

```
❌ Bad: "This code is bad"

✅ Good: "This function is doing too much. The XML parsing
and database insertion should be separate functions
for easier testing."
```

### How to Receive Feedback

Treat feedback as information, not judgment:

```
Someone says: "This is overcomplicated"
Think: "What about this is complex? How can I simplify?"
Not: "They don't understand my approach"
```

---

## QUICK REFERENCE

| Context | Rule |
|---------|------|
| Error messages | [What] [Why] [How to fix] |
| Comments | Explain WHY, not WHAT |
| Commit messages | Use conventional commits |
| Naming | Clear, descriptive, snake_case |
| Documentation | Start big picture, zoom into details |
| Feedback | Be specific and constructive |

---

## RELATED SKILLS

- **SKILL 1**: Hard Rules (rules documented clearly)
- **SKILL 3**: Coding Standards (comments in code)
- **SKILL 2**: Architecture (documented consistently)
