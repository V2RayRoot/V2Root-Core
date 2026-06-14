#!/usr/bin/env python3
"""Package platform builds and generate per-file and aggregate checksums."""

from __future__ import annotations

import argparse
import hashlib
import shutil
import zipfile
from pathlib import Path


TARGETS = (
    ("linux", "amd64", "so"),
    ("linux", "arm64", "so"),
    ("linux", "386", "so"),
    ("windows", "amd64", "dll"),
    ("windows", "386", "dll"),
)


def digest(path: Path) -> str:
    checksum = hashlib.sha256()
    with path.open("rb") as file:
        for chunk in iter(lambda: file.read(1024 * 1024), b""):
            checksum.update(chunk)
    return checksum.hexdigest()


def checksum_line(path: Path) -> str:
    return f"{digest(path)}  {path.name}\n"


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--dist", type=Path, required=True)
    parser.add_argument("--version", required=True)
    args = parser.parse_args()
    dist = args.dist.resolve()

    raw_files: list[Path] = []
    zip_files: list[Path] = []
    for operating_system, architecture, extension in TARGETS:
        binary = dist / f"xray-{operating_system}-{architecture}.{extension}"
        header = dist / f"xray-{operating_system}-{architecture}.h"
        for path in (binary, header):
            if not path.is_file() or path.stat().st_size == 0:
                raise SystemExit(f"Missing or empty build artifact: {path}")
        raw_files.extend((binary, header))

        package_name = (
            f"V2Root-Core-{operating_system}-{architecture}-{args.version}"
        )
        package_dir = dist / package_name
        package_dir.mkdir()
        packaged_binary = package_dir / binary.name
        packaged_header = package_dir / header.name
        shutil.copy2(binary, packaged_binary)
        shutil.copy2(header, packaged_header)
        (package_dir / "SHA256SUMS").write_text(
            checksum_line(packaged_binary) + checksum_line(packaged_header),
            encoding="ascii",
            newline="\n",
        )

        zip_path = dist / f"{package_name}.zip"
        with zipfile.ZipFile(zip_path, "w", zipfile.ZIP_DEFLATED) as archive:
            for path in sorted(package_dir.iterdir()):
                archive.write(path, f"{package_name}/{path.name}")
        shutil.rmtree(package_dir)
        zip_files.append(zip_path)

    release_files = raw_files + zip_files
    for path in release_files:
        (dist / f"{path.name}.sha256").write_text(
            checksum_line(path), encoding="ascii", newline="\n"
        )

    (dist / "SHA256SUMS").write_text(
        "".join(checksum_line(path) for path in sorted(release_files)),
        encoding="ascii",
        newline="\n",
    )


if __name__ == "__main__":
    main()
