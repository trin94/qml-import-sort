# SPDX-FileCopyrightText: Elias Mueller
#
# SPDX-License-Identifier: MIT

alias fmt := format

@_default:
    just --list --unsorted

format:
    prek run --all-files
    go fmt ./...

# Build the qmlimportsort binary
@build:
	go build -o qmlimportsort ./cmd/qmlimportsort

# Run all tests
@test *FLAGS:
	go test ./... {{ FLAGS }}
