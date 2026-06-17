#!/usr/bin/env python3

import pathlib
import sys
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer


response_path = pathlib.Path(sys.argv[1])
url_path = pathlib.Path(sys.argv[2])
response = response_path.read_bytes()


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200)
        self.send_header("Content-Type", "application/yaml")
        self.end_headers()
        self.wfile.write(response)

    def log_message(self, format, *args):
        return


server = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
url_path.write_text(f"http://127.0.0.1:{server.server_port}\n", encoding="utf-8")
server.serve_forever()
