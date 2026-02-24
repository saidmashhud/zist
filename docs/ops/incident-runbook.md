# Zist Incident Runbook

## Alert intake
1. Check active alerts in Grafana/Prometheus (`ZistServiceDown`, `ZistProbeLatencyP95High`, `ZistProbeFlapping`).
2. Determine blast radius by service label (`zist-gateway`, `zist-listings`, `zist-bookings`, `zist-payments`, `zist-web`).
3. Pull latest gate artifacts from `.artifacts/prod-gate/` and `.artifacts/dr-drill/` for baseline comparison.

## Immediate triage
1. Confirm container health:
   - `docker compose -f /Users/saidmashhud/Projects/personal/zist/docker-compose.yml ps`
2. Check recent logs for failing service:
   - `docker logs --since 10m <container-name>`
3. Validate critical API path:
   - `python3 /Users/saidmashhud/Projects/personal/zist/tests/ops/prod-gate.py` with reduced params (`LOAD_EVENTS=20 SOAK_SECONDS=60`).

## Rollback playbook
1. Roll back service config/image:
   - `docker compose -f /Users/saidmashhud/Projects/personal/zist/docker-compose.yml up -d <service>` with last known-good env/image.
2. Validate health and replay synthetic flow:
   - `python3 /Users/saidmashhud/Projects/personal/zist/tests/ops/prod-gate.py`.
3. Capture rollback metrics:
   - Record `ROLLBACK_RTO_SECONDS` from DR drill output.

## Database restore playbook
1. Run DR drill script (includes backup + restore verification):
   - `python3 /Users/saidmashhud/Projects/personal/zist/tests/ops/dr-drill.py`
2. Confirm post-restore invariants:
   - baseline data exists,
   - canary-after-backup data is absent,
   - services return healthy.
3. Record `RESTORE_RTO_SECONDS` and `RPO_SECONDS` in incident report.

## Incident exit criteria
1. All critical alerts resolved for at least 15 minutes.
2. Synthetic gate passes (`GATE_STATUS=PASS`).
3. Incident report contains root cause, remediation, and follow-up tasks.
