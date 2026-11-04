#!/usr/bin/env bash
# Minimal UnRAR stand-in for progress / OnFile / OnVolume tests.
# Output mirrors real UnRAR 7.x: volume headers, dest-prefixed member lines,
# backspace-redrawn percentages, and "..." continuation lines.
set -euo pipefail

cmd=""
archive=""
dest=""
for a in "$@"; do
  case "$a" in
    lb|x|t) cmd="$a" ;;
    -p*|-o*|-y|-idq|-v) ;;
    *)
      if [[ -z "$archive" ]]; then
        archive="$a"
      else
        dest="$a"
      fi
      ;;
  esac
done

case "$cmd" in
  lb)
    # -v lists split members once per volume they span.
    printf '%s\n' "a.txt" "a.txt" "b.txt"
    exit 0
    ;;
  t)
    exit 0
    ;;
  x)
    dest="${dest%/}"
    mkdir -p "$dest"
    vol2="${archive%.rar}.part2.rar"
    printf '\nUNRAR 7.23 freeware      Copyright (c) 1993-2026 Alexander Roshal\n\n'
    printf 'Extracting from %s\n\n' "$archive"
    printf 'Extracting  %s/a.txt                    \b\b\b\b 40%%\n' "$dest"
    # Pause so callers can observe callbacks before the process exits.
    sleep 0.15
    printf '\nExtracting from %s\n\n' "$vol2"
    printf '...         a.txt                       \b\b\b\b100%%\b\b\b\b\b  OK \n'
    echo "a" >"$dest/a.txt"
    printf 'Extracting  %s/b.txt                    \b\b\b\b\b  OK \n' "$dest"
    echo "b" >"$dest/b.txt"
    printf 'All OK\n'
    exit 0
    ;;
  *)
    echo "fake_unrar: unknown command" >&2
    exit 2
    ;;
esac
