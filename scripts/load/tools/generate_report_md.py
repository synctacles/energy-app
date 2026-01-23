#!/usr/bin/env python3
import argparse
import json
from datetime import UTC, datetime
from pathlib import Path


def load_json(p):
    with open(p, encoding="utf-8") as f:
        return json.load(f)

def metric(summary, key):
    return summary.get("metrics", {}).get(key, {})

def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--results-dir", default="results")
    ap.add_argument("--out", default="reports/loadtest_report.md")
    ap.add_argument("--hits-per-user", type=int, default=1000)
    args = ap.parse_args()

    results_dir = Path(args.results_dir)
    out = Path(args.out)
    out.parent.mkdir(parents=True, exist_ok=True)

    files = sorted(results_dir.glob("*.json"))
    if not files:
        raise SystemExit("No k6 result JSON files found.")

    now = datetime.now(UTC).strftime("%Y-%m-%d %H:%M UTC")

    lines = []
    lines.append("# SYNCTACLES Load Test Report\n")
    lines.append(f"Generated: **{now}**\n")
    lines.append("## Context\n")
    lines.append(
        "- Server: Hetzner CX33 (4 vCPU / 8 GB RAM)\n"
        "- Target: localhost API (no auth)\n"
        "- Endpoints: `/load`, `/generation-mix`, `/balance`\n"
        f"- Assumption: **{args.hits_per_user} API calls per user per day**\n"
    )

    lines.append("\n## Test Results\n")
    lines.append("| Test | req/s | p95 (ms) | p99 (ms) | error rate | est. users |")
    lines.append("|------|-------|----------|----------|------------|------------|")

    for f in files:
        s = load_json(f)
        name = f.stem

        reqs = metric(s, "http_reqs")
        dur = metric(s, "http_req_duration")
        err = metric(s, "http_req_failed")

        rps = reqs.get("rate", 0.0)
        p95 = dur.get("p(95)", 0.0)
        p99 = dur.get("p(99)", 0.0)
        er = err.get("rate", 0.0)

        hits_per_sec_user = args.hits_per_user / 86400.0
        users = int(rps / hits_per_sec_user) if rps > 0 else 0

        lines.append(
            f"| {name} | {rps:.2f} | {p95:.1f} | {p99:.1f} | {er*100:.2f}% | {users:,} |"
        )

    lines.append(
        "\n## Interpretation\n"
        "- User capacity is a **planning figure**, not a guarantee.\n"
        "- Home Assistant polling (3 endpoints every 60s ≈ 4,320 calls/day/user)\n"
        "  will reduce real capacity by ~4.3× compared to 1,000 calls/day.\n"
        "- Sustained CPU around **80%** with stable p95/p99 is considered acceptable for V1.\n"
    )

    out.write_text("\n".join(lines), encoding="utf-8")
    print(f"Report written to {out}")

if __name__ == "__main__":
    main()
