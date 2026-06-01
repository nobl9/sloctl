#!/usr/bin/env python3

import json
import os
import socketserver
import ssl
import sys
from pathlib import Path


GITHUB_HOST = "api.github.com"
GITHUB_PORT = "443"
RELEASE_PATH = "/repos/nobl9/sloctl/releases/latest"
DEFAULT_RELEASE_BODY_FILE = Path(__file__).with_name("release-bodies") / "feature.md"


class Proxy(socketserver.BaseRequestHandler):
    def handle(self):
        try:
            self.handle_connect()
        except OSError:
            return

    def handle_connect(self):
        request = read_headers(self.request)
        request_line = request.splitlines()[0]
        method, target, _ = request_line.split(" ", 2)
        if method != "CONNECT" or target != f"{GITHUB_HOST}:{GITHUB_PORT}":
            self.request.sendall(b"HTTP/1.1 502 Bad Gateway\r\n\r\n")
            return

        self.request.sendall(b"HTTP/1.1 200 Connection Established\r\n\r\n")
        context = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
        context.load_cert_chain(certfile=self.server.cert_file, keyfile=self.server.key_file)
        tls_socket = context.wrap_socket(self.request, server_side=True)
        try:
            self.handle_github_request(tls_socket)
        finally:
            tls_socket.close()

    def handle_github_request(self, tls_socket):
        request = read_headers(tls_socket)
        lines = request.splitlines()
        method, path, _ = lines[0].split(" ", 2)
        headers = parse_headers(lines[1:])
        log_request(method, path, headers)
        if (
            method != "GET"
            or path != RELEASE_PATH
            or headers.get("host") != GITHUB_HOST
            or headers.get("accept") != "application/vnd.github+json"
            or headers.get("user-agent") != "sloctl"
        ):
            tls_socket.sendall(b"HTTP/1.1 502 Bad Gateway\r\n\r\n")
            return

        status = int(os.environ.get("RELEASE_PROXY_STATUS", "200"))
        raw_body = os.environ.get("RELEASE_PROXY_RAW_RESPONSE")
        if raw_body is None:
            raw_body = json.dumps(
                {
                    "tag_name": os.environ.get("RELEASE_PROXY_TAG", "v1.1.0"),
                    "body": release_body(),
                    "html_url": os.environ.get(
                        "RELEASE_PROXY_HTML_URL",
                        "https://github.com/nobl9/sloctl/releases/tag/v1.1.0",
                    ),
                }
            )
        body = raw_body.encode()
        reason = "OK" if status == 200 else "Error"
        response = (
            f"HTTP/1.1 {status} {reason}\r\n".encode()
            + b"Content-Type: application/json\r\n"
            + f"Content-Length: {len(body)}\r\n".encode()
            + b"Connection: close\r\n"
            + b"\r\n"
            + body
        )
        tls_socket.sendall(response)


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
    body_file = os.environ.get("RELEASE_PROXY_BODY_FILE")
    if body_file:
        return Path(body_file).read_text(encoding="utf-8")
    body = os.environ.get("RELEASE_PROXY_BODY")
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
    log_path = os.environ.get("RELEASE_PROXY_LOG")
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


def main():
    port_file = Path(sys.argv[1])
    cert_file = sys.argv[2]
    key_file = sys.argv[3]
    with ThreadingTCPServer(("127.0.0.1", 0), Proxy) as server:
        server.cert_file = cert_file
        server.key_file = key_file
        port_file.write_text(str(server.server_address[1]), encoding="utf-8")
        server.serve_forever()


if __name__ == "__main__":
    main()
