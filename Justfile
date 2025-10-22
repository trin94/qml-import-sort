# SPDX-FileCopyrightText: Elias Mueller
#
# SPDX-License-Identifier: MIT

alias fmt := format

@_default:
    just --list

format:
    prek run --all-files
    go fmt

@test:
	go test ./... -v
