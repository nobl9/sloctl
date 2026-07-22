#!/usr/bin/env python3

import json
import os
import sys
from http import HTTPStatus
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path


RELEASE_PATH = "/repos/nobl9/sloctl/releases/latest"
DEFAULT_RELEASE_BODY_FILE = Path(__file__).with_name("release-bodies") / "feature.md"


class ReleaseHandler(BaseHTTPRequestHandler):
    def do_GET(self):
        log_request(self.command, self.path, self.headers)
        if (
            self.path != RELEASE_PATH
            or self.headers.get("Accept") != "application/vnd.github+json"
            or self.headers.get("User-Agent") != "sloctl"
        ):
            self.send_response(HTTPStatus.BAD_GATEWAY)
            self.end_headers()
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

        self.send_response(status, reason_phrase(status))
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, format, *args):
        pass


def release_body():
    body_file = os.environ.get("RELEASE_SERVER_BODY_FILE")
    if body_file:
        return Path(body_file).read_text(encoding="utf-8")
    return DEFAULT_RELEASE_BODY_FILE.read_text(encoding="utf-8")


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
                    "accept": headers.get("Accept"),
                    "userAgent": headers.get("User-Agent"),
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
    with ThreadingHTTPServer(("127.0.0.1", 0), ReleaseHandler) as server:
        port_file.write_text(str(server.server_address[1]), encoding="utf-8")
        server.serve_forever()


if __name__ == "__main__":
    main()
