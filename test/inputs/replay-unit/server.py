#!/usr/bin/env python3

import json
import sys
from http.server import BaseHTTPRequestHandler, HTTPServer
from pathlib import Path
from urllib.parse import parse_qs, urlparse


port_file = Path(sys.argv[1])
request_log = Path(sys.argv[2])
request_log.write_text("", encoding="utf-8")
slos_by_project = {
    "replay-project-a": ["replay-slo-a"],
    "replay-project-b": ["replay-slo-b"],
    "replay-source-project": ["replay-source-slo"],
}


class ReplayHandler(BaseHTTPRequestHandler):
    def do_GET(self):
        target = urlparse(self.path)

        if target.path == "/api/internal/plan-info":
            self.write_json({"enabledPlaylists": True})
            return
        if target.path == "/api/get/slo":
            query = parse_qs(target.query)
            project = self.headers.get("Project", "")
            names = query.get("name", [])
            if names != slos_by_project.get(project):
                self.write_not_found(target.path)
                return
            self.write_json([self.slo(name) for name in names])
            return
        if target.path == "/api/internal/timemachine/availability":
            self.write_json({"available": True})
            return
        self.write_not_found(target.path)

    def do_POST(self):
        if self.path != "/api/timetravel":
            self.write_not_found(self.path)
            return

        length = int(self.headers["Content-Length"])
        body = json.loads(self.rfile.read(length))
        with request_log.open("a", encoding="utf-8") as stream:
            stream.write(json.dumps(body, separators=(",", ":")) + "\n")
        self.write_json({})

    def write_json(self, value, status=200):
        payload = json.dumps(value).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(payload)))
        self.end_headers()
        self.wfile.write(payload)

    def write_not_found(self, path):
        self.write_json({"errors": [{"title": f"unexpected path: {path}"}]}, status=404)

    @staticmethod
    def slo(name):
        return {
            "apiVersion": "n9/v1alpha",
            "kind": "SLO",
            "metadata": {"name": name},
            "spec": {
                "indicator": {
                    "metricSource": {
                        "name": "replay-source",
                        "project": "replay-data-source-project",
                        "kind": "Agent",
                    }
                }
            },
        }

    def log_message(self, _format, *_args):
        return


server = HTTPServer(("127.0.0.1", 0), ReplayHandler)
port_file.write_text(str(server.server_port), encoding="utf-8")
server.serve_forever()
