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

    styled = b"\x1b[" in output
    if styled != (args.expect == "styled"):
        print(f"expected {args.expect} terminal output, got {bytes(output)!r}", file=sys.stderr)
        return 1
    text = output.decode().replace("\r\n", "\n")
    text = re.sub(r"\x1b\]8;[^\x07]*\x07", "", text)
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
