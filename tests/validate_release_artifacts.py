#!/usr/bin/env python3
"""Validate release files, ZIP contents, and all SHA-256 manifests."""

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


def digest_file(path: Path) -> str:
    return digest_bytes(path.read_bytes())


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
        expected_release_files.update((binary_name, header_name, zip_name))

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

    aggregate = parse_manifest((dist / "SHA256SUMS").read_text(encoding="ascii"))
    assert set(aggregate) == expected_release_files
    for filename in expected_release_files:
        assert aggregate[filename] == digest_file(dist / filename)
        individual = parse_manifest(
            (dist / f"{filename}.sha256").read_text(encoding="ascii")
        )
        assert individual == {filename: aggregate[filename]}

    print(
        f"PASS: validated {len(expected_release_files)} release files, "
        "all ZIP contents, and all SHA-256 manifests"
    )


if __name__ == "__main__":
    main()
