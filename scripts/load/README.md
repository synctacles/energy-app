# SYNCTACLES Load Tests (k6)

This folder is intended to live at:

`/opt/github/synctacles-repo/scripts/load`

Logs/results are written to:
- JSON results: `./results/`
- Temporary files (optional): `/tmp/synctacles-load/`

## Targets
Localhost API (no auth):
- http://localhost:8000/api/v1/load?country=NL
- http://localhost:8000/api/v1/generation-mix?country=NL
- http://localhost:8000/api/v1/balance?country=NL

## Quick start
1) Install k6 on the server (Ubuntu example):
```bash
sudo apt-get update
sudo apt-get install -y gnupg ca-certificates
sudo mkdir -p /etc/apt/keyrings
curl -fsSL https://dl.k6.io/key.gpg | sudo gpg --dearmor -o /etc/apt/keyrings/k6.gpg
echo "deb [signed-by=/etc/apt/keyrings/k6.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update && sudo apt-get install -y k6
```

2) From this directory run:
```bash
make sanity
make solo
make combined
make soak DURATION=45m
```

3) Generate a PDF report (after tests):
```bash
python3 tools/generate_report.py --results-dir results --out docs/loadtest_report.pdf
```

## What each test does
- `k6/solo_*.js`: isolated endpoint baselines.
- `k6/combined.js`: 3 endpoints in parallel per VU loop (realistic mixed load).
- `k6/soak.js`: long-running stability test.

## Pass/Fail defaults (edit in tools/config.json)
Defaults are conservative and meant for V1:
- Error rate <= 0.5%
- p95 latency <= 250ms (per endpoint)
- p99 latency <= 750ms (per endpoint)
- Avg CPU around 80% is acceptable (you stated this), but sustained 95%+ usually indicates saturation.

## Notes
- Run load tests on an otherwise idle server.
- Ensure Postgres/Timescale and the API are in the same state as production (same config, indexes, pool sizes).
