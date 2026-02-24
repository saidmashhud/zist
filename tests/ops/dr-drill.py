#!/usr/bin/env python3
"""Zist rollback/restore + DR drill with measured RTO/RPO outputs."""

from __future__ import annotations

import datetime as dt
import gzip
import hashlib
import hmac
import json
import os
import pathlib
import subprocess
import sys
import time
import urllib.error
import urllib.request
from typing import Dict, Optional, Tuple


def env(name: str, default: str) -> str:
    value = os.getenv(name)
    return value if value not in (None, "") else default


ROOT_DIR = pathlib.Path(__file__).resolve().parents[2]
ARTIFACTS_DIR = pathlib.Path(env("ARTIFACTS_DIR", str(ROOT_DIR / ".artifacts" / "dr-drill")))
ARTIFACTS_DIR.mkdir(parents=True, exist_ok=True)

LISTINGS_URL = env("LISTINGS_URL", "http://localhost:8001")
BOOKINGS_URL = env("BOOKINGS_URL", "http://localhost:8002")
PAYMENTS_URL = env("PAYMENTS_URL", "http://localhost:8003")
GATEWAY_URL = env("GATEWAY_URL", "http://localhost:8000")

TENANT_ID = env("TENANT_ID", "dr-tenant-a")
GOOD_INTERNAL_TOKEN = env("INTERNAL_TOKEN", "dev-internal-token")
BAD_INTERNAL_TOKEN = env("BAD_INTERNAL_TOKEN", "rollback-drill-bad-token")
WEBHOOK_SECRET = env("MASHGATE_WEBHOOK_SECRET", "")

COMPOSE_FILE = env("COMPOSE_FILE", str(ROOT_DIR / "docker-compose.yml"))
DB_CONTAINER = env("DB_CONTAINER", "zist-db")
DB_NAME = env("DB_NAME", "zist")
DB_USER = env("DB_USER", "dev")
BACKUP_DIR = pathlib.Path(env("BACKUP_DIR", "/tmp/zist-drill-backups"))
BACKUP_DIR.mkdir(parents=True, exist_ok=True)

HOST_SCOPES = (
    "zist.listings.read zist.listings.manage "
    "zist.bookings.read zist.bookings.manage "
    "zist.payments.create zist.webhooks.manage"
)
GUEST_SCOPES = "zist.listings.read zist.bookings.read zist.bookings.manage zist.payments.create"
APP_CONTAINERS = ["zist-gateway", "zist-listings", "zist-bookings", "zist-payments", "zist-web"]


def log(msg: str) -> None:
    print(f"[zist-dr] {msg}", flush=True)


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


def recreate_payments(internal_token: str) -> None:
    compose_env = os.environ.copy()
    compose_env["MASHGATE_API_KEY"] = compose_env.get("MASHGATE_API_KEY") or "dev-local-key"
    compose_env["SESSION_SECRET"] = compose_env.get("SESSION_SECRET") or "dev-session-secret"
    compose_env["MASHGATE_WEBHOOK_SECRET"] = compose_env.get("MASHGATE_WEBHOOK_SECRET") or WEBHOOK_SECRET or "dev-whsec"
    compose_env["INTERNAL_TOKEN"] = internal_token
    subprocess.check_call(
        ["docker", "compose", "-f", COMPOSE_FILE, "up", "-d", "payments"],
        env=compose_env,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    )


def create_active_listing(title: str) -> str:
    headers = auth_headers("dr-host", TENANT_ID, "dr-host@zist.local", HOST_SCOPES)
    payload = {
        "title": title,
        "description": "dr drill listing",
        "city": "Samarkand",
        "country": "UZ",
        "pricePerNight": "170000.00",
        "currency": "UZS",
        "maxGuests": 2,
        "instantBook": True,
    }
    code, body = request_json("POST", f"{LISTINGS_URL}/listings", payload=payload, headers=headers)
    if code != 201:
        raise RuntimeError(f"create listing failed: code={code} body={body.decode('utf-8', 'ignore')}")
    listing_id = parse_json(body).get("id", "")
    if not listing_id:
        raise RuntimeError("create listing returned empty id")

    pcode, pbody = request_json(
        "POST",
        f"{LISTINGS_URL}/listings/{listing_id}/photos",
        payload={"url": "https://example.com/dr.jpg", "caption": "cover"},
        headers=headers,
    )
    if pcode != 201:
        raise RuntimeError(f"add photo failed: code={pcode} body={pbody.decode('utf-8', 'ignore')}")

    pub_code, pub_body = request_json("POST", f"{LISTINGS_URL}/listings/{listing_id}/publish", headers=headers)
    if pub_code != 200:
        raise RuntimeError(f"publish failed: code={pub_code} body={pub_body.decode('utf-8', 'ignore')}")

    return listing_id


def create_booking(listing_id: str, index: int) -> str:
    headers = auth_headers("dr-guest", TENANT_ID, "dr-guest@zist.local", GUEST_SCOPES)
    check_in = (dt.date(2031, 1, 1) + dt.timedelta(days=index * 5)).isoformat()
    check_out = (dt.date(2031, 1, 3) + dt.timedelta(days=index * 5)).isoformat()
    payload = {
        "listingId": listing_id,
        "checkIn": check_in,
        "checkOut": check_out,
        "guests": 1,
    }
    code, body = request_json("POST", f"{BOOKINGS_URL}/bookings", payload=payload, headers=headers)
    if code != 201:
        raise RuntimeError(f"create booking failed: code={code} body={body.decode('utf-8', 'ignore')}")
    booking_id = parse_json(body).get("id", "")
    if not booking_id:
        raise RuntimeError("create booking returned empty id")
    return booking_id


def send_payment_captured(booking_id: str, suffix: str) -> Tuple[int, str]:
    payload = {
        "event_id": f"dr-{suffix}-{now_ms()}",
        "event_type": "payment.captured",
        "aggregate_id": f"pay-{suffix}",
        "tenant_id": TENANT_ID,
        "data": {"metadata": {"bookingId": booking_id}},
    }
    payload_bytes = json.dumps(payload, separators=(",", ":")).encode("utf-8")
    headers = webhook_headers(payload_bytes)
    code, body = request_json("POST", f"{PAYMENTS_URL}/webhooks/mashgate", payload=payload, headers=headers)
    return code, body.decode("utf-8", "ignore")


def booking_status(booking_id: str) -> Tuple[int, str]:
    headers = auth_headers("dr-guest", TENANT_ID, "dr-guest@zist.local", "zist.bookings.read")
    code, body = request_json("GET", f"{BOOKINGS_URL}/bookings/{booking_id}", headers=headers)
    status = parse_json(body).get("status", "")
    return code, status


def wait_booking_status(booking_id: str, expected: str, timeout_sec: int = 30) -> bool:
    deadline = time.time() + timeout_sec
    while time.time() < deadline:
        code, status = booking_status(booking_id)
        if code == 200 and status == expected:
            return True
        time.sleep(0.25)
    return False


def backup_database(backup_file: pathlib.Path) -> int:
    dump = subprocess.check_output(
        ["docker", "exec", DB_CONTAINER, "pg_dump", "-U", DB_USER, "-d", DB_NAME],
        stderr=subprocess.STDOUT,
    )
    with gzip.open(backup_file, "wb") as fh:
        fh.write(dump)
    return len(dump)


def restore_database(backup_file: pathlib.Path) -> None:
    with gzip.open(backup_file, "rb") as fh:
        data = fh.read()

    # Ensure no active app connections during drop/create.
    subprocess.check_call(["docker", "stop", *APP_CONTAINERS], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)

    terminate_sql = (
        "SELECT pg_terminate_backend(pid) "
        "FROM pg_stat_activity "
        "WHERE datname = '" + DB_NAME + "' AND pid <> pg_backend_pid();"
    )
    subprocess.check_call(
        [
            "docker",
            "exec",
            DB_CONTAINER,
            "psql",
            "-U",
            DB_USER,
            "-d",
            "postgres",
            "-v",
            "ON_ERROR_STOP=1",
            "-c",
            terminate_sql,
            "-c",
            f"DROP DATABASE IF EXISTS {DB_NAME};",
            "-c",
            f"CREATE DATABASE {DB_NAME};",
        ],
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    )

    subprocess.run(
        ["docker", "exec", "-i", DB_CONTAINER, "psql", "-U", DB_USER, "-d", DB_NAME],
        input=data,
        check=True,
    )

    subprocess.check_call(["docker", "start", *APP_CONTAINERS], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)


def main() -> int:
    if not wait_stack_health():
        raise RuntimeError("stack is not healthy before DR drill")

    # Enforce known-good token baseline for payments before starting drill.
    recreate_payments(GOOD_INTERNAL_TOKEN)
    if not wait_service_healthy("payments"):
        raise RuntimeError("payments did not become healthy on preflight")

    log("creating baseline listing and booking for rollback drill")
    baseline_listing = create_active_listing("DR Baseline Listing")
    rollback_booking = create_booking(baseline_listing, 1)

    # Rollback drill: bad token rollout -> rollback to known-good token.
    log("rollback drill: rollout BAD internal token for payments")
    recreate_payments(BAD_INTERNAL_TOKEN)
    if not wait_service_healthy("payments"):
        raise RuntimeError("payments did not become healthy with bad token")

    bad_code, bad_body = send_payment_captured(rollback_booking, "bad")
    status_after_bad = wait_booking_status(rollback_booking, "confirmed", timeout_sec=8)

    rollback_start = time.time()
    log("rollback drill: rollback GOOD internal token for payments")
    recreate_payments(GOOD_INTERNAL_TOKEN)
    if not wait_service_healthy("payments"):
        raise RuntimeError("payments did not become healthy after restart")

    good_code, good_body = send_payment_captured(rollback_booking, "good")
    rollback_ok = wait_booking_status(rollback_booking, "confirmed", timeout_sec=45)
    rollback_rto = round(time.time() - rollback_start, 3)

    # Backup + restore drill.
    ts = dt.datetime.utcnow().strftime("%Y%m%d-%H%M%S")
    backup_file = BACKUP_DIR / f"zist_{DB_NAME}_{ts}.sql.gz"

    log("backup drill: creating postgres dump")
    backup_start = time.time()
    backup_size = backup_database(backup_file)
    backup_end = time.time()

    log("creating post-backup canary listing")
    canary_created_ts = time.time()
    canary_listing = create_active_listing("DR Canary Listing")

    log("restore drill: restoring database from backup")
    restore_start = time.time()
    restore_database(backup_file)
    restore_ok = wait_stack_health()
    restore_rto = round(time.time() - restore_start, 3)

    # Verify restore semantics.
    base_code, _ = request_json("GET", f"{LISTINGS_URL}/listings/{baseline_listing}", headers={"X-Tenant-ID": TENANT_ID})
    canary_code, _ = request_json("GET", f"{LISTINGS_URL}/listings/{canary_listing}", headers={"X-Tenant-ID": TENANT_ID})
    baseline_exists_after = base_code == 200
    canary_exists_after = canary_code == 200

    rpo_seconds = max(0, int(canary_created_ts - backup_end))
    backup_duration = round(backup_end - backup_start, 3)

    gate_ok = (
        bad_code == 200
        and not status_after_bad
        and good_code == 200
        and rollback_ok
        and restore_ok
        and baseline_exists_after
        and not canary_exists_after
    )

    artifact = {
        "timestamp_utc": dt.datetime.utcnow().strftime("%Y-%m-%dT%H:%M:%SZ"),
        "rollback": {
            "bad_webhook_code": bad_code,
            "bad_webhook_body": bad_body,
            "confirmed_while_bad_token": status_after_bad,
            "good_webhook_code": good_code,
            "good_webhook_body": good_body,
            "rollback_ok": rollback_ok,
            "rollback_rto_seconds": rollback_rto,
        },
        "dr": {
            "backup_file": str(backup_file),
            "backup_size_bytes": backup_size,
            "backup_duration_seconds": backup_duration,
            "restore_ok": restore_ok,
            "restore_rto_seconds": restore_rto,
            "rpo_seconds": rpo_seconds,
            "baseline_listing_exists_after_restore": baseline_exists_after,
            "canary_listing_exists_after_restore": canary_exists_after,
        },
        "gate_status": "PASS" if gate_ok else "FAIL",
    }

    artifact_path = ARTIFACTS_DIR / f"dr-drill-{ts}.json"
    artifact_path.write_text(json.dumps(artifact, indent=2), encoding="utf-8")

    print(f"DR_STATUS={artifact['gate_status']}")
    print(f"ROLLBACK_BAD_WEBHOOK_CODE={bad_code}")
    print(f"ROLLBACK_CONFIRMED_WHILE_BAD_TOKEN={str(status_after_bad).lower()}")
    print(f"ROLLBACK_GOOD_WEBHOOK_CODE={good_code}")
    print(f"ROLLBACK_OK={str(rollback_ok).lower()}")
    print(f"ROLLBACK_RTO_SECONDS={rollback_rto}")
    print(f"BACKUP_FILE={backup_file}")
    print(f"BACKUP_DURATION_SECONDS={backup_duration}")
    print(f"RESTORE_OK={str(restore_ok).lower()}")
    print(f"RESTORE_RTO_SECONDS={restore_rto}")
    print(f"RPO_SECONDS={rpo_seconds}")
    print(f"BASELINE_LISTING_EXISTS_AFTER_RESTORE={str(baseline_exists_after).lower()}")
    print(f"CANARY_LISTING_EXISTS_AFTER_RESTORE={str(canary_exists_after).lower()}")
    print(f"ARTIFACT={artifact_path}")

    return 0 if gate_ok else 1


if __name__ == "__main__":
    try:
        sys.exit(main())
    except Exception as exc:  # noqa: BLE001
        print("DR_STATUS=FAIL")
        print(f"ERROR={exc}")
        sys.exit(1)
