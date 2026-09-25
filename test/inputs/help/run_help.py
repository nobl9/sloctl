#!/usr/bin/env python3

import argparse
import errno
import os
import pty
import re
import selectors
import subprocess
import sys
import tempfile
import termios
import time


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--width", type=int, required=True)
    parser.add_argument("--stream", choices=("stdout", "stderr"), default="stdout")
    parser.add_argument("--expect", choices=("styled", "plain"), required=True)
    parser.add_argument("--background", choices=("dark", "light", "unknown", "unresponsive"), default="dark")
    parser.add_argument("--raw", action="store_true")
    parser.add_argument("command", nargs=argparse.REMAINDER)
    args = parser.parse_args()
    command = args.command[1:] if args.command[:1] == ["--"] else args.command

    controller, terminal = pty.openpty()
    try:
        termios.tcsetwinsize(terminal, (40, args.width))
        with tempfile.TemporaryFile() as redirected:
            streams = {"stdout": redirected, "stderr": redirected}
            streams[args.stream] = terminal
            with subprocess.Popen(command, stdin=subprocess.DEVNULL, **streams) as process:
                os.close(terminal)
                terminal = None
                output = bytearray()
                query = b"\x1b]11;?\x07\x1b[c"
                answered_queries = 0
                deadline = time.monotonic() + 10
                with selectors.DefaultSelector() as selector:
                    selector.register(controller, selectors.EVENT_READ)
                    while True:
                        remaining = deadline - time.monotonic()
                        if remaining <= 0 or not selector.select(remaining):
                            process.kill()
                            raise TimeoutError("help command timed out after 10 seconds")
                        try:
                            data = os.read(controller, 65536)
                        except OSError as error:
                            if error.errno != errno.EIO:
                                process.kill()
                                raise
                            break
                        if not data:
                            break
                        output.extend(data)
                        while answered_queries < output.count(query):
                            reply = b""
                            if args.background == "dark":
                                reply = b"\x1b]11;rgb:2e2e/3434/4040\x07"
                            elif args.background == "light":
                                reply = b"\x1b]11;rgb:ffff/ffff/ffff\x07"
                            if args.background != "unresponsive":
                                os.write(controller, reply + b"\x1b[?1;2c")
                            answered_queries += 1
                try:
                    status = process.wait(timeout=max(deadline - time.monotonic(), 0))
                except subprocess.TimeoutExpired:
                    process.kill()
                    raise
            redirected.seek(0)
            other_output = redirected.read()
    finally:
        os.close(controller)
        if terminal is not None:
            os.close(terminal)

    if args.expect == "plain" and query in output:
        print("plain help queried the terminal background", file=sys.stderr)
        return 1
    styled = re.search(rb"\x1b\[[0-9;]*m", output) is not None
    if styled != (args.expect == "styled"):
        print(f"expected {args.expect} terminal output, got {bytes(output)!r}", file=sys.stderr)
        return 1
    text = output.decode().replace("\r\n", "\n")
    text = text.replace(query.decode(), "")
    text = re.sub(r"\x1b\]8;[^\x07]*\x07", "", text)
    if args.raw:
        normalized = text
    else:
        text = re.sub(r"\x1b\[[0-9;]*m", "", text)
        lines = [line.rstrip() for line in text.splitlines()]
        normalized = "\n".join(lines) + "\n"
    if args.stream == "stdout":
        sys.stdout.write(normalized)
        sys.stderr.buffer.write(other_output)
    else:
        sys.stdout.buffer.write(other_output)
        if output:
            sys.stderr.write(normalized)
    return status


if __name__ == "__main__":
    sys.exit(main())
