#!/usr/bin/env python3

import os
import socket
import sys
import time

from winpty.enums import Backend
from winpty.ptyprocess import PtyProcess


COMMAND_TIMEOUT_SECONDS = 10
READ_TIMEOUT_SECONDS = 0.1


def main():
    if len(sys.argv) < 2:
        print("usage: run_with_windows_pty.py COMMAND [ARG...]", file=sys.stderr)
        return 2

    process = PtyProcess.spawn(
        sys.argv[1:],
        env=os.environ.copy(),
        dimensions=(24, 80),
        backend=Backend.ConPTY,
    )
    process.fileobj.settimeout(READ_TIMEOUT_SECONDS)
    deadline = time.monotonic() + COMMAND_TIMEOUT_SECONDS
    chunks = []
    timed_out = False
    while True:
        if time.monotonic() >= deadline:
            timed_out = True
            break
        try:
            chunks.append(process.read())
        except socket.timeout:
            continue
        except EOFError:
            break

    while not timed_out and process.isalive():
        remaining = deadline - time.monotonic()
        if remaining <= 0:
            timed_out = True
            break
        time.sleep(min(READ_TIMEOUT_SECONDS, remaining))

    if timed_out:
        process.terminate(force=True)

    output = "".join(chunks).replace("\r\n", "\n").replace("\r", "")
    sys.stdout.buffer.write(output.encode("utf-8"))
    sys.stdout.buffer.flush()
    if timed_out:
        print(
            f"command timed out after {COMMAND_TIMEOUT_SECONDS} seconds",
            file=sys.stderr,
        )
        return 124
    return process.exitstatus


if __name__ == "__main__":
    sys.exit(main())
