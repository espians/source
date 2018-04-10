#! /usr/bin/env bash

# Public Domain (-) 2018-present, The Espian Source Authors.
# See the Espian Source UNLICENSE file for details.

cd "$(dirname "$0")"

go tool cgo -exportheader _cgo_export.h worker.go
