# V2Root Core

V2Root shared-library bindings for Xray-core.

## Automated releases

The `Build latest Xray with V2Root` GitHub Actions workflow runs once per day.
It checks the latest stable
[XTLS/Xray-core release](https://github.com/XTLS/Xray-core/releases), adds this
repository's `v2root` package to that exact source tag, and publishes Linux
amd64 and Windows amd64 shared libraries.

Each generated release is tagged as `v2root-<xray-tag>`. The workflow skips a
version when that release tag already exists. It can also be run manually from
the Actions page, with the optional force setting enabled to rebuild the latest
version.
