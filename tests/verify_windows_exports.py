#!/usr/bin/env python3
"""Verify that the Windows DLL exposes the complete public C API."""

from __future__ import annotations

import argparse
import re
import subprocess
from pathlib import Path

from smoke_shared_library import EXPORTED_FUNCTIONS


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("dll", type=Path)
    args = parser.parse_args()

    output = subprocess.check_output(
        ["x86_64-w64-mingw32-objdump", "-p", str(args.dll)],
        text=True,
    )
    exports = set(re.findall(r"\]\s+([A-Za-z][A-Za-z0-9_]*)$", output, re.MULTILINE))
    missing = sorted(EXPORTED_FUNCTIONS - exports)
    assert not missing, f"Missing Windows DLL exports: {', '.join(missing)}"
    print(f"PASS: verified all {len(EXPORTED_FUNCTIONS)} Windows exports")


if __name__ == "__main__":
    main()
