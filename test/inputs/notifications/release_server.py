#!/usr/bin/env python3

import json
import os
import socketserver
import sys
from http import HTTPStatus
from pathlib import Path


RELEASE_PATH = "/repos/nobl9/sloctl/releases/latest"
DEFAULT_RELEASE_BODY_FILE = Path(__file__).with_name("release-bodies") / "feature.md"


class ReleaseHandler(socketserver.BaseRequestHandler):
    def handle(self):
        try:
            request = read_headers(self.request)
        except OSError:
            return

        lines = request.splitlines()
        if not lines:
            return

        try:
            method, path, _ = lines[0].split(" ", 2)
        except ValueError:
            self.request.sendall(b"HTTP/1.1 400 Bad Request\r\n\r\n")
            return

        headers = parse_headers(lines[1:])
        log_request(method, path, headers)
        if (
            method != "GET"
            or path != RELEASE_PATH
            or headers.get("accept") != "application/vnd.github+json"
            or headers.get("user-agent") != "sloctl"
        ):
            self.request.sendall(b"HTTP/1.1 502 Bad Gateway\r\n\r\n")
            return

        status = int(os.environ.get("RELEASE_SERVER_STATUS", "200"))
        raw_body = os.environ.get("RELEASE_SERVER_RAW_RESPONSE")
        if raw_body is None:
            raw_body = json.dumps(
                {
                    "tag_name": os.environ.get("RELEASE_SERVER_TAG", "v1.1.0"),
                    "body": release_body(),
                    "html_url": os.environ.get(
                        "RELEASE_SERVER_HTML_URL",
                        "https://github.com/nobl9/sloctl/releases/tag/v1.1.0",
                    ),
                }
            )
        body = raw_body.encode()
        response = (
            f"HTTP/1.1 {status} {reason_phrase(status)}\r\n".encode()
            + b"Content-Type: application/json\r\n"
            + f"Content-Length: {len(body)}\r\n".encode()
            + b"Connection: close\r\n"
            + b"\r\n"
            + body
        )
        self.request.sendall(response)


class ThreadingTCPServer(socketserver.ThreadingTCPServer):
    allow_reuse_address = True


def read_headers(sock):
    data = b""
    while b"\r\n\r\n" not in data:
        chunk = sock.recv(4096)
        if not chunk:
            raise OSError("connection closed while reading headers")
        data += chunk
    return data.decode("iso-8859-1")


def release_body():
    body_file = os.environ.get("RELEASE_SERVER_BODY_FILE")
    if body_file:
        return Path(body_file).read_text(encoding="utf-8")
    body = os.environ.get("RELEASE_SERVER_BODY")
    if body is not None:
        return body
    return DEFAULT_RELEASE_BODY_FILE.read_text(encoding="utf-8")


def parse_headers(lines):
    headers = {}
    for line in lines:
        if not line:
            continue
        name, value = line.split(":", 1)
        headers[name.lower()] = value.strip()
    return headers


def log_request(method, path, headers):
    log_path = os.environ.get("RELEASE_SERVER_LOG")
    if not log_path:
        return
    with open(log_path, "a", encoding="utf-8") as log_file:
        log_file.write(
            json.dumps(
                {
                    "method": method,
                    "path": path,
                    "accept": headers.get("accept"),
                    "userAgent": headers.get("user-agent"),
                },
                sort_keys=True,
            )
            + "\n"
        )


def reason_phrase(status):
    try:
        return HTTPStatus(status).phrase
    except ValueError:
        return "Status"


def main():
    port_file = Path(sys.argv[1])
    with ThreadingTCPServer(("127.0.0.1", 0), ReleaseHandler) as server:
        port_file.write_text(str(server.server_address[1]), encoding="utf-8")
        server.serve_forever()


if __name__ == "__main__":
    main()
