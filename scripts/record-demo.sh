#!/usr/bin/env bash
# Paced screencast script for the everestctl POC.
#
# Usage:
#   ./scripts/record-demo.sh            # default pacing, ~75s total
#   SPEED=fast ./scripts/record-demo.sh # ~45s, good for previews
#   SPEED=slow ./scripts/record-demo.sh # ~110s, easier to follow
#
# Capture with: asciinema rec demo.cast    (or OBS, 1080p, font >= 18pt)
#
# Note: each binary invocation is a fresh process so the in-memory
# backend resets between calls — the script demonstrates each command's
# *shape* against seeded data. The full create -> get -> delete lifecycle
# is proven inside one process by TestDBCreateGetDelete_RoundTrip
# (internal/cli/cli_test.go) which the demo runs in step 1.

set -euo pipefail
cd "$(dirname "$0")/.."

case "${SPEED:-normal}" in
    fast)   THINK=0.4; READ=1.2 ;;
    slow)   THINK=1.2; READ=3.5 ;;
    *)      THINK=0.7; READ=2.2 ;;
esac

cyan="\033[1;36m"; dim="\033[2m"; bold="\033[1m"; reset="\033[0m"

# title shows a banner without running a command.
title() {
    clear 2>/dev/null || printf '\n\n'
    printf "${bold}%s${reset}\n${dim}%s${reset}\n\n" "$1" "${2:-}"
    sleep "$READ"
}

# say prints a section header between commands.
say() {
    printf "\n${bold}── %s${reset}\n" "$1"
    sleep "$THINK"
}

# run echoes the command as if typed, then executes it.
run() {
    printf "${cyan}\$ %s${reset}\n" "$*"
    sleep "$THINK"
    eval "$@"
    sleep "$READ"
}

title "everestctl POC" "CNCF OpenEverest — LFX 2026 Term 2"

say "1. Build + run the tests (fresh, no cache)"
run "go clean -testcache"
run "go build -o bin/everestctl ./cmd/everestctl"
run "go test ./... -coverpkg=./internal/... -coverprofile=cover.out >/dev/null && go tool cover -func=cover.out | tail -1"

say "2. The CLI surface"
run "./bin/everestctl --help"

say "3. Listing databases — table and JSON output"
run "./bin/everestctl db list"
run "./bin/everestctl db list -o json"

say "4. Inspect and mutate seeded databases"
run "./bin/everestctl db get orders-pg -o yaml"
run "./bin/everestctl db create billing-pg --engine postgresql --version 16.2 --replicas 2"
run "./bin/everestctl db delete sessions-mongo --yes"
run "./bin/everestctl db logs orders-pg | head -3"

say "5. Cluster management"
run "./bin/everestctl cluster list"
run "./bin/everestctl cluster register prod --endpoint https://k8s.prod.example.com --context prod"

say "6. Plugin management"
run "./bin/everestctl plugin list"
run "./bin/everestctl plugin install pmm"
run "./bin/everestctl plugin configure backup-s3 --set bucket=my-backups --set region=eu-west-1"

say "7. Shell completion (bash, first 8 lines)"
run "./bin/everestctl completion bash | head -8"

printf "\n${bold}Done — see README for architecture diagrams & roadmap.${reset}\n"
printf "${dim}Repo: https://github.com/alokkumardalei-wq/everestctl-poc${reset}\n"
