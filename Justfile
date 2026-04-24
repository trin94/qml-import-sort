# SPDX-FileCopyrightText: Elias Mueller
#
# SPDX-License-Identifier: MIT

alias fmt := format

@_default:
    just --list --unsorted

init:
    go install honnef.co/go/tools/cmd/staticcheck@2025.1.1
    go install github.com/securego/gosec/v2/cmd/gosec@v2.22.10

# Build the qmlimportsort binary
@build:
    go build -o qmlimportsort ./cmd/qmlimportsort

[group('dev')]
format:
    prek run --all-files
    go fmt ./...

[group('dev')]
lint:
    staticcheck ./...
    gosec \
    	-exclude=G304 \
    	-quiet ./...

# Run all tests
[group('dev')]
@test *FLAGS:
    go clean -testcache
    go test ./... {{ FLAGS }}
