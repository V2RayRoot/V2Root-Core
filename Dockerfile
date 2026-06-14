# syntax=docker/dockerfile:1

# ----------- Windows DLL Build (MinGW-w64) -----------
FROM golang:1.25 AS build-windows
RUN apt-get update && apt-get install -y mingw-w64
WORKDIR /src
COPY . .
# Build DLL for Windows 64-bit
RUN CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc go build -buildmode=c-shared -o /dist/xray.dll ./v2root

# ----------- Linux SO Build (Native) -----------
FROM golang:1.25 AS build-linux
WORKDIR /src
COPY . .
# Build SO for Linux 64-bit
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -buildmode=c-shared -o /dist/xray.so ./v2root

# ----------- Output Stage -----------
FROM debian:bullseye-slim AS dist
WORKDIR /dist
COPY --from=build-windows /dist/xray.dll ./
COPY --from=build-linux /dist/xray.so ./
# Optionally copy headers
COPY --from=build-windows /src/v2root/xray.h ./
