#!/usr/bin/env bash
set -euo pipefail
export GOBIND=gobind-matsuri
pushd ../third_party/gomobile
go install ./cmd/gomobile ./cmd/gobind
popd
GOPATH=${GOPATH:-$(go env GOPATH)}
cp "$GOPATH/bin/gomobile" "$GOPATH/bin/gomobile-matsuri"
cp "$GOPATH/bin/gobind" "$GOPATH/bin/gobind-matsuri"
gomobile-matsuri init
