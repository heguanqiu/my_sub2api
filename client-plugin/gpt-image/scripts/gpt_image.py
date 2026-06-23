#!/usr/bin/env python3
"""Generate or edit images with jlaudeapi.com gpt-image-2."""

from __future__ import annotations

import argparse
import base64
import json
import mimetypes
import os
from pathlib import Path
import socket
import sys
import time
from typing import Any
from urllib import error, request


DEFAULT_BASE_URL = "https://img-api.jlaudeapi.com/"
DEFAULT_SIZE = "1024x1024"
SIZES = ("1024x1024", "1024x1792", "1792x1024")
DEFAULT_TIMEOUT_SECONDS = 600


class ImageApiError(RuntimeError):
    """Raised when the API returns an unusable response."""


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Generate or edit images with the jlaudeapi.com gpt-image-2 API."
    )
    parser.add_argument(
        "args",
        nargs="+",
        help='Prompt text, or: edit <image-path> <prompt text>',
    )
    parser.add_argument("--size", choices=SIZES, default=DEFAULT_SIZE)
    parser.add_argument("--n", type=positive_int, default=1)
    parser.add_argument(
        "--out-dir",
        default=os.environ.get("JLAUDE_IMAGE_DIR", "./images"),
        help="Directory for saved image files. Defaults to JLAUDE_IMAGE_DIR or ./images.",
    )
    parser.add_argument(
        "--base-url",
        default=os.environ.get("JLAUDE_BASE_URL", DEFAULT_BASE_URL),
        help="API base URL. Defaults to JLAUDE_BASE_URL or https://img-api.jlaudeapi.com/.",
    )
    parser.add_argument(
        "--timeout",
        type=positive_int,
        default=positive_int(os.environ.get("JLAUDE_TIMEOUT_SECONDS", str(DEFAULT_TIMEOUT_SECONDS))),
        help="Request timeout in seconds. Defaults to JLAUDE_TIMEOUT_SECONDS or 600.",
    )
    parser.add_argument(
        "--response-format",
        choices=("url", "b64_json"),
        default=os.environ.get("JLAUDE_RESPONSE_FORMAT", "url"),
        help="Image response format. Defaults to JLAUDE_RESPONSE_FORMAT or url.",
    )
    return parser.parse_args()


def positive_int(value: str) -> int:
    try:
        parsed = int(value)
    except ValueError as exc:
        raise argparse.ArgumentTypeError("--n must be an integer") from exc
    if parsed < 1:
        raise argparse.ArgumentTypeError("--n must be at least 1")
    return parsed


def split_mode(raw_args: list[str]) -> tuple[str, Path | None, str]:
    if len(raw_args) >= 3 and raw_args[0].lower() == "edit":
        image_path = Path(raw_args[1]).expanduser()
        prompt = " ".join(raw_args[2:]).strip()
        if not prompt:
            raise ImageApiError("Edit mode requires a prompt after the image path.")
        if not image_path.is_file():
            raise ImageApiError(f"Edit image not found: {image_path}")
        return "edit", image_path, prompt

    prompt = " ".join(raw_args).strip()
    if not prompt:
        raise ImageApiError("A prompt is required.")
    return "generate", None, prompt


def api_key() -> str:
    key = os.environ.get("JLAUDE_API_KEY", "").strip()
    if not key:
        raise ImageApiError("JLAUDE_API_KEY is not set.")
    return key


def request_json(
    url: str,
    api_key_value: str,
    body: bytes,
    content_type: str,
    timeout_seconds: int,
) -> dict[str, Any]:
    req = request.Request(
        url,
        data=body,
        method="POST",
        headers={
            "Authorization": f"Bearer {api_key_value}",
            "Content-Type": content_type,
        },
    )
    try:
        with request.urlopen(req, timeout=timeout_seconds) as response:
            payload = response.read()
    except error.HTTPError as exc:
        payload = exc.read()
        raise ImageApiError(format_api_error(payload, exc.code)) from exc
    except error.URLError as exc:
        raise ImageApiError(f"Request failed: {exc.reason}") from exc
    except (TimeoutError, socket.timeout) as exc:
        raise ImageApiError(
            f"Request timed out after {timeout_seconds} seconds."
        ) from exc

    try:
        data = json.loads(payload.decode("utf-8"))
    except json.JSONDecodeError as exc:
        snippet = payload[:500].decode("utf-8", errors="replace")
        raise ImageApiError(f"API returned non-JSON response: {snippet}") from exc
    return data


def format_api_error(payload: bytes, status_code: int) -> str:
    try:
        data = json.loads(payload.decode("utf-8"))
    except json.JSONDecodeError:
        snippet = payload[:500].decode("utf-8", errors="replace")
        return f"API HTTP {status_code}: {snippet}"

    if isinstance(data, dict) and "error" in data:
        err = data["error"]
        if isinstance(err, dict):
            return f"API HTTP {status_code}: {err.get('message', err)}"
        return f"API HTTP {status_code}: {err}"
    return f"API HTTP {status_code}: {data}"


def generate(
    base_url: str,
    api_key_value: str,
    prompt: str,
    size: str,
    count: int,
    timeout_seconds: int,
    response_format: str,
) -> dict[str, Any]:
    body = json.dumps(
        {
            "model": "gpt-image-2",
            "prompt": prompt,
            "size": size,
            "n": count,
            "response_format": response_format,
        }
    ).encode("utf-8")
    return request_json(
        f"{base_url.rstrip('/')}/v1/images/generations",
        api_key_value,
        body,
        "application/json",
        timeout_seconds,
    )


def edit(
    base_url: str,
    api_key_value: str,
    image_path: Path,
    prompt: str,
    size: str,
    timeout_seconds: int,
    response_format: str,
) -> dict[str, Any]:
    fields: dict[str, str] = {
        "model": "gpt-image-2",
        "prompt": prompt,
        "size": size,
        "response_format": response_format,
    }
    files = {"image": image_path}
    body, content_type = multipart_body(fields, files)
    return request_json(
        f"{base_url.rstrip('/')}/v1/images/edits",
        api_key_value,
        body,
        content_type,
        timeout_seconds,
    )


def multipart_body(fields: dict[str, str], files: dict[str, Path]) -> tuple[bytes, str]:
    boundary = f"----gpt-image-{int(time.time() * 1000)}"
    chunks: list[bytes] = []

    for name, value in fields.items():
        chunks.extend(
            [
                f"--{boundary}\r\n".encode("utf-8"),
                f'Content-Disposition: form-data; name="{name}"\r\n\r\n'.encode("utf-8"),
                value.encode("utf-8"),
                b"\r\n",
            ]
        )

    for name, path in files.items():
        filename = path.name
        mime_type = mimetypes.guess_type(filename)[0] or "application/octet-stream"
        chunks.extend(
            [
                f"--{boundary}\r\n".encode("utf-8"),
                (
                    f'Content-Disposition: form-data; name="{name}"; '
                    f'filename="{filename}"\r\n'
                ).encode("utf-8"),
                f"Content-Type: {mime_type}\r\n\r\n".encode("utf-8"),
                path.read_bytes(),
                b"\r\n",
            ]
        )

    chunks.append(f"--{boundary}--\r\n".encode("utf-8"))
    return b"".join(chunks), f"multipart/form-data; boundary={boundary}"


def save_images(data: dict[str, Any], out_dir: Path, timeout_seconds: int) -> list[str]:
    if "error" in data:
        err = data["error"]
        if isinstance(err, dict):
            raise ImageApiError(str(err.get("message", err)))
        raise ImageApiError(str(err))

    items = data.get("data")
    if not isinstance(items, list) or not items:
        raise ImageApiError("API response did not contain image data.")

    out_dir.mkdir(parents=True, exist_ok=True)
    timestamp = int(time.time())
    saved: list[str] = []

    for index, item in enumerate(items, start=1):
        if not isinstance(item, dict):
            continue
        path = out_dir / f"gpt-image-{timestamp}-{index}.png"

        b64_json = item.get("b64_json")
        if isinstance(b64_json, str) and b64_json:
            path.write_bytes(base64.b64decode(b64_json))
            saved.append(str(path.resolve()))
            continue

        url = item.get("url")
        if isinstance(url, str) and url:
            downloaded = download_url(url, path, timeout_seconds)
            saved.append(str(downloaded.resolve()) if downloaded else url)

    if not saved:
        raise ImageApiError("No supported image payloads were found in the API response.")
    return saved


def download_url(url: str, path: Path, timeout_seconds: int) -> Path | None:
    try:
        with request.urlopen(url, timeout=timeout_seconds) as response:
            path.write_bytes(response.read())
        return path
    except Exception:
        return None


def main() -> int:
    try:
        parsed = parse_args()
        mode, image_path, prompt = split_mode(parsed.args)
        key = api_key()
        out_dir = Path(parsed.out_dir).expanduser()
        out_dir.mkdir(parents=True, exist_ok=True)

        if mode == "edit":
            assert image_path is not None
            data = edit(
                parsed.base_url,
                key,
                image_path,
                prompt,
                parsed.size,
                parsed.timeout,
                parsed.response_format,
            )
        else:
            data = generate(
                parsed.base_url,
                key,
                prompt,
                parsed.size,
                parsed.n,
                parsed.timeout,
                parsed.response_format,
            )

        saved = save_images(data, out_dir, parsed.timeout)
        print(f"Generated: {len(saved)}")
        for path in saved:
            print(f"Saved: {path}")
        return 0
    except ImageApiError as exc:
        print(f"Error: {exc}", file=sys.stderr)
        return 1
    except KeyboardInterrupt:
        print("Error: interrupted", file=sys.stderr)
        return 130


if __name__ == "__main__":
    raise SystemExit(main())
