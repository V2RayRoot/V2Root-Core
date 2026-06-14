#!/usr/bin/env python3
"""Send, edit, and attach CI reports through the Telegram Bot API."""

from __future__ import annotations

import argparse
import json
import mimetypes
import os
import secrets
import urllib.parse
import urllib.request
from pathlib import Path


def credentials() -> tuple[str, str]:
    token = os.environ.get("TG_TOKEN_BOT", "")
    chat_id = os.environ.get("TG_CHAT_ID", "")
    if not token or not chat_id:
        raise SystemExit("TG_TOKEN_BOT and TG_CHAT_ID must be configured")
    return token, chat_id


def api(method: str, data: dict[str, str]) -> dict:
    token, _ = credentials()
    request = urllib.request.Request(
        f"https://api.telegram.org/bot{token}/{method}",
        data=urllib.parse.urlencode(data).encode(),
    )
    with urllib.request.urlopen(request, timeout=60) as response:
        result = json.load(response)
    if not result.get("ok"):
        raise SystemExit(f"Telegram API error: {result}")
    return result["result"]


def send(text: str) -> int:
    _, chat_id = credentials()
    result = api(
        "sendMessage",
        {
            "chat_id": chat_id,
            "text": text,
            "disable_web_page_preview": "true",
        },
    )
    return int(result["message_id"])


def edit(message_id: str, text: str) -> None:
    _, chat_id = credentials()
    api(
        "editMessageText",
        {
            "chat_id": chat_id,
            "message_id": message_id,
            "text": text,
            "disable_web_page_preview": "true",
        },
    )


def send_document(path: Path, caption: str) -> None:
    token, chat_id = credentials()
    boundary = f"----V2Root{secrets.token_hex(16)}"
    mime_type = mimetypes.guess_type(path.name)[0] or "application/octet-stream"
    body = bytearray()

    def field(name: str, value: str) -> None:
        body.extend(f"--{boundary}\r\n".encode())
        body.extend(
            f'Content-Disposition: form-data; name="{name}"\r\n\r\n'.encode()
        )
        body.extend(value.encode())
        body.extend(b"\r\n")

    field("chat_id", chat_id)
    field("caption", caption)
    body.extend(f"--{boundary}\r\n".encode())
    body.extend(
        (
            f'Content-Disposition: form-data; name="document"; '
            f'filename="{path.name}"\r\n'
        ).encode()
    )
    body.extend(f"Content-Type: {mime_type}\r\n\r\n".encode())
    body.extend(path.read_bytes())
    body.extend(b"\r\n")
    body.extend(f"--{boundary}--\r\n".encode())

    request = urllib.request.Request(
        f"https://api.telegram.org/bot{token}/sendDocument",
        data=bytes(body),
        headers={"Content-Type": f"multipart/form-data; boundary={boundary}"},
    )
    with urllib.request.urlopen(request, timeout=120) as response:
        result = json.load(response)
    if not result.get("ok"):
        raise SystemExit(f"Telegram API error: {result}")


def main() -> None:
    parser = argparse.ArgumentParser()
    subparsers = parser.add_subparsers(dest="command", required=True)

    send_parser = subparsers.add_parser("send")
    send_parser.add_argument("--text", required=True)

    edit_parser = subparsers.add_parser("edit")
    edit_parser.add_argument("--message-id", required=True)
    edit_parser.add_argument("--text", required=True)

    document_parser = subparsers.add_parser("document")
    document_parser.add_argument("--path", type=Path, required=True)
    document_parser.add_argument("--caption", required=True)

    args = parser.parse_args()
    if args.command == "send":
        print(send(args.text))
    elif args.command == "edit":
        edit(args.message_id, args.text)
    else:
        send_document(args.path, args.caption)


if __name__ == "__main__":
    main()
