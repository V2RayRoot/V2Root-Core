#!/usr/bin/env python3
"""Update the generated release metadata block in README.md."""

from __future__ import annotations

import argparse
import re
from pathlib import Path


START = "<!-- release-metadata:start -->"
END = "<!-- release-metadata:end -->"


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--readme", type=Path, default=Path("README.md"))
    parser.add_argument("--xray-version", required=True)
    parser.add_argument("--release-tag", required=True)
    parser.add_argument("--build-time-tehran", required=True)
    args = parser.parse_args()

    readme = args.readme.read_text(encoding="utf-8")
    block = f"""\
{START}
| Item | Value |
| --- | --- |
| Latest Xray version built | `{args.xray_version}` |
| V2Root release tag | [`{args.release_tag}`](https://github.com/V2RayRoot/V2Root-Core/releases/tag/{args.release_tag}) |
| Build time (Tehran) | `{args.build_time_tehran}` |
| Build status | Tested and released automatically |
{END}"""
    pattern = re.compile(re.escape(START) + r".*?" + re.escape(END), re.DOTALL)
    updated, replacements = pattern.subn(block, readme)
    if replacements != 1:
        raise SystemExit("README release metadata markers are missing or duplicated")
    args.readme.write_text(updated, encoding="utf-8", newline="\n")


if __name__ == "__main__":
    main()
