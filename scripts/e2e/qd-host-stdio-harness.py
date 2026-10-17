#!/usr/bin/env python3
"""Run quickdesk_host in native-messaging mode and optionally connect it."""

import argparse
import json
import os
import struct
import subprocess
import sys
import threading


def read_exact(stream, size):
    data = bytearray()
    while len(data) < size:
        chunk = stream.read(size - len(data))
        if not chunk:
            return None
        data.extend(chunk)
    return bytes(data)


def read_message(stream):
    header = read_exact(stream, 4)
    if header is None:
        return None
    length = struct.unpack("<I", header)[0]
    payload = read_exact(stream, length)
    if payload is None:
        return None
    return json.loads(payload.decode("utf-8"))


def write_message(stream, message):
    payload = json.dumps(message, separators=(",", ":")).encode("utf-8")
    stream.write(struct.pack("<I", len(payload)))
    stream.write(payload)
    stream.flush()


def copy_stderr(stream):
    for line in iter(stream.readline, b""):
        sys.stderr.buffer.write(line)
        sys.stderr.buffer.flush()


def parse_args():
    parser = argparse.ArgumentParser()
    parser.add_argument("--host", required=True)
    parser.add_argument("--config-dir", required=True)
    parser.add_argument("--log-dir", required=True)
    parser.add_argument("--server", required=True)
    parser.add_argument("--api-key", default="")
    parser.add_argument("--app-version", default="4.2.9.0")
    parser.add_argument("--access-code", default="")
    return parser.parse_args()


def main():
    args = parse_args()
    os.makedirs(args.config_dir, exist_ok=True)
    os.makedirs(args.log_dir, exist_ok=True)

    process = subprocess.Popen(
        [
            args.host,
            f"--config-dir={args.config_dir}",
            f"--log-dir={args.log_dir}",
        ],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        bufsize=0,
    )
    threading.Thread(
        target=copy_stderr,
        args=(process.stderr,),
        daemon=True,
    ).start()

    connect_sent = False
    try:
        write_message(
            process.stdin,
            {"type": "hello", "protocol_version": 3},
        )
        while True:
            message = read_message(process.stdout)
            if message is None:
                break
            print(json.dumps(message, ensure_ascii=False), flush=True)
            message_type = message.get("type")

            if message_type == "helloResponse" and not connect_sent:
                connect = {
                    "type": "connect",
                    "signalingServerUrl": args.server,
                    "appVersion": args.app_version,
                    "useElevatedHost": False,
                }
                if args.api_key:
                    connect["apiKey"] = args.api_key
                if args.access_code:
                    connect["accessCode"] = args.access_code
                write_message(process.stdin, connect)
                connect_sent = True
    except KeyboardInterrupt:
        pass
    finally:
        process.terminate()
        try:
            process.wait(timeout=5)
        except subprocess.TimeoutExpired:
            process.kill()
            process.wait()
    return process.returncode or 0


if __name__ == "__main__":
    raise SystemExit(main())
