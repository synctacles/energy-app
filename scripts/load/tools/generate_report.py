
#!/usr/bin/env python3

import argparse
import json
import os
import math
from datetime import datetime, timezone
from glob import glob

from reportlab.lib.pagesizes import A4
from reportlab.lib.styles import getSampleStyleSheet
from reportlab.platypus import SimpleDocTemplate, Paragraph, Spacer, Table, TableStyle, PageBreak
from reportlab.lib import colors

def load_json(path: str) -> dict:
    with open(path, "r", encoding="utf-8") as f:
        return json.load(f)

def pick_metric(summary: dict, key: str, default=None):
    return summary.get("metrics", {}).get(key, default)

def fmt_ms(v):
    if v is None:
        return "n/a"
    return f"{v:.1f} ms"

def fmt_rate(v):
    if v is None:
        return "n/a"
    return f"{v*100:.2f}%"

def compute_capacity(rps: float, hits_per_user_per_day: int):
    # Users supported at steady-state, if each user generates hits_per_user_per_day calls spread over a day.
    if rps is None or rps <= 0:
        return None
    hits_per_user_per_sec = hits_per_user_per_day / 86400.0
    return rps / hits_per_user_per_sec

def summary_row(name: str, s: dict):
    reqs = pick_metric(s, "http_reqs", {})
    duration = pick_metric(s, "http_req_duration", {})
    failed = pick_metric(s, "http_req_failed", {})
    return [
        name,
        f"{reqs.get('count','n/a')}",
        f"{reqs.get('rate','n/a'):.2f} req/s" if isinstance(reqs.get("rate"), (int, float)) else "n/a",
        fmt_ms(duration.get("p(95)")),
        fmt_ms(duration.get("p(99)")),
        fmt_rate(failed.get("rate")),
    ]

def make_table(data):
    t = Table(data, hAlign="LEFT", colWidths=[120, 80, 80, 70, 70, 70])
    t.setStyle(TableStyle([
        ("BACKGROUND", (0,0), (-1,0), colors.lightgrey),
        ("TEXTCOLOR", (0,0), (-1,0), colors.black),
        ("FONTNAME", (0,0), (-1,0), "Helvetica-Bold"),
        ("GRID", (0,0), (-1,-1), 0.25, colors.grey),
        ("VALIGN", (0,0), (-1,-1), "MIDDLE"),
        ("PADDING", (0,0), (-1,-1), 6),
    ]))
    return t

def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--results-dir", default="results", help="Directory containing k6 --summary-export JSON files")
    ap.add_argument("--config", default=os.path.join("tools","config.json"))
    ap.add_argument("--out", default=os.path.join("docs","loadtest_report.pdf"))
    args = ap.parse_args()

    cfg = load_json(args.config)
    hits_per_user_per_day = int(cfg["assumptions"]["hits_per_user_per_day"])

    files = sorted(glob(os.path.join(args.results_dir, "*.json")))
    if not files:
        raise SystemExit(f"No k6 summary JSON files found in {args.results_dir}")

    styles = getSampleStyleSheet()
    doc = SimpleDocTemplate(args.out, pagesize=A4, title="SYNCTACLES Load Test Report")
    story = []

    now = datetime.now(timezone.utc).strftime("%Y-%m-%d %H:%M UTC")
    story.append(Paragraph("SYNCTACLES Load Test Report", styles["Title"]))
    story.append(Paragraph(f"Generated: {now}", styles["Normal"]))
    story.append(Spacer(1, 12))

    story.append(Paragraph("Context", styles["Heading2"]))
    story.append(Paragraph(
        "This report summarizes k6 load tests executed against the local SYNCTACLES API on a Hetzner CX33 "
        "(4 vCPU, 8 GB RAM, local disk). Endpoints tested: /load, /generation-mix, /balance (country=NL). "
        "No authentication was used. Results are based on k6 summary exports.",
        styles["BodyText"]
    ))
    story.append(Spacer(1, 12))

    story.append(Paragraph("Key assumptions for 'users supported'", styles["Heading2"]))
    story.append(Paragraph(
        f"We translate measured requests/second (req/s) into an estimated number of users by assuming "
        f"each user produces {hits_per_user_per_day} API calls per day (average over 24h). "
        "User capacity = (measured req/s) / (hits_per_user_per_day / 86400). "
        "This is a planning model; real Home Assistant polling can be higher (e.g., multiple endpoints every 60s).",
        styles["BodyText"]
    ))
    story.append(Spacer(1, 12))

    story.append(Paragraph("Test summaries", styles["Heading2"]))
    table_data = [["Test", "Requests", "Avg Rate", "p95", "p99", "Error rate"]]

    capacities = []
    for fp in files:
        s = load_json(fp)
        name = os.path.basename(fp).replace(".json","")
        table_data.append(summary_row(name, s))

        reqs = pick_metric(s, "http_reqs", {})
        rps = reqs.get("rate") if isinstance(reqs.get("rate"), (int, float)) else None
        cap = compute_capacity(rps, hits_per_user_per_day)
        if cap is not None:
            capacities.append((name, rps, cap))

    story.append(make_table(table_data))
    story.append(Spacer(1, 12))

    story.append(Paragraph("Estimated user capacity (planning)", styles["Heading2"]))
    if capacities:
        cap_table = [["Test", "Measured req/s", f"Users @ {hits_per_user_per_day} hits/day/user"]]
        for name, rps, cap in capacities:
            cap_table.append([name, f"{rps:.2f}", f"{math.floor(cap):,}"])
        t = Table(cap_table, hAlign="LEFT", colWidths=[220, 120, 180])
        t.setStyle(TableStyle([
            ("BACKGROUND", (0,0), (-1,0), colors.lightgrey),
            ("FONTNAME", (0,0), (-1,0), "Helvetica-Bold"),
            ("GRID", (0,0), (-1,-1), 0.25, colors.grey),
            ("PADDING", (0,0), (-1,-1), 6),
        ]))
        story.append(t)
    else:
        story.append(Paragraph("No valid req/s metrics found to compute capacity.", styles["BodyText"]))

    story.append(Spacer(1, 12))
    story.append(Paragraph("How to interpret this", styles["Heading2"]))
    story.append(Paragraph(
        "The 'users supported' number is not a guarantee. It is a conversion of throughput into a daily-usage model. "
        "For Home Assistant, a more realistic pattern might be 3 endpoints every 60 seconds "
        "(~4,320 hits/day/user). If you want, update tools/config.json and regenerate this report.",
        styles["BodyText"]
    ))

    story.append(Spacer(1, 12))
    story.append(Paragraph("Next steps (recommended)", styles["Heading2"]))
    story.append(Paragraph(
        "1) Run the combined test until you observe stable p95/p99 and CPU ~80%.\n"
        "2) Run the 45–60 minute soak test and verify memory stays flat and error rate stays <=0.5%.\n"
        "3) If p99 spikes: check DB indexes (EXPLAIN ANALYZE), connection pooling, and uvicorn worker count.",
        styles["BodyText"]
    ))

    os.makedirs(os.path.dirname(args.out), exist_ok=True)
    doc.build(story)

if __name__ == "__main__":
    main()
