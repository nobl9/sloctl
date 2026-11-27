#!/usr/bin/env python3

import argparse
import errno
import fcntl
import os
import pty
import re
import selectors
import signal
import subprocess
import sys
import tempfile
import termios
import time
import tty


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--width", type=int, required=True)
    parser.add_argument("--stream", choices=("stdout", "stderr"), default="stdout")
    parser.add_argument("--expect", choices=("styled", "plain"), required=True)
    parser.add_argument("--background", choices=("dark", "light", "unknown", "unresponsive"), default="dark")
    parser.add_argument("--raw", action="store_true")
    parser.add_argument("--write-only", action="store_true")
    parser.add_argument("--redirect-stdin", action="store_true")
    parser.add_argument("--interrupt", choices=("ctrl-c", "sigint", "sigterm"))
    parser.add_argument("command", nargs=argparse.REMAINDER)
    args = parser.parse_args()
    command = args.command[1:] if args.command[:1] == ["--"] else args.command

    controller, terminal = pty.openpty()
    terminal_input = None
    terminal_output = None
    try:
        termios.tcsetwinsize(terminal, (40, args.width))
        original_attributes = termios.tcgetattr(terminal)
        terminal_input = os.open(os.ttyname(terminal), os.O_RDONLY)
        if args.write_only:
            terminal_output = os.open(os.ttyname(terminal), os.O_WRONLY)
        with tempfile.TemporaryFile() as redirected, tempfile.TemporaryFile() as input_file:
            input_file.write(b"input must remain unread\n")
            input_file.seek(0)
            streams = {"stdout": redirected, "stderr": redirected}
            streams[args.stream] = terminal_output if args.write_only else terminal
            stdin = input_file if args.redirect_stdin else terminal_input
            with subprocess.Popen(command, stdin=stdin, preexec_fn=attach_terminal, **streams) as process:
                output = bytearray()
                query = b"\x1b]11;?\x07\x1b[c"
                answered_queries = 0
                deadline = time.monotonic() + 10
                with selectors.DefaultSelector() as selector:
                    selector.register(controller, selectors.EVENT_READ)
                    while True:
                        remaining = deadline - time.monotonic()
                        if remaining <= 0:
                            process.kill()
                            if args.interrupt and answered_queries:
                                raise TimeoutError("help did not stop within 2 seconds of interruption")
                            raise TimeoutError("help command timed out after 10 seconds")
                        if not selector.select(min(remaining, 0.1)):
                            if process.poll() is not None:
                                break
                            continue
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
                            if args.interrupt:
                                if args.interrupt == "ctrl-c":
                                    os.write(controller, b"\x03")
                                else:
                                    process.send_signal(signal.SIGINT if args.interrupt == "sigint" else signal.SIGTERM)
                                deadline = min(deadline, time.monotonic() + 2)
                                answered_queries += 1
                                continue
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
            if args.redirect_stdin and input_file.tell() != 0:
                raise AssertionError("help consumed redirected input")
        if termios.tcgetattr(terminal) != original_attributes:
            raise AssertionError("help did not restore terminal attributes")
        if args.interrupt and answered_queries == 0:
            raise AssertionError("help did not query the background before interruption")
        tty.setraw(terminal, when=termios.TCSANOW)
        os.set_blocking(terminal, False)
        try:
            pending = os.read(terminal, 65536)
        except BlockingIOError:
            pending = b""
        if pending:
            raise AssertionError(f"help left terminal replies unread: {pending!r}")
    finally:
        os.close(controller)
        os.close(terminal)
        if terminal_input is not None:
            os.close(terminal_input)
        if terminal_output is not None:
            os.close(terminal_output)

    if ((args.expect == "plain" and not args.interrupt) or args.redirect_stdin) and query in output:
        print("help queried the background without interactive color input", file=sys.stderr)
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


def attach_terminal():
    os.setsid()
    if os.isatty(0):
        fcntl.ioctl(0, termios.TIOCSCTTY, 0)


if __name__ == "__main__":
    sys.exit(main())
