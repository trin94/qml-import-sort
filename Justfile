# SPDX-FileCopyrightText: Elias Mueller
#
# SPDX-License-Identifier: MIT

alias fmt := format

@_default:
    just --list --unsorted

init:
    go install honnef.co/go/tools/cmd/staticcheck@2025.1.1
    go install github.com/securego/gosec/v2/cmd/gosec@v2.22.10

format:
    prek run --all-files
    go fmt ./...

lint:
    staticcheck ./...
    gosec \
    	-exclude=G304 \
    	-quiet ./...

# Build the qmlimportsort binary
@build:
	go build -o qmlimportsort ./cmd/qmlimportsort

# Run all tests
@test *FLAGS:
	go test ./... {{ FLAGS }}
