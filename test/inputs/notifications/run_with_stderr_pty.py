#!/usr/bin/env python3

import errno
import fcntl
import os
import pty
import selectors
import struct
import subprocess
import sys
import termios


def main():
    if len(sys.argv) < 2:
        print("usage: run_with_stderr_pty.py COMMAND [ARG...]", file=sys.stderr)
        return 2

    controller_fd, terminal_fd = pty.openpty()
    join_output = os.environ.get("SLOCTL_TEST_TTY_JOIN_OUTPUT") == "1"
    columns = os.environ.get("SLOCTL_TEST_TTY_COLUMNS")
    if columns:
        fcntl.ioctl(
            terminal_fd,
            termios.TIOCSWINSZ,
            struct.pack("HHHH", 24, int(columns), 0, 0),
        )
    input_text = os.environ.get("SLOCTL_TEST_TTY_INPUT")
    stdin = terminal_fd if join_output else subprocess.DEVNULL
    if input_text is not None:
        attrs = termios.tcgetattr(terminal_fd)
        attrs[3] &= ~termios.ECHO
        termios.tcsetattr(terminal_fd, termios.TCSANOW, attrs)
        stdin = terminal_fd

    process = subprocess.Popen(
        sys.argv[1:],
        stdin=stdin,
        stdout=terminal_fd if join_output else subprocess.PIPE,
        stderr=terminal_fd,
        close_fds=True,
    )
    if input_text is not None:
        os.write(controller_fd, input_text.encode())
    os.close(terminal_fd)

    selector = selectors.DefaultSelector()
    if not join_output:
        selector.register(process.stdout, selectors.EVENT_READ, sys.stdout.buffer)
    selector.register(
        controller_fd,
        selectors.EVENT_READ,
        sys.stdout.buffer if join_output else sys.stderr.buffer,
    )

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

            key.data.write(data.replace(b"\r\n", b"\n").replace(b"\r", b""))
            key.data.flush()

    return process.wait()


if __name__ == "__main__":
    sys.exit(main())
