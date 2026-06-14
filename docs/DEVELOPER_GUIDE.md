# V2Root Core Developer Integration Guide

This document is the authoritative integration guide for application developers
embedding V2Root Core in desktop, service, command-line, or mobile-adjacent
software.

V2Root Core packages Xray-core as a native C-compatible shared library. The
library is designed specifically for V2Root-App and the V2Root ecosystem, but
its ABI can be consumed by any language capable of calling a native C library.

## Contents

1. [Scope and architecture](#scope-and-architecture)
2. [Supported release packages](#supported-release-packages)
3. [ABI and memory ownership](#abi-and-memory-ownership)
4. [Loading the library](#loading-the-library)
5. [Lifecycle and process model](#lifecycle-and-process-model)
6. [Complete API reference](#complete-api-reference)
7. [Parser input schema](#parser-input-schema)
8. [Configuration examples](#configuration-examples)
9. [Language integration examples](#language-integration-examples)
10. [Concurrency and thread safety](#concurrency-and-thread-safety)
11. [Error handling](#error-handling)
12. [Geo assets](#geo-assets)
13. [Logging and observability](#logging-and-observability)
14. [Security guidance](#security-guidance)
15. [Production integration checklist](#production-integration-checklist)

## Scope and Architecture

V2Root Core exposes three functional layers:

| Layer | Purpose |
| --- | --- |
| Runtime | Start and stop one embedded Xray instance and query its state |
| Configuration | Parse share URIs, validate Xray JSON, and convert JSON back to a share URI |
| Operations | Test latency, query traffic, configure logs, download geo assets, and inspect build metadata |

The library is built with Go `-buildmode=c-shared`. Every release ZIP contains:

- One native shared library (`.dll` or `.so`)
- The generated C header for that exact build
- An internal `SHA256SUMS` file covering the binary and header

The embedded runtime is process-global. Loading the same library multiple times
inside one process does not create independent Xray engines. Applications that
need isolated concurrent engines must use separate operating-system processes.

## Supported Release Packages

| Package | Binary |
| --- | --- |
| `V2Root-Core-windows-amd64-<version>.zip` | `xray-windows-amd64.dll` |
| `V2Root-Core-windows-386-<version>.zip` | `xray-windows-386.dll` |
| `V2Root-Core-linux-amd64-<version>.zip` | `xray-linux-amd64.so` |
| `V2Root-Core-linux-arm64-<version>.zip` | `xray-linux-arm64.so` |
| `V2Root-Core-linux-386-<version>.zip` | `xray-linux-386.so` |

Always ship the binary and header from the same ZIP. Do not combine a header
from one release or architecture with a binary from another.

Verify the package before installation:

```bash
sha256sum -c SHA256SUMS
```

## ABI and Memory Ownership

### Encoding

All input `char*` values are expected to be UTF-8, null-terminated strings. All
returned textual values are UTF-8, null-terminated strings.

### Returned strings

Most functions returning `char*` allocate memory using the library's C
allocator. The caller must release every non-null returned pointer with
`FreeCString`.

```c
char *value = GetStatus();
if (value != NULL) {
    printf("%s\n", value);
    FreeCString(value);
}
```

Do not release returned pointers with:

- C or C++ `free`
- Windows `LocalFree`, `HeapFree`, or `CoTaskMemFree`
- A language runtime allocator
- A garbage collector finalizer that may execute after the library is unloaded

Use only `FreeCString`.

### Null versus empty string

This distinction is important:

| Result | Meaning |
| --- | --- |
| Null pointer from `Start` or `Stop` | Success |
| Non-null pointer from `Start` or `Stop` | Error text; release it with `FreeCString` |
| Non-null pointer containing `""` from parser/conversion APIs | Parsing or conversion failed |

Parser APIs normally return an allocated empty C string on failure, not a null
pointer. That empty result must still be released.

### Integer ABI

`TestLatency` receives a C `int` timeout measured in seconds. Build-generated
headers are the source of truth for exact platform declarations.

## Loading the Library

### Windows

Place the DLL beside the executable or load it by absolute path. The process
architecture must match the DLL:

- 64-bit process: `xray-windows-amd64.dll`
- 32-bit process: `xray-windows-386.dll`

### Linux

Load the `.so` by absolute path or configure the dynamic loader:

```bash
export LD_LIBRARY_PATH="/opt/v2root/lib:${LD_LIBRARY_PATH}"
```

Applications may also install the library under an application-private
directory and use `dlopen` directly.

### Native declarations

Include the generated header:

```c
#include "xray-linux-amd64.h"
```

The exported API is intentionally flat and C-compatible. Do not call internal
Go symbols or depend on symbol names not present in the generated header.

## Lifecycle and Process Model

The runtime follows this logical state machine:

```text
STOPPED -> STARTING -> RUNNING -> STOPPING -> STOPPED
```

Current public status strings are:

- `STOPPED`
- `STARTING`
- `RUNNING`
- `STOPPING`

Only one server instance can run at a time. Calling `Start` while the instance
is running returns `server already running`.

Recommended lifecycle:

1. Load the library.
2. Call `GetVersionInfo`.
3. Configure logging with `SetLogLevel` and optionally `SetLogOutput`.
4. Call `ValidateConfig`.
5. Call `Start`.
6. Poll `GetStatus` if the UI needs state synchronization.
7. Query traffic or speed only while running.
8. Call `Stop` before unloading the library or terminating the host process.
9. Release every returned string.

`Start` launches Xray asynchronously. A successful null return means the Xray
instance was created and startup was scheduled. Applications that require a
readiness check should poll `GetStatus` and verify their local proxy port before
declaring the connection usable.

## Complete API Reference

### `FreeCString`

```c
void FreeCString(char *value);
```

Releases any non-null string returned by V2Root Core. Passing null is safe.

### `GetStatus`

```c
char *GetStatus(void);
```

Returns the current process-global runtime state.

Example result:

```text
RUNNING
```

The result is allocated and must be released.

### `SetLogOutput`

```c
void SetLogOutput(char *path);
```

Sets both Xray access and error log destinations to the supplied path. An empty
path does not clear a previously configured path.

Call this before `Start`. The path must be writable by the host process.

### `SetLogLevel`

```c
void SetLogLevel(char *level);
```

Sets the Xray `loglevel` value applied at startup. Typical values are:

- `debug`
- `info`
- `warning`
- `error`
- `none`

Call this before `Start`.

### `Start`

```c
char *Start(char *configInput, char *optionsJSON);
```

Starts the process-global Xray runtime.

`configInput` accepts one of:

- A complete Xray JSON object
- A filesystem path to a JSON configuration
- A `vless://` share URI
- A `vmess://` share URI
- A `trojan://` share URI
- An `ss://` share URI

`optionsJSON` accepts:

```json
{
  "geositeFile": "/absolute/path/geosite.dat",
  "geositePath": "/absolute/path/geosite.dat",
  "vpn_mode": false
}
```

`geositeFile` is strict: if supplied but missing, startup fails. `geositePath`
is treated as optional.

Return contract:

- Null: startup accepted
- Non-null: human-readable error

Internally, `Start`:

1. Loads or parses the input.
2. Migrates legacy `dnsConfig` to `dns`.
3. Applies configured logging.
4. Creates a loopback-only StatsService port.
5. Adds API inbound, outbound, and routing entries.
6. Enables system traffic statistics.
7. Normalizes key routing and outbound fields.
8. Asks Xray-core to load the finalized JSON.
9. Starts Xray asynchronously.

### `Stop`

```c
char *Stop(void);
```

Stops the active engine and clears runtime statistics state.

Return contract:

- Null: stop completed
- `server not running`: no active instance existed

Calling `Stop` repeatedly is safe if the caller handles the second call as a
non-fatal state result.

### `ValidateConfig`

```c
char *ValidateConfig(char *configInput, char *optionsJSON);
```

Accepts the same configuration forms as `Start`, applies the same major
normalization rules, and asks Xray-core to load the final configuration without
starting a persistent engine.

Success:

```json
{"result":"valid"}
```

VPN-mode success:

```json
{"result":"valid","vpn_mode":true}
```

Failure:

```json
{"error":"config validation failed: ..."}
```

Always parse the result as JSON and test for `error`; do not compare the entire
string because error details may change with Xray-core.

### `TestLatency`

```c
char *TestLatency(char *configsJSON, char *testURL, int timeout);
```

Tests configurations concurrently through temporary local HTTP/SOCKS proxy
instances.

`configsJSON` can be:

- A JSON array of configuration strings
- A single raw JSON configuration string
- A single supported share URI

Example:

```json
[
  "vless://...",
  "trojan://..."
]
```

`testURL` defaults to `https://www.gstatic.com/generate_204` when empty.
`timeout` is in seconds.

The return value is a JSON array of strings preserving input order:

```json
["143","287","[ERROR]failed to load config: ..."]
```

A numeric string is latency in milliseconds. Failure values include `ERROR`,
`-1`, or strings beginning with `[ERROR]`. Treat every non-numeric item as a
failed measurement.

Latency testing requires an HTTP or SOCKS inbound. A VPN-only/TUN configuration
cannot be tested through this API.

### `GetTotalTraffics`

```c
char *GetTotalTraffics(void);
```

Returns cumulative non-API outbound traffic while the engine is running:

```json
{
  "uplink": 1048576,
  "downlink": 8388608
}
```

Values are bytes.

When stopped:

```json
{"error":"server is not running"}
```

### `GetRealtimeSpeed`

```c
char *GetRealtimeSpeed(void);
```

Returns transfer rates calculated from the difference between consecutive
calls:

```json
{
  "uplinkSpeed": 12500.4,
  "downlinkSpeed": 84000.2
}
```

Values are bytes per second. The first successful call establishes a baseline
and normally returns zero. Poll at a stable interval such as one second. Very
frequent calls below approximately 100 ms do not produce useful measurements.

### `UpdateGeoAssets`

```c
char *UpdateGeoAssets(char *assetPath);
```

Downloads:

- `geosite.dat`
- `geoip.dat`

The source is the latest
`Loyalsoldier/v2ray-rules-dat` GitHub release. Downloads run concurrently.

If `assetPath` is empty, the target is a `Resources` directory beside the host
executable.

Example:

```json
{
  "geosite.dat":"geosite.dat",
  "geoip.dat":"geoip.dat"
}
```

Partial failure is reported per file:

```json
{
  "geosite.dat":"failed: ...",
  "geoip.dat":"geoip.dat"
}
```

This operation performs network and filesystem I/O. Run it off the UI thread.

### `GetVersionInfo`

```c
char *GetVersionInfo(void);
```

Returns build metadata injected by the release pipeline:

```json
{
  "codeVersion": 12,
  "version": "v26.3.27",
  "releaseDate": "2026-06-15 03:00:00 +0330"
}
```

`version` is the upstream Xray release tag. `releaseDate` is the build timestamp
in Tehran time. `codeVersion` is the GitHub Actions run number used for that
binary.

### `Parse`

```c
char *Parse(char *optionsJSON);
```

Auto-detects VLESS, VMess, Trojan, or Shadowsocks from the `uri` field and
returns formatted Xray JSON. It is the recommended parser entry point.

On failure it returns an allocated empty string.

### `ParseVless`

```c
char *ParseVless(char *optionsJSON);
```

Parses a VLESS URI from `optionsJSON.uri`. It supports TCP, WebSocket, HTTP,
QUIC, mKCP, gRPC, TLS, and REALITY share parameters. It returns Xray JSON or an
allocated empty string on failure.

### `ParseTrojan`

```c
char *ParseTrojan(char *optionsJSON);
```

Parses a Trojan URI from `optionsJSON.uri`. TLS is the default security mode
when the URI omits `security`. It returns Xray JSON or an allocated empty string
on failure.

### `ParseVmess`

```c
char *ParseVmess(char *optionsJSON);
```

Parses the base64-encoded JSON payload of a VMess URI. It supports the transport
fields described later in this guide and returns Xray JSON or an allocated
empty string on failure.

### `ParseShadowsocks`

```c
char *ParseShadowsocks(char *optionsJSON);
```

Parses standard Shadowsocks URIs and supported `v2ray-plugin` WebSocket/TLS
options. It returns Xray JSON or an allocated empty string on failure.

All protocol-specific parsers use the same options schema as `Parse`. Use them
when the protocol is already known and protocol mismatch should fail
immediately.

### `JSONToConfigString`

```c
char *JSONToConfigString(char *configJSON);
```

Converts the first eligible outbound to a share URI. It prefers an outbound
tagged `Proxy`; otherwise it chooses the first outbound not tagged `Direct` or
`Reject`, then falls back to the first outbound.

Supported output schemes:

- `vless://`
- `vmess://`
- `trojan://`
- `ss://`

An unsupported or incomplete configuration returns an allocated empty string.
The conversion is intentionally lossy for Xray fields that have no equivalent
share-URI representation.

## Parser Input Schema

The complete parser options object is:

```json
{
  "uri": "vless://...",
  "httpPort": 10809,
  "socksPort": 10808,
  "vpn_mode": false,
  "routingMode": "proxy",
  "geositePath": "/opt/v2root/geosite.dat",
  "geositeFile": "/opt/v2root/geosite.dat",
  "geositeDomain": "ir",
  "geositeDNS": "1.1.1.1",
  "dnsConfig": {
    "servers": ["1.1.1.1", "8.8.8.8"],
    "clientIp": "",
    "tag": "",
    "strategy": "UseIP"
  },
  "geositeRules": [
    {
      "domain": "ads",
      "action": "block"
    },
    {
      "domain": "ir",
      "action": "direct"
    }
  ],
  "geoipRules": [
    {
      "country": "private",
      "action": "direct"
    },
    {
      "country": "cn",
      "outboundTag": "CustomOutbound"
    }
  ]
}
```

### Parser defaults

| Option | Default |
| --- | --- |
| `httpPort` | `10809` |
| `socksPort` | `10808` |
| `vpn_mode` | `false` |
| `routingMode` | `proxy` in `Parse`; protocol-specific functions pass an empty value if omitted |
| VLESS/Trojan network | `tcp` |
| VLESS security | `none` |
| Trojan security | `tls` |

Ports must fit in the unsigned 16-bit range. Applications should validate
`1..65535` before calling the library.

### Routing actions

| Action | Resulting outbound |
| --- | --- |
| `proxy` | `Proxy` |
| `direct` | `Direct` |
| `block` or `reject` | `Reject` |
| `outboundTag` supplied | Exact custom tag |

Recognized geosite aliases include:

`ads`, `porn`, `media`, `anticensorship`, `vpn`, `games`, `dev`, `ai`,
`malware`, `phishing`, `messaging`, `cn`, `not-cn`, `private`, `win-spy`, and
`win-update`.

### Supported transport data

The parser understands common share-URI fields for:

- TCP, including HTTP header camouflage
- WebSocket
- HTTP transport
- QUIC
- mKCP
- gRPC
- TLS
- REALITY
- Shadowsocks `v2ray-plugin` WebSocket/TLS options

For REALITY, the public key (`pbk`) is required.

## Configuration Examples

### Parse a VLESS URI

```json
{
  "uri": "vless://UUID@example.com:443?encryption=none&security=tls&sni=example.com&type=ws&path=%2Fws#Example",
  "httpPort": 10809,
  "socksPort": 10808,
  "vpn_mode": false
}
```

### Validate and start raw JSON

```json
{
  "log": {
    "loglevel": "warning"
  },
  "inbounds": [
    {
      "listen": "127.0.0.1",
      "port": 10808,
      "protocol": "socks",
      "settings": {
        "udp": true
      }
    }
  ],
  "outbounds": [
    {
      "tag": "Proxy",
      "protocol": "freedom",
      "settings": {}
    }
  ]
}
```

### Safe native call pattern

```c
static char *copy_and_release(char *value) {
    if (value == NULL) {
        return NULL;
    }
    char *copy = strdup(value);
    FreeCString(value);
    return copy;
}

char *validation_ptr = ValidateConfig(config_json, "{}");
char *validation = copy_and_release(validation_ptr);
if (validation == NULL) {
    /* Unexpected: ValidateConfig normally returns JSON. */
} else {
    /* Parse validation as JSON and inspect "error" or "result". */
    free(validation);
}

char *start_error = Start(config_json, "{}");
if (start_error != NULL) {
    fprintf(stderr, "Start failed: %s\n", start_error);
    FreeCString(start_error);
}
```

## Language Integration Examples

### Python `ctypes`

```python
import ctypes
import json

lib = ctypes.CDLL("./xray-linux-amd64.so")

lib.FreeCString.argtypes = [ctypes.c_void_p]
lib.FreeCString.restype = None

lib.GetVersionInfo.argtypes = []
lib.GetVersionInfo.restype = ctypes.c_void_p

lib.ValidateConfig.argtypes = [ctypes.c_char_p, ctypes.c_char_p]
lib.ValidateConfig.restype = ctypes.c_void_p

lib.Start.argtypes = [ctypes.c_char_p, ctypes.c_char_p]
lib.Start.restype = ctypes.c_void_p

lib.Stop.argtypes = []
lib.Stop.restype = ctypes.c_void_p


def take_string(pointer):
    if not pointer:
        return None
    try:
        return ctypes.string_at(pointer).decode("utf-8")
    finally:
        lib.FreeCString(pointer)


version = json.loads(take_string(lib.GetVersionInfo()))
print(version)

config = json.dumps({
    "inbounds": [],
    "outbounds": [{"protocol": "freedom", "tag": "Proxy", "settings": {}}],
}).encode()

validation = json.loads(
    take_string(lib.ValidateConfig(config, b"{}"))
)
if "error" in validation:
    raise RuntimeError(validation["error"])

start_error = take_string(lib.Start(config, b"{}"))
if start_error is not None:
    raise RuntimeError(start_error)

try:
    # Application work
    pass
finally:
    stop_error = take_string(lib.Stop())
    if stop_error is not None:
        print(f"Stop warning: {stop_error}")
```

### C# P/Invoke

```csharp
using System;
using System.Runtime.InteropServices;

internal static class V2RootNative
{
    private const string Library = "xray-windows-amd64.dll";

    [DllImport(Library, CallingConvention = CallingConvention.Cdecl)]
    internal static extern IntPtr GetVersionInfo();

    [DllImport(Library, CallingConvention = CallingConvention.Cdecl)]
    internal static extern IntPtr ValidateConfig(
        [MarshalAs(UnmanagedType.LPUTF8Str)] string config,
        [MarshalAs(UnmanagedType.LPUTF8Str)] string options);

    [DllImport(Library, CallingConvention = CallingConvention.Cdecl)]
    internal static extern IntPtr Start(
        [MarshalAs(UnmanagedType.LPUTF8Str)] string config,
        [MarshalAs(UnmanagedType.LPUTF8Str)] string options);

    [DllImport(Library, CallingConvention = CallingConvention.Cdecl)]
    internal static extern IntPtr Stop();

    [DllImport(Library, CallingConvention = CallingConvention.Cdecl)]
    internal static extern void FreeCString(IntPtr value);

    internal static string? TakeString(IntPtr pointer)
    {
        if (pointer == IntPtr.Zero)
            return null;

        try
        {
            return Marshal.PtrToStringUTF8(pointer);
        }
        finally
        {
            FreeCString(pointer);
        }
    }
}
```

Do not declare a returned string directly as C# `string`; doing so prevents the
application from reliably returning ownership through `FreeCString`.

### Dart / Flutter FFI

```dart
import 'dart:ffi';
import 'package:ffi/ffi.dart';

typedef _GetStatusNative = Pointer<Utf8> Function();
typedef _GetStatusDart = Pointer<Utf8> Function();

typedef _FreeNative = Void Function(Pointer<Utf8>);
typedef _FreeDart = void Function(Pointer<Utf8>);

final library = DynamicLibrary.open('xray-windows-amd64.dll');
final getStatus =
    library.lookupFunction<_GetStatusNative, _GetStatusDart>('GetStatus');
final freeCString =
    library.lookupFunction<_FreeNative, _FreeDart>('FreeCString');

String takeString(Pointer<Utf8> pointer) {
  if (pointer == nullptr) {
    throw StateError('Unexpected null native string');
  }
  try {
    return pointer.toDartString();
  } finally {
    freeCString(pointer);
  }
}

final status = takeString(getStatus());
```

Perform blocking calls such as `UpdateGeoAssets` and large latency batches away
from the Flutter UI isolate.

### C++ RAII wrapper

```cpp
#include <memory>
#include <string>

struct V2RootStringDeleter {
    void operator()(char* value) const noexcept {
        FreeCString(value);
    }
};

using V2RootString = std::unique_ptr<char, V2RootStringDeleter>;

std::string get_status() {
    V2RootString value(GetStatus());
    return value ? std::string(value.get()) : std::string();
}
```

## Concurrency and Thread Safety

### Serialized lifecycle

`Start`, `Stop`, and `GetStatus` synchronize access to global runtime state.
Do not design the host application around concurrent start/stop races even
though internal locking exists. Serialize lifecycle commands in the application
layer.

### Statistics

Realtime speed baseline state is protected internally, but concurrent callers
share the same baseline. For stable UI metrics, designate one polling task as
the sole caller of `GetRealtimeSpeed`.

### Parser calls

Parser calls do not use the lifecycle mutex. They may set the process-wide
`XRAY_LOCATION_ASSET` environment variable when geo paths are supplied.
Therefore, concurrent parser calls using different asset directories should be
serialized.

### Latency tests

`TestLatency` runs configurations concurrently and may consume significant CPU,
memory, sockets, and file descriptors. Limit batch size in user-facing
applications.

### Callbacks

The ABI currently exposes no callbacks. Poll status and statistics from the
host application.

## Error Handling

V2Root Core currently uses three error styles:

| API family | Error representation |
| --- | --- |
| `Start`, `Stop` | Non-null human-readable string |
| JSON operational APIs | JSON object containing `error` |
| Parser/conversion APIs | Empty string |
| `TestLatency` | Per-item string such as `[ERROR]...`, `ERROR`, or `-1` |

Recommended wrapper strategy:

1. Convert and free the native result immediately.
2. Convert each API family to one host-language exception/result type.
3. Preserve the original native message for diagnostics.
4. Avoid matching complete error text.
5. Log version metadata with every failure report.

Do not include share URIs, UUIDs, passwords, private keys, or complete
configurations in telemetry unless the user explicitly opts in.

## Geo Assets

Routing rules using `geosite:` or `geoip:` require compatible asset files.

Recommended application layout:

```text
V2RootApp/
  bin/
    xray-windows-amd64.dll
  resources/
    geosite.dat
    geoip.dat
```

Call `UpdateGeoAssets(resourcesPath)` during an explicit update operation, not
on every startup. Downloads are currently replaced directly at the destination;
applications requiring transactional updates should download through their own
updater, verify files, and atomically replace the active assets.

The library sets `XRAY_LOCATION_ASSET` when a valid geo path is provided. This
is a process-wide environment setting.

## Logging and Observability

Set logging before startup:

```c
SetLogLevel("warning");
SetLogOutput("/var/log/v2root/xray.log");
```

Operational recommendations:

- Rotate log files in the host application or operating system.
- Restrict log directory permissions.
- Avoid `debug` in production unless diagnosing a specific problem.
- Record `GetVersionInfo` in support bundles.
- Record status transitions and sanitized error strings.
- Poll total traffic less frequently than realtime speed.

## Security Guidance

- Treat all imported share URIs and JSON as untrusted input.
- Call `ValidateConfig` before `Start`.
- Bind proxy inbounds to loopback unless remote access is intentional.
- Protect geo asset and log directories from untrusted modification.
- Never expose the internal StatsService port; the library binds it to
  `127.0.0.1`.
- TUN mode may require elevated operating-system privileges.
- Do not unload the shared library while its runtime or native calls are active.
- Verify release ZIP checksums before distribution.
- Keep the DLL/SO architecture aligned with the host process.
- Store secrets outside configuration logs and crash reports.

## Production Integration Checklist

### Packaging

- [ ] Select the correct OS and architecture ZIP.
- [ ] Verify the ZIP's internal `SHA256SUMS`.
- [ ] Ship the matching generated header for native builds.
- [ ] Use an absolute library path where possible.

### ABI wrapper

- [ ] Declare C calling conventions exactly as generated.
- [ ] Marshal all text as UTF-8.
- [ ] Treat null and empty strings differently.
- [ ] Call `FreeCString` exactly once for every non-null returned string.
- [ ] Copy native text before releasing its pointer.

### Runtime

- [ ] Configure logging before startup.
- [ ] Validate before starting.
- [ ] Serialize `Start` and `Stop`.
- [ ] Prevent multiple engines in one process.
- [ ] Stop before unloading or process shutdown.
- [ ] Poll speed from one task at a stable interval.

### User experience

- [ ] Run network and latency operations off the UI thread.
- [ ] Show actionable validation errors without exposing credentials.
- [ ] Distinguish stopped, starting, running, and stopping states.
- [ ] Make geo asset updates explicit and observable.

### Diagnostics

- [ ] Include `GetVersionInfo` in support reports.
- [ ] Include OS, process architecture, and package name.
- [ ] Include sanitized validation/start errors.
- [ ] Never include full credentials or share URIs by default.

## Compatibility Policy

Release binaries are built against a specific upstream Xray tag. The
`GetVersionInfo.version` field identifies that tag. Consumers should:

- Prefer capability detection over assumptions.
- Keep the generated header synchronized with the binary.
- Run integration tests before adopting a new release.
- Treat undocumented symbols and internal JSON normalization as implementation
  details.

The public ABI is represented by the generated header and the functions
documented here. Changes to return schemas or ownership rules should be treated
as integration-significant even if function names remain unchanged.
