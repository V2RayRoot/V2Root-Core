#!/usr/bin/env python3
"""Validate that release staging contains only complete platform ZIP files."""

from __future__ import annotations

import argparse
import hashlib
import re
import sys
import zipfile
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from scripts.package_release import TARGETS


CHECKSUM_PATTERN = re.compile(r"^([0-9a-f]{64})  (.+)$")


def digest_bytes(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def parse_manifest(content: str) -> dict[str, str]:
    entries: dict[str, str] = {}
    for line in content.splitlines():
        match = CHECKSUM_PATTERN.fullmatch(line)
        if not match:
            raise AssertionError(f"Invalid checksum line: {line!r}")
        entries[match.group(2)] = match.group(1)
    return entries


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--dist", type=Path, required=True)
    parser.add_argument("--version", required=True)
    args = parser.parse_args()
    dist = args.dist.resolve()

    expected_release_files: set[str] = set()
    for operating_system, architecture, extension in TARGETS:
        binary_name = f"xray-{operating_system}-{architecture}.{extension}"
        header_name = f"xray-{operating_system}-{architecture}.h"
        package_name = (
            f"V2Root-Core-{operating_system}-{architecture}-{args.version}"
        )
        zip_name = f"{package_name}.zip"
        expected_release_files.add(zip_name)

        with zipfile.ZipFile(dist / zip_name) as archive:
            expected_members = {
                f"{package_name}/{binary_name}",
                f"{package_name}/{header_name}",
                f"{package_name}/SHA256SUMS",
            }
            assert set(archive.namelist()) == expected_members
            internal = parse_manifest(
                archive.read(f"{package_name}/SHA256SUMS").decode("ascii")
            )
            for filename in (binary_name, header_name):
                assert internal[filename] == digest_bytes(
                    archive.read(f"{package_name}/{filename}")
                )

    actual_release_files = {path.name for path in dist.iterdir() if path.is_file()}
    assert actual_release_files == expected_release_files, (
        f"Release staging contains unexpected files: "
        f"{sorted(actual_release_files - expected_release_files)}"
    )

    print(
        f"PASS: validated {len(expected_release_files)} ZIP-only release assets, "
        "their contents, and internal SHA-256 manifests"
    )


if __name__ == "__main__":
    main()
