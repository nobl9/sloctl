#!/usr/bin/env python3

import json
import socketserver
import ssl
import sys
from pathlib import Path


GITHUB_HOST = "api.github.com"
GITHUB_PORT = "443"
RELEASE_PATH = "/repos/nobl9/sloctl/releases/latest"


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
        if method != "GET" or path != RELEASE_PATH or headers.get("host") != GITHUB_HOST:
            tls_socket.sendall(b"HTTP/1.1 502 Bad Gateway\r\n\r\n")
            return

        body = json.dumps(
            {
                "tag_name": "v1.1.0",
                "body": "## Features\n\n- feat: Add notification tests (#321) @octocat\n",
                "html_url": "https://github.com/nobl9/sloctl/releases/tag/v1.1.0",
            }
        ).encode()
        response = (
            b"HTTP/1.1 200 OK\r\n"
            b"Content-Type: application/json\r\n"
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


def parse_headers(lines):
    headers = {}
    for line in lines:
        if not line:
            continue
        name, value = line.split(":", 1)
        headers[name.lower()] = value.strip()
    return headers


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
