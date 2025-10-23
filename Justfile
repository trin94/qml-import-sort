# SPDX-FileCopyrightText: Elias Mueller
#
# SPDX-License-Identifier: MIT

alias fmt := format

@_default:
    just --list --unsorted

init:
    go install honnef.co/go/tools/cmd/staticcheck@2025.1.1

format:
    prek run --all-files
    go fmt ./...

lint:
    staticcheck ./...

# Build the qmlimportsort binary
@build:
	go build -o qmlimportsort ./cmd/qmlimportsort

# Run all tests
@test *FLAGS:
	go test ./... {{ FLAGS }}
