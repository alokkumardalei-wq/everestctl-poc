#!/usr/bin/env bash
# Runnable demo of the everestctl POC. Builds the binary, then walks
# through every command surface. Each section is self-contained because
# the in-memory backend resets between binary invocations.
set -euo pipefail

cd "$(dirname "$0")/.."
go build -o bin/everestctl ./cmd/everestctl

bin=./bin/everestctl

section() { printf '\n\033[1m== %s ==\033[0m\n' "$1"; }

section "db list (table)"
"$bin" db list

section "db list (json)"
"$bin" db list -o json

section "db create + get + delete within one process is exercised by tests"
echo "(see internal/cli/cli_test.go::TestDBCreateGetDelete_RoundTrip)"

section "cluster list"
"$bin" cluster list

section "plugin list"
"$bin" plugin list

section "shell completion (bash, first lines)"
"$bin" completion bash | head -10
