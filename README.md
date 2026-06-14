# V2Root Core

V2Root Core is the native shared-library integration of Xray-core built
**exclusively for V2Root-App and the V2Root ecosystem**. It provides a stable C
API for configuration parsing, Xray lifecycle management, latency checks,
traffic statistics, logging, and geo-asset management.

> This project is specifically designed, maintained, tested, and released for
> V2Root-App and V2Root. It is not intended to be a general-purpose replacement
> for the official Xray-core distribution.

## Latest Automated Build

<!-- release-metadata:start -->
| Item | Value |
| --- | --- |
| Latest Xray version built | `v26.3.27` |
| V2Root release tag | [`v2root-v26.3.27`](https://github.com/V2RayRoot/V2Root-Core/releases/tag/v2root-v26.3.27) |
| Build time (Tehran) | `2026-06-14 23:12 +0330` |
| Build status | Tested and released automatically |
<!-- release-metadata:end -->

## Automated Builds

Builds are fully automated through GitHub Actions. Once per day, the release
pipeline checks the latest stable
[XTLS/Xray-core release](https://github.com/XTLS/Xray-core/releases). When a new
version is available, it:

1. Checks out the exact upstream Xray release tag.
2. Adds the V2Root native integration.
3. Builds Linux `amd64`, `arm64`, and `386` shared libraries.
4. Builds Windows `amd64` and `386` shared libraries.
5. Exercises every exported API function against the compiled Linux library.
6. Verifies every public symbol in both Windows DLLs.
7. Packages each platform and architecture as a separate ZIP archive.
8. Generates individual and aggregate SHA-256 checksums.
9. Validates every release file and ZIP before publication.
10. Publishes a tested GitHub Release.
11. Updates this README with the version and Tehran build time.

Releases use the `v2root-<xray-tag>` naming convention. Existing versions are
skipped unless a maintainer explicitly requests a force rebuild.

## C API Reference

All returned `char*` values are UTF-8 strings allocated by the library. The
caller owns them and must call `FreeCString` after use. A null pointer returned
by `Start` or `Stop` means success.

| Function | Input arguments | Return value |
| --- | --- | --- |
| `FreeCString` | `char* value` | `void`; releases a string returned by this library |
| `GetStatus` | None | `char*`; `STOPPED`, `STARTING`, `RUNNING`, `RUNNING (VPN)`, `STOPPING`, or `ERROR` |
| `SetLogOutput` | `char* path` | `void`; sets access and error log output paths |
| `SetLogLevel` | `char* level` | `void`; sets the Xray log level |
| `Start` | `char* configInput`, `char* optionsJSON` | `char*`; null on success, otherwise an error message |
| `Stop` | None | `char*`; null on success, otherwise an error message |
| `TestLatency` | `char* configsJSON`, `char* testURL`, `int timeout` | `char*`; JSON array containing latency results |
| `ValidateConfig` | `char* configInput`, `char* optionsJSON` | `char*`; JSON validation result or error |
| `GetTotalTraffics` | None | `char*`; JSON totals for uplink and downlink traffic |
| `GetRealtimeSpeed` | None | `char*`; JSON uplink and downlink speeds |
| `UpdateGeoAssets` | `char* assetPath` | `char*`; JSON result for geo-asset downloads |
| `GetVersionInfo` | None | `char*`; JSON with `codeVersion`, `version`, and `releaseDate` |
| `Parse` | `char* optionsJSON` | `char*`; generated Xray JSON configuration or an empty string |
| `ParseVless` | `char* optionsJSON` | `char*`; generated VLESS Xray JSON or an empty string |
| `ParseTrojan` | `char* optionsJSON` | `char*`; generated Trojan Xray JSON or an empty string |
| `ParseVmess` | `char* optionsJSON` | `char*`; generated VMess Xray JSON or an empty string |
| `ParseShadowsocks` | `char* optionsJSON` | `char*`; generated Shadowsocks Xray JSON or an empty string |
| `JSONToConfigString` | `char* configJSON` | `char*`; share URI derived from the first supported outbound |

## Release Assets

Each automated release contains:

| Asset | Platform |
| --- | --- |
| `xray-linux-amd64.so` | Linux amd64 shared library |
| `xray-linux-amd64.h` | Linux C header |
| `xray-linux-arm64.so` | Linux arm64 shared library |
| `xray-linux-arm64.h` | Linux C header |
| `xray-linux-386.so` | Linux 32-bit x86 shared library |
| `xray-linux-386.h` | Linux C header |
| `xray-windows-amd64.dll` | Windows amd64 shared library |
| `xray-windows-amd64.h` | Windows C header |
| `xray-windows-386.dll` | Windows 32-bit x86 shared library |
| `xray-windows-386.h` | Windows C header |
| `V2Root-Core-<os>-<arch>-<version>.zip` | Platform package containing its binary, header, and internal `SHA256SUMS` |
| `<artifact>.sha256` | Individual checksum for each binary, header, and ZIP |
| `SHA256SUMS` | Aggregate checksum manifest for every binary, header, and ZIP |

## Quality Gates

A release is published only when every quality gate below passes:

| Gate | What it verifies | Failure result |
| --- | --- | --- |
| Compilation | Every supported Linux and Windows target compiles from the exact upstream Xray tag with CGO enabled. | No Release is created if any architecture fails to build. |
| API smoke tests | The compiled Linux amd64 library is loaded dynamically and every exported C function is called, including lifecycle, parser, validation, latency, statistics, logging, asset-update error handling, and memory release. | A crash, invalid response, missing behavior, or failed assertion blocks the Release. |
| Windows export verification | The export tables of both Windows DLLs contain the complete public C API. | A DLL with any missing exported function is rejected. |
| Artifact validation | Every expected binary and header exists and is non-empty; every ZIP contains the correct binary, header, and internal checksum manifest; every individual and aggregate SHA-256 checksum is recalculated and compared. | Missing, empty, incorrectly packaged, or corrupted files block the Release. |

In practical terms, “the release is blocked” means GitHub Actions stops before
the publication job. No partial or untested GitHub Release is uploaded. Test
implementation and usage instructions are available in [`tests/`](tests/).
