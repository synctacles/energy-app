# Status Directory

This directory contains live state files for project tracking.

## Files

| File | Owner | Purpose |
|------|-------|---------|
| `STATUS_CC_CURRENT.md` | Claude Code | CC's current status (server state, code changes, git status) |
| `STATUS_CAI_CURRENT.md` | Claude AI | CAI's current status (project phase, planning, architecture) |
| `STATUS_MERGED_CURRENT.md` | Leo | Single Source of Truth (merged from CC + CAI) |
| `NEXT_ACTIONS.md` | Leo | Prioritized backlog of next actions |

## Workflow

1. **Start session:** Read `STATUS_MERGED_CURRENT.md` (SSOT)
2. **During session:** Work on assigned tasks
3. **End session:** Update own status file (`STATUS_CC_CURRENT.md` or `STATUS_CAI_CURRENT.md`)
4. **Leo merges:** Combines into new `STATUS_MERGED_CURRENT.md`

## See Also

- SKILL_00 Section M: Dual Status Model
- SKILL_00 Section N: Handoff Protocol
