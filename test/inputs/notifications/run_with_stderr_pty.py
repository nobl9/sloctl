#!/usr/bin/env python3

import errno
import os
import pty
import selectors
import signal
import subprocess
import sys
import termios
import time
from contextlib import suppress


COMMAND_TIMEOUT_SECONDS = 10


def main():
    if len(sys.argv) < 2:
        print("usage: run_with_stderr_pty.py COMMAND [ARG...]", file=sys.stderr)
        return 2

    controller_fd = None
    terminal_fd = None
    process = None
    process_completed = False
    selector = selectors.DefaultSelector()
    try:
        controller_fd, terminal_fd = pty.openpty()
        input_text = os.environ.get("SLOCTL_TEST_TTY_INPUT")
        wait_for_raw_mode = (
            os.environ.get("SLOCTL_TEST_TTY_INPUT_WHEN_RAW") == "1"
        )
        stdin = terminal_fd
        if input_text is not None:
            attrs = termios.tcgetattr(terminal_fd)
            attrs[3] &= ~termios.ECHO
            termios.tcsetattr(terminal_fd, termios.TCSANOW, attrs)

        process = subprocess.Popen(
            sys.argv[1:],
            stdin=stdin,
            stdout=subprocess.PIPE,
            stderr=terminal_fd,
            close_fds=True,
            start_new_session=True,
        )
        deadline = time.monotonic() + COMMAND_TIMEOUT_SECONDS
        if input_text is not None and not wait_for_raw_mode:
            os.write(controller_fd, input_text.encode())
        if not wait_for_raw_mode:
            os.close(terminal_fd)
            terminal_fd = None

        selector.register(process.stdout, selectors.EVENT_READ, sys.stdout.buffer)
        selector.register(
            controller_fd,
            selectors.EVENT_READ,
            sys.stderr.buffer,
        )
        while selector.get_map():
            remaining = deadline - time.monotonic()
            if remaining <= 0:
                print(
                    f"command timed out after {COMMAND_TIMEOUT_SECONDS} seconds",
                    file=sys.stderr,
                )
                return 124

            timeout = min(0.1, remaining) if terminal_fd is not None else remaining
            for key, _ in selector.select(timeout):
                try:
                    data = os.read(key.fd, 4096)
                except OSError as err:
                    if err.errno != errno.EIO:
                        raise
                    data = b""

                if not data:
                    selector.unregister(key.fileobj)
                    if isinstance(key.fileobj, int):
                        os.close(key.fileobj)
                        controller_fd = None
                    else:
                        key.fileobj.close()
                    continue

                key.data.write(data.replace(b"\r\n", b"\n").replace(b"\r", b""))
                key.data.flush()
                if key.fileobj == controller_fd and terminal_fd is not None:
                    attrs = termios.tcgetattr(terminal_fd)
                    if not attrs[3] & termios.ICANON:
                        os.write(controller_fd, input_text.encode())
                        os.close(terminal_fd)
                        terminal_fd = None

            if terminal_fd is not None and process.poll() is not None:
                os.close(terminal_fd)
                terminal_fd = None

        try:
            status = process.wait(timeout=max(deadline - time.monotonic(), 0))
        except subprocess.TimeoutExpired:
            print(
                f"command timed out after {COMMAND_TIMEOUT_SECONDS} seconds",
                file=sys.stderr,
            )
            return 124
        process_completed = True
        return status
    finally:
        selector.close()
        if process is not None and not process_completed:
            with suppress(ProcessLookupError):
                os.killpg(process.pid, signal.SIGTERM)
            try:
                process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                pass
            with suppress(ProcessLookupError):
                os.killpg(process.pid, signal.SIGKILL)
            with suppress(subprocess.TimeoutExpired):
                process.wait(timeout=5)
        if process is not None and process.stdout is not None:
            process.stdout.close()
        for fd in (controller_fd, terminal_fd):
            if fd is not None:
                with suppress(OSError):
                    os.close(fd)


if __name__ == "__main__":
    sys.exit(main())
