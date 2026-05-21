#!/usr/bin/env python3

import errno
import os
import pty
import selectors
import subprocess
import sys


def main():
    if len(sys.argv) < 2:
        print("usage: run_with_stderr_pty.py COMMAND [ARG...]", file=sys.stderr)
        return 2

    master_fd, slave_fd = pty.openpty()
    process = subprocess.Popen(
        sys.argv[1:],
        stdin=subprocess.DEVNULL,
        stdout=subprocess.PIPE,
        stderr=slave_fd,
        close_fds=True,
    )
    os.close(slave_fd)

    selector = selectors.DefaultSelector()
    selector.register(process.stdout, selectors.EVENT_READ, sys.stdout.buffer)
    selector.register(master_fd, selectors.EVENT_READ, sys.stderr.buffer)

    while selector.get_map():
        for key, _ in selector.select():
            fd = key.fileobj if isinstance(key.fileobj, int) else key.fileobj.fileno()
            try:
                data = os.read(fd, 4096)
            except OSError as err:
                if err.errno != errno.EIO:
                    raise
                data = b""

            if not data:
                selector.unregister(key.fileobj)
                if isinstance(key.fileobj, int):
                    os.close(key.fileobj)
                else:
                    key.fileobj.close()
                continue

            key.data.write(data.replace(b"\r\n", b"\n"))
            key.data.flush()

    return process.wait()


if __name__ == "__main__":
    sys.exit(main())
