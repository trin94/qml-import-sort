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
    uv run prek --config .config/prek.toml run --all-files
    go fmt ./...

update-git-hooks:
    uv run prek --config .config/prek.toml auto-update

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

# Build the qmlimportsort binary
[group('build')]
@build:
    go build -o qmlimportsort ./cmd/qmlimportsort
