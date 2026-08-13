#!/usr/bin/env python3

import json
import sys
from http.server import BaseHTTPRequestHandler, HTTPServer
from pathlib import Path
from urllib.parse import parse_qs, urlparse


port_file = Path(sys.argv[1])
request_log = Path(sys.argv[2])
availability_log = Path(sys.argv[3])
control_log = Path(sys.argv[4])
request_log.write_text("", encoding="utf-8")
availability_log.write_text("", encoding="utf-8")
control_log.write_text("", encoding="utf-8")
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
        if target.path == "/api/timetravel/list":
            self.write_json(
                [
                    {
                        "slo": "replay-slo-a",
                        "project": "replay-project-a",
                        "createdAt": "2026-08-06T12:34:56Z",
                        "status": "in progress",
                    }
                ]
            )
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
            query = {
                key: values[0] if len(values) == 1 else values
                for key, values in parse_qs(target.query).items()
            }
            self.write_log(
                availability_log,
                {
                    "project": self.headers.get("Project", ""),
                    "query": query,
                },
            )
            self.write_json({"available": True})
            return
        self.write_not_found(target.path)

    def do_POST(self):
        body = self.read_json_body()
        if self.path == "/api/timetravel":
            self.write_log(request_log, body)
            self.write_json({})
            return
        if self.path == "/api/timetravel/cancel":
            self.write_control_log(body)
            self.write_json({})
            return
        self.write_not_found(self.path)

    def do_DELETE(self):
        if self.path != "/api/timetravel":
            self.write_not_found(self.path)
            return
        self.write_control_log(self.read_json_body())
        self.write_json({})

    def read_json_body(self):
        length = int(self.headers["Content-Length"])
        return json.loads(self.rfile.read(length))

    def write_control_log(self, body):
        self.write_log(
            control_log,
            {
                "method": self.command,
                "path": self.path,
                "project": self.headers.get("Project", ""),
                "body": body,
            },
        )

    @staticmethod
    def write_log(path, value):
        with path.open("a", encoding="utf-8") as stream:
            stream.write(json.dumps(value, separators=(",", ":")) + "\n")

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
