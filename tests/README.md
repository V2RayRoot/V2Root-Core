# V2Root Test Suite

The release pipeline runs these tests after compiling the shared libraries and
before publishing a release:

- `smoke_shared_library.py` loads the Linux shared library through `ctypes` and
  exercises every exported C API function, including parser, lifecycle,
  validation, traffic, latency, version, logging, asset-update error handling,
  and memory-release paths.
- `verify_windows_exports.py` inspects the Windows DLL export table and fails if
  any public API symbol is missing.
- `validate_release_artifacts.py` checks all expected architecture ZIPs,
  validates their contents and internal `SHA256SUMS`, and rejects any loose
  non-ZIP release file.

Run the Linux suite from the repository root after building the library:

```bash
python3 tests/smoke_shared_library.py ./dist/xray-linux-amd64.so
```
