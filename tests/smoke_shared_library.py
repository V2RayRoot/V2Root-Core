#!/usr/bin/env python3
"""Behavioral smoke tests for every exported V2Root C API function."""

from __future__ import annotations

import argparse
import ctypes
import json
import os
import tempfile
from pathlib import Path


EXPORTED_FUNCTIONS = {
    "FreeCString",
    "GetStatus",
    "SetLogOutput",
    "SetLogLevel",
    "Start",
    "Stop",
    "TestLatency",
    "ValidateConfig",
    "GetTotalTraffics",
    "GetRealtimeSpeed",
    "UpdateGeoAssets",
    "GetVersionInfo",
    "Parse",
    "ParseVless",
    "ParseTrojan",
    "ParseVmess",
    "ParseShadowsocks",
    "JSONToConfigString",
}


class V2RootLibrary:
    def __init__(self, path: Path) -> None:
        self.lib = ctypes.CDLL(str(path))
        self.lib.FreeCString.argtypes = [ctypes.c_void_p]
        self.lib.FreeCString.restype = None

        for name in EXPORTED_FUNCTIONS - {
            "FreeCString",
            "SetLogOutput",
            "SetLogLevel",
            "TestLatency",
        }:
            function = getattr(self.lib, name)
            function.restype = ctypes.c_void_p

        self.lib.SetLogOutput.argtypes = [ctypes.c_char_p]
        self.lib.SetLogOutput.restype = None
        self.lib.SetLogLevel.argtypes = [ctypes.c_char_p]
        self.lib.SetLogLevel.restype = None
        self.lib.Start.argtypes = [ctypes.c_char_p, ctypes.c_char_p]
        self.lib.Stop.argtypes = []
        self.lib.TestLatency.argtypes = [
            ctypes.c_char_p,
            ctypes.c_char_p,
            ctypes.c_int,
        ]
        self.lib.TestLatency.restype = ctypes.c_void_p
        self.lib.ValidateConfig.argtypes = [ctypes.c_char_p, ctypes.c_char_p]
        self.lib.GetTotalTraffics.argtypes = []
        self.lib.GetRealtimeSpeed.argtypes = []
        self.lib.UpdateGeoAssets.argtypes = [ctypes.c_char_p]
        self.lib.GetVersionInfo.argtypes = []
        for name in (
            "Parse",
            "ParseVless",
            "ParseTrojan",
            "ParseVmess",
            "ParseShadowsocks",
            "JSONToConfigString",
        ):
            getattr(self.lib, name).argtypes = [ctypes.c_char_p]

    def text(self, function_name: str, *args: object) -> str:
        pointer = getattr(self.lib, function_name)(*args)
        if not pointer:
            return ""
        try:
            return ctypes.string_at(pointer).decode("utf-8")
        finally:
            self.lib.FreeCString(pointer)


def encoded(value: object) -> bytes:
    return json.dumps(value, separators=(",", ":")).encode()


def require_json(value: str, context: str) -> object:
    try:
        return json.loads(value)
    except json.JSONDecodeError as error:
        raise AssertionError(f"{context} returned invalid JSON: {value!r}") from error


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("library", type=Path)
    parser.add_argument("--expected-version")
    parser.add_argument("--expected-release-date")
    parser.add_argument("--expected-code-version", type=int)
    args = parser.parse_args()
    api = V2RootLibrary(args.library.resolve())

    missing = sorted(name for name in EXPORTED_FUNCTIONS if not hasattr(api.lib, name))
    assert not missing, f"Missing exports: {', '.join(missing)}"

    assert api.text("GetStatus") == "STOPPED"
    api.lib.SetLogLevel(b"warning")
    with tempfile.TemporaryDirectory() as temporary_directory:
        log_path = Path(temporary_directory) / "v2root.log"
        api.lib.SetLogOutput(os.fsencode(log_path))

        version = require_json(api.text("GetVersionInfo"), "GetVersionInfo")
        assert {"codeVersion", "version", "releaseDate"} <= set(version)
        if args.expected_version is not None:
            assert version["version"] == args.expected_version, version
        if args.expected_release_date is not None:
            assert version["releaseDate"] == args.expected_release_date, version
        if args.expected_code_version is not None:
            assert version["codeVersion"] == args.expected_code_version, version

        invalid = require_json(
            api.text("ValidateConfig", b"", b"{}"), "ValidateConfig"
        )
        assert invalid.get("error") == "empty config"

        config = {
            "log": {"loglevel": "warning"},
            "inbounds": [],
            "outbounds": [{"protocol": "freedom", "tag": "direct"}],
        }
        validation = require_json(
            api.text("ValidateConfig", encoded(config), b"{}"), "ValidateConfig"
        )
        assert validation.get("result") == "valid", validation

        assert api.text("Start", encoded(config), b"{}") == ""
        assert api.text("GetStatus").startswith("RUNNING")
        require_json(api.text("GetTotalTraffics"), "GetTotalTraffics")
        require_json(api.text("GetRealtimeSpeed"), "GetRealtimeSpeed")
        assert api.text("Start", encoded(config), b"{}") == "server already running"
        assert api.text("Stop") == ""
        assert api.text("GetStatus") == "STOPPED"
        assert api.text("Stop") == "server not running"

        latency = require_json(
            api.text("TestLatency", b"not-json", b"http://127.0.0.1:1", 1),
            "TestLatency",
        )
        assert isinstance(latency, list) and len(latency) == 1

        invalid_asset_target = Path(temporary_directory) / "not-a-directory"
        invalid_asset_target.write_text("occupied", encoding="utf-8")
        asset_result = require_json(
            api.text("UpdateGeoAssets", os.fsencode(invalid_asset_target)),
            "UpdateGeoAssets",
        )
        assert "error" in asset_result

    parser_cases = {
        "Parse": {"uri": "invalid://configuration"},
        "ParseVless": {"uri": "invalid://configuration"},
        "ParseTrojan": {"uri": "invalid://configuration"},
        "ParseVmess": {"uri": "invalid://configuration"},
        "ParseShadowsocks": {"uri": "invalid://configuration"},
    }
    for function_name, options in parser_cases.items():
        assert api.text(function_name, encoded(options)) == ""

    proxy_config = require_json(
        api.text(
            "ParseVless",
            encoded(
                {
                    "uri": (
                        "vless://11111111-1111-1111-1111-111111111111"
                        "@example.com:443?encryption=none"
                    ),
                    "vpn_mode": True,
                }
            ),
        ),
        "ParseVless",
    )
    protocols = {
        inbound.get("protocol") for inbound in proxy_config.get("inbounds", [])
    }
    assert {"http", "socks"} <= protocols, proxy_config
    assert "tun" not in protocols, proxy_config

    uri = api.text(
        "JSONToConfigString",
        encoded(
            {
                "outbounds": [
                    {
                        "protocol": "vless",
                        "settings": {
                            "vnext": [
                                {
                                    "address": "example.com",
                                    "port": 443,
                                    "users": [
                                        {
                                            "id": "11111111-1111-1111-1111-111111111111",
                                            "encryption": "none",
                                        }
                                    ],
                                }
                            ]
                        },
                    }
                ]
            }
        ),
    )
    assert uri.startswith("vless://"), uri

    print(f"PASS: exercised all {len(EXPORTED_FUNCTIONS)} exported functions")


if __name__ == "__main__":
    main()
