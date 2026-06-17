#!/usr/bin/env python3

import os
import pathlib
import subprocess
import sys
import threading
import time


timeout = float(sys.argv[1])
delay = float(sys.argv[2])
input_path = pathlib.Path(sys.argv[3])
command = sys.argv[4:]

read_fd, write_fd = os.pipe()


def write_stdin():
    try:
        time.sleep(delay)
        with input_path.open("rb") as stdin_file:
            while chunk := stdin_file.read(8192):
                os.write(write_fd, chunk)
    finally:
        os.close(write_fd)


writer = threading.Thread(target=write_stdin)
writer.start()

try:
    with os.fdopen(read_fd, "rb") as stdin:
        process = subprocess.Popen(
            command,
            stdin=stdin,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
        )

    try:
        stdout, stderr = process.communicate(timeout=timeout)
    except subprocess.TimeoutExpired:
        process.kill()
        stdout, stderr = process.communicate()
        sys.stdout.write(stdout)
        sys.stderr.write(stderr)
        sys.exit(124)

    sys.stdout.write(stdout)
    sys.stderr.write(stderr)
    sys.exit(process.returncode)
finally:
    writer.join(timeout=1)
