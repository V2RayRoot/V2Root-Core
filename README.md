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
| Build time (Tehran) | `2026-06-14 23:01 +0330` |
| Build status | Tested and released automatically |
<!-- release-metadata:end -->

## Automated Builds

Builds are fully automated through GitHub Actions. Once per day, the release
pipeline checks the latest stable
[XTLS/Xray-core release](https://github.com/XTLS/Xray-core/releases). When a new
version is available, it:

1. Checks out the exact upstream Xray release tag.
2. Adds the V2Root native integration.
3. Builds Linux amd64 and Windows amd64 shared libraries.
4. Exercises every exported API function against the compiled Linux library.
5. Verifies every public symbol in the Windows DLL.
6. Generates SHA-256 checksums.
7. Publishes a tested GitHub Release.
8. Updates this README with the version and Tehran build time.

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
| `xray-windows-amd64.dll` | Windows amd64 shared library |
| `xray-windows-amd64.h` | Windows C header |
| `SHA256SUMS` | Integrity checksums |

## Quality Gates

The release is blocked if compilation, API smoke tests, Windows export
verification, or artifact validation fails. Test implementation and local usage
instructions are available in [`tests/`](tests/).
