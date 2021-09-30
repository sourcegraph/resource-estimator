#!/usr/bin/env bash

cd "$(dirname "${BASH_SOURCE[0]}")"/..
set -euxo pipefail

go get github.com/hajimehoshi/wasmserve

GOROOT=$(go env GOROOT)
export GOROOT

wasmserve -allow-origin='*'
