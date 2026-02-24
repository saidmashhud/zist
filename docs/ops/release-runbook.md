# Zist Release / Baseline Runbook

## Preconditions
1. Target stack is up and healthy (`/healthz` on all services).
2. Required env set: `MASHGATE_API_KEY`, `SESSION_SECRET`, `INTERNAL_TOKEN`, `MASHGATE_WEBHOOK_SECRET`.
3. Worktree is clean for release tagging.

## Production gate
1. Full gate run:
   - `LOAD_EVENTS=500 SOAK_SECONDS=300 CHAOS_RESTART=true python3 /Users/saidmashhud/Projects/personal/zist/tests/ops/prod-gate.py`
2. Success criteria:
   - `GATE_STATUS=PASS`
   - no tenant isolation failures
   - chaos restart completed

## DR / rollback gate
1. Run drill:
   - `python3 /Users/saidmashhud/Projects/personal/zist/tests/ops/dr-drill.py`
2. Success criteria:
   - `DR_STATUS=PASS`
   - `ROLLBACK_OK=true`
   - `RESTORE_OK=true`
   - `RTO/RPO` within SLO

## Clean baseline + tag
1. Execute baseline script:
   - `bash /Users/saidmashhud/Projects/personal/zist/scripts/release/tag-baseline.sh`
2. Script behavior:
   - blocks if git worktree is dirty,
   - records baseline report under `.baseline/`,
   - creates annotated git tag (`zist/vYYYYMMDD-HHMMSS`).

## Artifacts
1. Gate artifacts: `.artifacts/prod-gate/*.json`
2. DR artifacts: `.artifacts/dr-drill/*.json`
3. Baseline reports: `.baseline/zist-baseline-*.md`
