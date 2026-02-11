#!/usr/bin/env bash
# Clean an asciicast v3 file:
#   - Strip hostname from shell prompts
#   - Cap delays to remove long pauses
#
# Usage:
#   ./scripts/clean-cast.sh docs/termsuji-demo.cast --inplace --max-delay 0.5
#   ./scripts/clean-cast.sh docs/termsuji-demo.cast -o docs/clean.cast
set -euo pipefail

python3 -c '
import json, sys, argparse

def main():
    p = argparse.ArgumentParser(description="Clean an asciicast v3 file")
    p.add_argument("input", help="Path to .cast file")
    p.add_argument("--max-delay", type=float, default=0.5, help="Cap delays at this value in seconds (default: 0.5)")
    p.add_argument("--inplace", action="store_true", help="Overwrite the input file")
    p.add_argument("-o", "--output", help="Write to a different file")
    args = p.parse_args()

    if not args.inplace and not args.output:
        p.error("Specify --inplace or -o OUTPUT")

    PROMPT_PREFIX = "grins@Deans-MacBook-Pro termsuji-local"

    with open(args.input, "r") as f:
        lines = f.readlines()

    out = []
    # Line 1: JSON header, pass through unchanged
    out.append(lines[0])

    for line in lines[1:]:
        line = line.strip()
        if not line:
            continue
        event = json.loads(line)
        delay, etype, data = event[0], event[1], event[2]

        # Strip prompt prefix
        if PROMPT_PREFIX in data:
            data = data.replace(PROMPT_PREFIX, "")

        # Cap delay
        if delay > args.max_delay:
            delay = args.max_delay

        out.append(json.dumps([delay, etype, data], ensure_ascii=False) + "\n")

    dest = args.input if args.inplace else args.output
    with open(dest, "w") as f:
        f.writelines(out)

    print(f"Wrote {len(out)} lines to {dest}")

main()
' "$@"
