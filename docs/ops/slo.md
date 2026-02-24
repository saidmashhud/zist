# Zist SLO / SLI

## Scope
SLO applies to `gateway`, `listings`, `bookings`, `payments` in the standalone Zist deployment.

## SLI definitions
1. Availability SLI (`service_availability`)
   - Source: Prometheus `probe_success{job="zist-blackbox"}`.
   - Formula: successful probes / total probes per service.
2. Latency SLI (`service_probe_p95`)
   - Source: Prometheus `probe_duration_seconds{job="zist-blackbox"}`.
   - Formula: `quantile_over_time(0.95, ...[10m])`.
3. Checkout-to-confirm SLI (`payment_confirm_success`)
   - Source: `tests/ops/prod-gate.py` artifacts.
   - Formula: confirmed booking flows / total synthetic flows.
4. DR SLI (`restore_rto`, `restore_rpo`)
   - Source: `tests/ops/dr-drill.py` artifacts.
   - Formula: measured restore completion time and data-loss window.

## SLO targets
1. `service_availability >= 99.90%` over rolling 30 days per service.
2. `service_probe_p95 <= 1.0s` over rolling 30 days per service.
3. `payment_confirm_success >= 99.5%` during scheduled gate runs.
4. DR objective: `RTO <= 900s`, `RPO <= 300s`.

## Alert thresholds
1. Critical: `probe_success == 0` for 2 minutes.
2. Warning: p95 probe duration above 1s for 10 minutes.
3. Warning: probe flapping more than 4 state changes in 10 minutes.

## Error budget policy
1. Availability monthly budget: `0.10%` per service.
2. If remaining budget < 50%, freeze non-critical releases.
3. If remaining budget < 20%, only reliability/security fixes until budget recovers.
