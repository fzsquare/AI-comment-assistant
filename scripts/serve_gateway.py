#!/usr/bin/env python3
"""Serve the built frontend and proxy /api to the local Go backend."""
from __future__ import annotations

import argparse
import mimetypes
import os
import posixpath
import sys
import urllib.error
import urllib.parse
import urllib.request
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path


HOP_BY_HOP_HEADERS = {
    "connection",
    "keep-alive",
    "proxy-authenticate",
    "proxy-authorization",
    "te",
    "trailer",
    "transfer-encoding",
    "upgrade",
}


class GatewayHandler(BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def __init__(self, *args, dist_dir: Path, backend_url: str, **kwargs) -> None:
        self.dist_dir = dist_dir.resolve()
        self.backend_url = backend_url.rstrip("/")
        super().__init__(*args, **kwargs)

    def do_GET(self) -> None:
        self._dispatch()

    def do_HEAD(self) -> None:
        self._dispatch(send_body=False)

    def do_POST(self) -> None:
        self._dispatch()

    def do_PUT(self) -> None:
        self._dispatch()

    def do_PATCH(self) -> None:
        self._dispatch()

    def do_DELETE(self) -> None:
        self._dispatch()

    def do_OPTIONS(self) -> None:
        self._dispatch()

    def log_message(self, fmt: str, *args) -> None:
        sys.stderr.write("%s - - [%s] %s\n" % (self.client_address[0], self.log_date_time_string(), fmt % args))

    def _dispatch(self, *, send_body: bool = True) -> None:
        path = urllib.parse.urlsplit(self.path).path
        # /api 与 /uploads（商家上传图片的静态资源）都反代到后端，其余走前端 SPA。
        if (
            path == "/api"
            or path.startswith("/api/")
            or path == "/uploads"
            or path.startswith("/uploads/")
        ):
            self._proxy_api(send_body=send_body)
            return
        if self.command not in {"GET", "HEAD"}:
            self.send_error(405, "Method Not Allowed")
            return
        self._serve_frontend(send_body=send_body)

    def _proxy_api(self, *, send_body: bool) -> None:
        target = self.backend_url + self.path
        length = int(self.headers.get("Content-Length") or 0)
        body = self.rfile.read(length) if length else None

        headers: dict[str, str] = {}
        for key, value in self.headers.items():
            lower = key.lower()
            if lower in HOP_BY_HOP_HEADERS or lower in {"host", "origin", "content-length"}:
                continue
            headers[key] = value
        headers["X-Forwarded-Host"] = self.headers.get("Host", "")
        headers["X-Forwarded-Proto"] = "http"

        request = urllib.request.Request(target, data=body, method=self.command, headers=headers)
        try:
            with urllib.request.urlopen(request, timeout=180) as response:
                response_body = b"" if not send_body else response.read()
                self._send_proxy_response(response.status, response.headers.items(), response_body, send_body=send_body)
        except urllib.error.HTTPError as exc:
            response_body = b"" if not send_body else exc.read()
            self._send_proxy_response(exc.code, exc.headers.items(), response_body, send_body=send_body)
        except urllib.error.URLError as exc:
            message = f"backend unavailable: {exc.reason}\n".encode("utf-8")
            self.send_response(502)
            self.send_header("Content-Type", "text/plain; charset=utf-8")
            self.send_header("Content-Length", str(len(message) if send_body else 0))
            self.end_headers()
            if send_body:
                self.wfile.write(message)

    def _send_proxy_response(
        self,
        status: int,
        response_headers,
        body: bytes,
        *,
        send_body: bool,
    ) -> None:
        self.send_response(status)
        has_content_type = False
        for key, value in response_headers:
            lower = key.lower()
            if lower in HOP_BY_HOP_HEADERS or lower == "content-length":
                continue
            if lower == "content-type":
                has_content_type = True
            self.send_header(key, value)
        if not has_content_type:
            self.send_header("Content-Type", "application/octet-stream")
        self.send_header("Content-Length", str(len(body) if send_body else 0))
        self.end_headers()
        if send_body and body:
            self.wfile.write(body)

    def _serve_frontend(self, *, send_body: bool) -> None:
        file_path = self._resolve_static_path()
        try:
            data = file_path.read_bytes() if send_body else b""
        except OSError:
            self.send_error(404, "Not Found")
            return

        content_type = mimetypes.guess_type(str(file_path))[0] or "application/octet-stream"
        self.send_response(200)
        self.send_header("Content-Type", content_type)
        self.send_header("Content-Length", str(len(data)))
        if file_path.name == "index.html":
            self.send_header("Cache-Control", "no-cache")
        self.end_headers()
        if send_body and data:
            self.wfile.write(data)

    def _resolve_static_path(self) -> Path:
        parsed = urllib.parse.urlsplit(self.path)
        raw_path = urllib.parse.unquote(parsed.path)
        normalized = posixpath.normpath(raw_path).lstrip("/")
        candidate = (self.dist_dir / normalized).resolve()

        dist_root = str(self.dist_dir)
        if not str(candidate).startswith(dist_root + os.sep) and candidate != self.dist_dir:
            return self.dist_dir / "index.html"
        if candidate.is_dir():
            candidate = candidate / "index.html"
        if candidate.is_file():
            return candidate
        return self.dist_dir / "index.html"


def main() -> None:
    parser = argparse.ArgumentParser(description="PPK frontend gateway")
    parser.add_argument("--host", default="0.0.0.0")
    parser.add_argument("--port", type=int, default=8989)
    parser.add_argument("--dist", required=True, type=Path)
    parser.add_argument("--backend", required=True)
    args = parser.parse_args()

    if not (args.dist / "index.html").is_file():
        raise SystemExit(f"frontend dist is missing index.html: {args.dist}")

    def handler(*handler_args, **handler_kwargs):
        return GatewayHandler(
            *handler_args,
            dist_dir=args.dist,
            backend_url=args.backend,
            **handler_kwargs,
        )

    server = ThreadingHTTPServer((args.host, args.port), handler)
    print(f"gateway listening on {args.host}:{args.port}, proxying /api to {args.backend}", flush=True)
    server.serve_forever()


if __name__ == "__main__":
    main()
