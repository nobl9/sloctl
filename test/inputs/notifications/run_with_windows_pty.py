#!/usr/bin/env python3

import os
import sys

from winpty.enums import Backend
from winpty.ptyprocess import PtyProcess


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
    chunks = []
    while True:
        try:
            chunks.append(process.read())
        except EOFError:
            break

    output = "".join(chunks).replace("\r\n", "\n").replace("\r", "")
    sys.stdout.buffer.write(output.encode("utf-8"))
    sys.stdout.buffer.flush()
    return process.wait()


if __name__ == "__main__":
    sys.exit(main())
