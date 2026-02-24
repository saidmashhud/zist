#!/usr/bin/env python3
"""Zist production gate: load + soak + chaos with tenant isolation checks.

Machine-readable output is printed as KEY=VALUE lines and a JSON artifact is
written to .artifacts/prod-gate/.
"""

from __future__ import annotations

import concurrent.futures
import datetime as dt
import hashlib
import hmac
import json
import math
import os
import pathlib
import statistics
import subprocess
import sys
import time
import urllib.error
import urllib.request
from dataclasses import dataclass
from typing import Dict, List, Optional, Tuple


def env(name: str, default: str) -> str:
    value = os.getenv(name)
    return value if value not in (None, "") else default


ROOT_DIR = pathlib.Path(__file__).resolve().parents[2]
ARTIFACTS_DIR = pathlib.Path(env("ARTIFACTS_DIR", str(ROOT_DIR / ".artifacts" / "prod-gate")))
ARTIFACTS_DIR.mkdir(parents=True, exist_ok=True)

GATEWAY_URL = env("GATEWAY_URL", "http://localhost:8000")
LISTINGS_URL = env("LISTINGS_URL", "http://localhost:8001")
BOOKINGS_URL = env("BOOKINGS_URL", "http://localhost:8002")
PAYMENTS_URL = env("PAYMENTS_URL", "http://localhost:8003")
COMPOSE_FILE = env("COMPOSE_FILE", str(ROOT_DIR / "docker-compose.yml"))

LOAD_EVENTS = int(env("LOAD_EVENTS", "500"))
LOAD_CONCURRENCY = int(env("LOAD_CONCURRENCY", "20"))
SOAK_SECONDS = int(env("SOAK_SECONDS", "300"))
SOAK_INTERVAL_SECONDS = float(env("SOAK_INTERVAL_SECONDS", "1"))
CHAOS_RESTART = env("CHAOS_RESTART", "true").lower() == "true"
CHAOS_AT_SEC = int(env("CHAOS_AT_SEC", "60"))
CHAOS_SERVICES = env("CHAOS_SERVICES", "bookings payments")

TENANT_A = env("TENANT_A", "prodgate-tenant-a")
TENANT_B = env("TENANT_B", "prodgate-tenant-b")
INTERNAL_TOKEN = env("INTERNAL_TOKEN", "dev-internal-token")
WEBHOOK_SECRET = env("MASHGATE_WEBHOOK_SECRET", "")

HOST_SCOPES = (
    "zist.listings.read zist.listings.manage "
    "zist.bookings.read zist.bookings.manage "
    "zist.payments.create zist.webhooks.manage"
)
GUEST_SCOPES = "zist.listings.read zist.bookings.read zist.bookings.manage zist.payments.create"


@dataclass
class FlowResult:
    ok: bool
    tenant: str
    latency_seconds: float
    booking_id: str
    error: str = ""
    isolation_ok: bool = True


@dataclass
class TenantActors:
    tenant_id: str
    host_user: str
    host_email: str
    guest_user: str
    guest_email: str
    listing_id: str = ""


def log(msg: str) -> None:
    print(f"[zist-gate] {msg}", flush=True)


def ensure_binaries() -> None:
    for binary in ("docker",):
        if subprocess.call(["bash", "-lc", f"command -v {binary} >/dev/null 2>&1"]) != 0:
            raise RuntimeError(f"missing required binary: {binary}")


def now_ms() -> int:
    return int(time.time() * 1000)


def request_json(
    method: str,
    url: str,
    payload: Optional[dict] = None,
    headers: Optional[Dict[str, str]] = None,
    timeout: float = 10,
) -> Tuple[int, bytes]:
    body: Optional[bytes] = None
    req_headers = dict(headers or {})
    if payload is not None:
        body = json.dumps(payload, separators=(",", ":")).encode("utf-8")
        req_headers.setdefault("Content-Type", "application/json")
    req = urllib.request.Request(url=url, data=body, method=method)
    for key, value in req_headers.items():
        req.add_header(key, value)

    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            return resp.getcode(), resp.read()
    except urllib.error.HTTPError as http_err:
        return http_err.code, http_err.read()
    except Exception as err:  # noqa: BLE001
        return 0, str(err).encode("utf-8")


def parse_json(data: bytes) -> dict:
    if not data:
        return {}
    try:
        return json.loads(data.decode("utf-8"))
    except Exception:  # noqa: BLE001
        return {}


def auth_headers(user_id: str, tenant_id: str, email: str, scopes: str) -> Dict[str, str]:
    return {
        "X-User-ID": user_id,
        "X-Tenant-ID": tenant_id,
        "X-User-Email": email,
        "X-User-Scopes": scopes,
    }


def webhook_headers(payload_bytes: bytes) -> Dict[str, str]:
    if not WEBHOOK_SECRET.strip():
        return {}
    ts = str(now_ms())
    mac = hmac.new(WEBHOOK_SECRET.encode("utf-8"), digestmod=hashlib.sha256)
    mac.update((ts + "." + payload_bytes.decode("utf-8")).encode("utf-8"))
    return {
        "x-gp-timestamp": ts,
        "x-gp-signature": "v1=" + mac.hexdigest(),
    }


def wait_http_health(url: str, timeout_sec: int = 180) -> bool:
    deadline = time.time() + timeout_sec
    while time.time() < deadline:
        code, _ = request_json("GET", url, timeout=3)
        if code == 200:
            return True
        time.sleep(2)
    return False


def wait_stack_health() -> bool:
    targets = [
        f"{GATEWAY_URL}/healthz",
        f"{LISTINGS_URL}/healthz",
        f"{BOOKINGS_URL}/healthz",
        f"{PAYMENTS_URL}/healthz",
    ]
    return all(wait_http_health(t, timeout_sec=180) for t in targets)


def wait_service_healthy(service_name: str, timeout_sec: int = 180) -> bool:
    cid = f"zist-{service_name}"
    deadline = time.time() + timeout_sec
    while time.time() < deadline:
        inspect_cmd = [
            "docker",
            "inspect",
            "-f",
            "{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}",
            cid,
        ]
        try:
            status = subprocess.check_output(inspect_cmd, text=True).strip()
        except subprocess.CalledProcessError:
            status = ""
        if status in ("healthy", "running"):
            return True
        time.sleep(2)
    return False


def restart_services(services: List[str]) -> bool:
    containers = [f"zist-{s}" for s in services]
    cmd = ["docker", "restart", *containers]
    try:
        subprocess.check_call(cmd, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    except subprocess.CalledProcessError:
        return False
    for service in services:
        if not wait_service_healthy(service):
            return False
    return wait_stack_health()


def create_active_listing(actors: TenantActors, title_suffix: str) -> str:
    payload = {
        "title": f"ProdGate {title_suffix}",
        "description": "production gate listing",
        "city": "Tashkent",
        "country": "UZ",
        "pricePerNight": "150000.00",
        "currency": "UZS",
        "maxGuests": 4,
        "instantBook": True,
    }
    headers = auth_headers(actors.host_user, actors.tenant_id, actors.host_email, HOST_SCOPES)
    code, body = request_json("POST", f"{LISTINGS_URL}/listings", payload=payload, headers=headers)
    if code != 201:
        raise RuntimeError(f"create listing failed for {actors.tenant_id}: code={code} body={body.decode('utf-8', 'ignore')}")
    listing_id = parse_json(body).get("id", "")
    if not listing_id:
        raise RuntimeError(f"create listing missing id for {actors.tenant_id}")

    photo_payload = {"url": "https://example.com/prod-gate.jpg", "caption": "cover"}
    pcode, pbody = request_json(
        "POST",
        f"{LISTINGS_URL}/listings/{listing_id}/photos",
        payload=photo_payload,
        headers=headers,
    )
    if pcode != 201:
        raise RuntimeError(
            f"add photo failed listing={listing_id} code={pcode} body={pbody.decode('utf-8', 'ignore')}"
        )

    pub_code, pub_body = request_json("POST", f"{LISTINGS_URL}/listings/{listing_id}/publish", headers=headers)
    if pub_code != 200:
        raise RuntimeError(
            f"publish failed listing={listing_id} code={pub_code} body={pub_body.decode('utf-8', 'ignore')}"
        )
    return listing_id


def booking_dates(index: int) -> Tuple[str, str]:
    start = dt.date(2030, 1, 1) + dt.timedelta(days=index * 3)
    end = start + dt.timedelta(days=2)
    return start.isoformat(), end.isoformat()


def run_payment_capture_flow(index: int, actors: TenantActors, other_tenant_id: str, isolation_probe: bool) -> FlowResult:
    start = time.time()
    guest_headers = auth_headers(actors.guest_user, actors.tenant_id, actors.guest_email, GUEST_SCOPES)

    check_in, check_out = booking_dates(index)
    create_payload = {
        "listingId": actors.listing_id,
        "checkIn": check_in,
        "checkOut": check_out,
        "guests": 1,
        "message": f"prod gate {index}",
    }
    code, body = request_json("POST", f"{BOOKINGS_URL}/bookings", payload=create_payload, headers=guest_headers)
    if code != 201:
        return FlowResult(False, actors.tenant_id, time.time() - start, "", f"create booking failed: {code}")

    booking_id = parse_json(body).get("id", "")
    if not booking_id:
        return FlowResult(False, actors.tenant_id, time.time() - start, "", "create booking missing id")

    event_payload = {
        "event_id": f"prodgate-{actors.tenant_id}-{index}-{now_ms()}",
        "event_type": "payment.captured",
        "aggregate_id": f"pay-{actors.tenant_id}-{index}",
        "tenant_id": actors.tenant_id,
        "data": {"metadata": {"bookingId": booking_id}},
    }
    payload_bytes = json.dumps(event_payload, separators=(",", ":")).encode("utf-8")
    wh_headers = webhook_headers(payload_bytes)
    wh_code, wh_body = request_json(
        "POST",
        f"{PAYMENTS_URL}/webhooks/mashgate",
        payload=event_payload,
        headers=wh_headers,
    )
    if wh_code != 200:
        return FlowResult(
            False,
            actors.tenant_id,
            time.time() - start,
            booking_id,
            f"webhook failed: {wh_code} {wh_body.decode('utf-8', 'ignore')}",
        )

    confirmed = False
    for _ in range(30):
        g_code, g_body = request_json("GET", f"{BOOKINGS_URL}/bookings/{booking_id}", headers=guest_headers)
        if g_code == 200:
            status = parse_json(g_body).get("status", "")
            if status == "confirmed":
                confirmed = True
                break
        time.sleep(0.25)

    if not confirmed:
        return FlowResult(False, actors.tenant_id, time.time() - start, booking_id, "booking not confirmed")

    isolation_ok = True
    if isolation_probe:
        cross_headers = auth_headers(
            "probe-user-cross",
            other_tenant_id,
            "cross@example.com",
            "zist.bookings.read",
        )
        x_code, _ = request_json("GET", f"{BOOKINGS_URL}/bookings/{booking_id}", headers=cross_headers)
        isolation_ok = x_code in (403, 404)

    return FlowResult(True, actors.tenant_id, time.time() - start, booking_id, isolation_ok=isolation_ok)


def p95(values: List[float]) -> float:
    if not values:
        return 0.0
    if len(values) == 1:
        return values[0]
    sorted_values = sorted(values)
    idx = max(0, min(len(sorted_values) - 1, math.ceil(len(sorted_values) * 0.95) - 1))
    return sorted_values[idx]


def summarize(results: List[FlowResult]) -> dict:
    total = len(results)
    ok_results = [r for r in results if r.ok]
    failures = [r for r in results if not r.ok]
    latencies = [r.latency_seconds for r in ok_results]
    by_tenant = {}
    for result in ok_results:
        by_tenant[result.tenant] = by_tenant.get(result.tenant, 0) + 1
    isolation_failures = [r for r in results if not r.isolation_ok]
    return {
        "total": total,
        "ok": len(ok_results),
        "fail": len(failures),
        "success_pct": round((len(ok_results) / total) * 100, 2) if total else 0.0,
        "p95_seconds": round(p95(latencies), 4),
        "avg_seconds": round(statistics.mean(latencies), 4) if latencies else 0.0,
        "by_tenant": by_tenant,
        "failure_samples": [f.error for f in failures[:5]],
        "isolation_failures": len(isolation_failures),
    }


def main() -> int:
    started_at = time.time()
    ensure_binaries()

    actors_a = TenantActors(
        tenant_id=TENANT_A,
        host_user="prodgate-host-a",
        host_email="host-a@zist.local",
        guest_user="prodgate-guest-a",
        guest_email="guest-a@zist.local",
    )
    actors_b = TenantActors(
        tenant_id=TENANT_B,
        host_user="prodgate-host-b",
        host_email="host-b@zist.local",
        guest_user="prodgate-guest-b",
        guest_email="guest-b@zist.local",
    )

    if not wait_stack_health():
        raise RuntimeError("stack is not healthy before gate start")

    log("creating active listings for two tenants")
    actors_a.listing_id = create_active_listing(actors_a, "A")
    actors_b.listing_id = create_active_listing(actors_b, "B")

    log(f"load phase started: events={LOAD_EVENTS} concurrency={LOAD_CONCURRENCY}")
    load_results: List[FlowResult] = []
    with concurrent.futures.ThreadPoolExecutor(max_workers=LOAD_CONCURRENCY) as pool:
        futures = []
        for i in range(LOAD_EVENTS):
            actor = actors_a if i % 2 == 0 else actors_b
            other = TENANT_B if actor.tenant_id == TENANT_A else TENANT_A
            futures.append(pool.submit(run_payment_capture_flow, i + 1, actor, other, False))
        for future in concurrent.futures.as_completed(futures):
            load_results.append(future.result())

    load_summary = summarize(load_results)
    log(
        "load phase complete: "
        f"ok={load_summary['ok']} fail={load_summary['fail']} "
        f"p95={load_summary['p95_seconds']}s"
    )

    log(f"soak phase started: seconds={SOAK_SECONDS} interval={SOAK_INTERVAL_SECONDS}")
    soak_results: List[FlowResult] = []
    chaos_done = False
    chaos_attempted = False
    soak_start = time.time()
    tick = 0
    while True:
        elapsed = int(time.time() - soak_start)
        if elapsed >= SOAK_SECONDS:
            break

        if CHAOS_RESTART and not chaos_attempted and elapsed >= CHAOS_AT_SEC:
            chaos_attempted = True
            services = [s for s in CHAOS_SERVICES.split() if s.strip()]
            if services:
                log(f"chaos restart: {' '.join(services)}")
                chaos_done = restart_services(services)
                log(f"chaos restart done={str(chaos_done).lower()}")

        tick += 1
        actor = actors_a if tick % 2 == 0 else actors_b
        other = TENANT_B if actor.tenant_id == TENANT_A else TENANT_A
        soak_results.append(run_payment_capture_flow(LOAD_EVENTS + tick, actor, other, tick == 1))
        time.sleep(max(0.0, SOAK_INTERVAL_SECONDS))

    soak_summary = summarize(soak_results)
    log(
        "soak phase complete: "
        f"ok={soak_summary['ok']} fail={soak_summary['fail']} "
        f"p95={soak_summary['p95_seconds']}s"
    )

    if CHAOS_RESTART and not chaos_done:
        # If chaos was requested but not executed due short soak, mark as not done.
        chaos_done = False

    gate_ok = (
        load_summary["fail"] == 0
        and soak_summary["fail"] == 0
        and soak_summary["isolation_failures"] == 0
        and (not CHAOS_RESTART or chaos_done)
    )

    artifact = {
        "timestamp_utc": dt.datetime.utcnow().strftime("%Y-%m-%dT%H:%M:%SZ"),
        "config": {
            "gateway_url": GATEWAY_URL,
            "listings_url": LISTINGS_URL,
            "bookings_url": BOOKINGS_URL,
            "payments_url": PAYMENTS_URL,
            "load_events": LOAD_EVENTS,
            "load_concurrency": LOAD_CONCURRENCY,
            "soak_seconds": SOAK_SECONDS,
            "soak_interval_seconds": SOAK_INTERVAL_SECONDS,
            "chaos_restart": CHAOS_RESTART,
            "chaos_at_sec": CHAOS_AT_SEC,
            "chaos_services": CHAOS_SERVICES,
            "tenant_a": TENANT_A,
            "tenant_b": TENANT_B,
        },
        "load": load_summary,
        "soak": soak_summary,
        "chaos_done": chaos_done,
        "duration_seconds": round(time.time() - started_at, 2),
        "gate_status": "PASS" if gate_ok else "FAIL",
    }

    ts = dt.datetime.utcnow().strftime("%Y%m%d-%H%M%S")
    artifact_path = ARTIFACTS_DIR / f"prod-gate-{ts}.json"
    artifact_path.write_text(json.dumps(artifact, indent=2), encoding="utf-8")

    print(f"GATE_STATUS={artifact['gate_status']}")
    print(f"LOAD_TOTAL={load_summary['total']}")
    print(f"LOAD_OK={load_summary['ok']}")
    print(f"LOAD_FAIL={load_summary['fail']}")
    print(f"LOAD_SUCCESS_PCT={load_summary['success_pct']}")
    print(f"LOAD_P95_SECONDS={load_summary['p95_seconds']}")
    print(f"SOAK_TOTAL={soak_summary['total']}")
    print(f"SOAK_OK={soak_summary['ok']}")
    print(f"SOAK_FAIL={soak_summary['fail']}")
    print(f"SOAK_SUCCESS_PCT={soak_summary['success_pct']}")
    print(f"SOAK_P95_SECONDS={soak_summary['p95_seconds']}")
    print(f"SOAK_ISOLATION_FAILURES={soak_summary['isolation_failures']}")
    print(f"TENANT_A_CONFIRMED={load_summary['by_tenant'].get(TENANT_A, 0) + soak_summary['by_tenant'].get(TENANT_A, 0)}")
    print(f"TENANT_B_CONFIRMED={load_summary['by_tenant'].get(TENANT_B, 0) + soak_summary['by_tenant'].get(TENANT_B, 0)}")
    print(f"CHAOS_DONE={str(chaos_done).lower()}")
    print(f"ARTIFACT={artifact_path}")

    return 0 if gate_ok else 1


if __name__ == "__main__":
    try:
        sys.exit(main())
    except Exception as exc:  # noqa: BLE001
        print(f"GATE_STATUS=FAIL")
        print(f"ERROR={exc}")
        sys.exit(1)
