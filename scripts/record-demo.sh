#!/usr/bin/env bash
# Paced screencast for the everestctl POC.
#
# Simulates a real shell session: live prompt with your $USER and
# hostname, character-by-character typing with jittered delays, and
# narration as shell comments — so the recording looks like a person
# at a terminal, not a tutorial script.
#
# Usage:
#   ./scripts/record-demo.sh              # ~85s, normal pacing
#   SPEED=fast ./scripts/record-demo.sh   # ~55s
#   SPEED=slow ./scripts/record-demo.sh   # ~130s
#
# Capture:
#   asciinema rec demo.cast --overwrite --window-size 120x32 \
#       -c './scripts/record-demo.sh'
#   agg --theme monokai --speed 1.0 --font-size 16 demo.cast demo.gif

set -euo pipefail
cd "$(dirname "$0")/.."

case "${SPEED:-normal}" in
    fast)   CHAR_MIN=15; CHAR_MAX=45;  THINK=0.5; READ_SHORT=0.6; READ_LONG=1.6 ;;
    slow)   CHAR_MIN=45; CHAR_MAX=110; THINK=1.4; READ_SHORT=1.6; READ_LONG=3.5 ;;
    *)      CHAR_MIN=25; CHAR_MAX=70;  THINK=0.8; READ_SHORT=1.0; READ_LONG=2.2 ;;
esac

USERNAME="$(whoami)"
HOST_SHORT="$(hostname -s)"

green="\033[32m"; blue="\033[34m"; dim="\033[2m"; reset="\033[0m"

# render the shell prompt using the live username, host and cwd, so
# whoever runs this script sees their own identity in the recording.
print_prompt() {
    local cwd
    cwd=$(pwd | sed "s|$HOME|~|")
    printf "${green}%s@%s${reset} ${blue}%s${reset} %% " \
        "$USERNAME" "$HOST_SHORT" "$cwd"
}

# random integer in [min, max] using bash $RANDOM, returned as seconds
# with millisecond precision. Used for per-character typing jitter.
rand_delay() {
    local min=$1
    local max=$2
    local range=$((max - min + 1))
    local ms=$((min + RANDOM % range))
    printf '0.%03d' "$ms"
}

# type a string character by character with jittered delays. Stdout is
# flushed implicitly by the pty asciinema attaches.
type_out() {
    local s="$1" i
    for ((i = 0; i < ${#s}; i++)); do
        printf '%s' "${s:i:1}"
        sleep "$(rand_delay "$CHAR_MIN" "$CHAR_MAX")"
    done
}

# show a shell-comment-style narration at the prompt. Reads as if the
# user is annotating what they're about to do.
say() {
    print_prompt
    printf "${dim}# "
    type_out "$1"
    printf "${reset}"
    sleep "$THINK"
    printf '\n'
}

# print the prompt, type the command, brief think-pause, then run it.
# pause_after lets each call tune the post-command read time.
run() {
    local cmd="$1" pause_after="${2:-$READ_LONG}"
    print_prompt
    type_out "$cmd"
    sleep "$THINK"
    printf '\n'
    eval "$cmd"
    sleep "$pause_after"
}

# --- session ---

say "build the binary and run the test suite with coverage"
run "go build -o bin/everestctl ./cmd/everestctl" "$READ_SHORT"
run "go test ./... -coverpkg=./internal/... -coverprofile=cover.out >/dev/null && go tool cover -func=cover.out | tail -1"

say "let's see what the CLI looks like"
run "./bin/everestctl --help"

say "list databases — table by default, json for scripts"
run "./bin/everestctl db list"
run "./bin/everestctl db list -o json"

say "inspect one"
run "./bin/everestctl db get orders-pg -o yaml"

say "spin up a new postgres, drop an old mongo, tail logs"
run "./bin/everestctl db create billing-pg --engine postgresql --version 16.2 --replicas 2" "$READ_SHORT"
run "./bin/everestctl db delete sessions-mongo --yes" "$READ_SHORT"
run "./bin/everestctl db logs orders-pg | head -3"

say "clusters work the same way"
run "./bin/everestctl cluster list"
run "./bin/everestctl cluster register prod --endpoint https://k8s.prod.example.com --context prod" "$READ_SHORT"

say "and plugins"
run "./bin/everestctl plugin list"
run "./bin/everestctl plugin install pmm" "$READ_SHORT"
run "./bin/everestctl plugin configure backup-s3 --set bucket=my-backups --set region=eu-west-1" "$READ_SHORT"

say "completion ships for bash, zsh, fish, powershell"
run "./bin/everestctl completion bash | head -8"

print_prompt
printf '\n'
sleep 1
