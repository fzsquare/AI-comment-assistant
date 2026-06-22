#!/usr/bin/env python3
"""Smoke-test the deployed frontend gateway and management APIs."""
from __future__ import annotations

import argparse
import json
import sys
import urllib.error
import urllib.parse
import urllib.request
from typing import Any


class CheckError(RuntimeError):
    pass


def log(message: str) -> None:
    print(f"[smoke] {message}", flush=True)


def request_json(
    base_url: str,
    method: str,
    path: str,
    *,
    body: dict[str, Any] | None = None,
    token: str | None = None,
    timeout: float,
) -> tuple[int, dict[str, Any]]:
    url = urllib.parse.urljoin(base_url.rstrip("/") + "/", path.lstrip("/"))
    data = None
    headers: dict[str, str] = {"Accept": "application/json"}
    if body is not None:
        data = json.dumps(body).encode("utf-8")
        headers["Content-Type"] = "application/json"
    if token:
        headers["Authorization"] = f"Bearer {token}"

    req = urllib.request.Request(url, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            raw = resp.read().decode("utf-8")
            payload = json.loads(raw) if raw else {}
            return resp.status, payload
    except urllib.error.HTTPError as exc:
        raw = exc.read().decode("utf-8", errors="replace")
        try:
            payload = json.loads(raw) if raw else {}
        except json.JSONDecodeError:
            payload = {"message": raw.strip()}
        return exc.code, payload
    except urllib.error.URLError as exc:
        raise CheckError(f"{method} {path} failed: {exc.reason}") from exc


def request_html(base_url: str, path: str, *, timeout: float) -> None:
    url = urllib.parse.urljoin(base_url.rstrip("/") + "/", path.lstrip("/"))
    req = urllib.request.Request(url, method="GET", headers={"Accept": "text/html"})
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            body = resp.read(2048).decode("utf-8", errors="replace")
            if resp.status != 200 or "<html" not in body.lower():
                raise CheckError(f"GET {path} returned unexpected frontend response")
    except urllib.error.URLError as exc:
        raise CheckError(f"GET {path} failed: {exc.reason}") from exc


def expect_api_ok(
    base_url: str,
    method: str,
    path: str,
    *,
    body: dict[str, Any] | None = None,
    token: str | None = None,
    timeout: float,
) -> dict[str, Any]:
    status, payload = request_json(base_url, method, path, body=body, token=token, timeout=timeout)
    if status != 200 or payload.get("code") != 0:
        message = payload.get("message") or payload
        raise CheckError(f"{method} {path} returned {status}: {message}")
    log(f"ok {method} {path}")
    return payload


def login(base_url: str, role: str, account: str, password: str, *, timeout: float) -> str:
    payload = expect_api_ok(
        base_url,
        "POST",
        f"/api/{role}/auth/login",
        body={"account": account, "password": password},
        timeout=timeout,
    )
    token = payload.get("data", {}).get("token")
    if not token:
        raise CheckError(f"{role} login did not return a token")
    return token


def check_gateway(base_url: str, *, timeout: float) -> None:
    request_html(base_url, "/admin/login", timeout=timeout)
    log("ok GET /admin/login")
    request_html(base_url, "/merchant/login", timeout=timeout)
    log("ok GET /merchant/login")

    status, _ = request_json(base_url, "GET", "/api/admin/stats", timeout=timeout)
    if status != 401:
        raise CheckError(f"GET /api/admin/stats without token returned {status}, want 401")
    log("ok /api proxy requires backend auth")


def check_admin(base_url: str, token: str, *, timeout: float) -> None:
    for path in (
        "/api/admin/stats",
        "/api/admin/merchants",
        "/api/admin/stores",
        "/api/admin/nfc-tags",
        "/api/admin/review-generation-tasks",
    ):
        expect_api_ok(base_url, "GET", path, token=token, timeout=timeout)


def check_merchant(base_url: str, token: str, *, timeout: float) -> None:
    for path in (
        "/api/merchant/store/detail",
        "/api/merchant/store/keywords",
        "/api/merchant/store/images",
        "/api/merchant/store/platform-links",
        "/api/merchant/reviews",
        "/api/merchant/review-generation-tasks",
    ):
        expect_api_ok(base_url, "GET", path, token=token, timeout=timeout)


def main() -> int:
    parser = argparse.ArgumentParser(description="Smoke-test deployed management flows")
    parser.add_argument("--base-url", required=True)
    parser.add_argument("--timeout", type=float, default=10)
    parser.add_argument("--skip-authenticated", action="store_true")
    parser.add_argument("--admin-account", default="admin")
    parser.add_argument("--admin-password", default="123456")
    parser.add_argument("--merchant-account", default="merchant")
    parser.add_argument("--merchant-password", default="123456")
    args = parser.parse_args()

    try:
        check_gateway(args.base_url, timeout=args.timeout)
        if args.skip_authenticated:
            log("authenticated management checks skipped")
            return 0

        admin_token = login(args.base_url, "admin", args.admin_account, args.admin_password, timeout=args.timeout)
        check_admin(args.base_url, admin_token, timeout=args.timeout)

        merchant_token = login(args.base_url, "merchant", args.merchant_account, args.merchant_password, timeout=args.timeout)
        check_merchant(args.base_url, merchant_token, timeout=args.timeout)
    except CheckError as exc:
        print(f"[smoke] ERROR: {exc}", file=sys.stderr)
        return 1

    log("management flows are reachable")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
