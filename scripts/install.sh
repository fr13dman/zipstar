#!/usr/bin/env bash
# Build zipstar for this machine and install it, plus a matching UnRAR binary.
#
# Usage:
#   ./scripts/install.sh                 # install to /usr/local/bin (sudo only if needed)
#   ./scripts/install.sh --prefix DIR    # install somewhere else, e.g. ~/.local/bin
#   ./scripts/install.sh --uninstall     # remove what a previous install placed
#
# Options:
#   --prefix DIR   install directory (default /usr/local/bin)
#   --unrar FILE   install this UnRAR binary instead of fetching one
#   --no-unrar     skip UnRAR; zipstar falls back to its built-in RAR decoder
#   --no-path      never edit shell startup files, even if DIR is not on PATH
#   --uninstall    remove zipstar, its UnRAR copy, the license, and the PATH block
#
# /usr/sbin is deliberately not the default: macOS System Integrity Protection
# blocks writes to it even with sudo, and on Linux it is reserved for system
# administration tools.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PREFIX="/usr/local/bin"
UNRAR_SRC=""
WITH_UNRAR=1
EDIT_PATH=1
UNINSTALL=0

MARK_BEGIN="# >>> zipstar PATH >>>"
MARK_END="# <<< zipstar PATH <<<"

usage() { sed -n '2,15p' "$0" | sed 's/^# \{0,1\}//'; }
die() { echo "error: $*" >&2; exit 1; }

while [[ $# -gt 0 ]]; do
  case "$1" in
    --prefix)    [[ $# -ge 2 ]] || die "--prefix needs a directory"; PREFIX="$2"; shift 2 ;;
    --prefix=*)  PREFIX="${1#*=}"; shift ;;
    --unrar)     [[ $# -ge 2 ]] || die "--unrar needs a file"; UNRAR_SRC="$2"; shift 2 ;;
    --unrar=*)   UNRAR_SRC="${1#*=}"; shift ;;
    --no-unrar)  WITH_UNRAR=0; shift ;;
    --no-path)   EDIT_PATH=0; shift ;;
    --uninstall) UNINSTALL=1; shift ;;
    -h|--help)   usage; exit 0 ;;
    *)           die "unknown option: $1 (see --help)" ;;
  esac
done

case "$PREFIX" in
  "~")   PREFIX="$HOME" ;;
  "~/"*) PREFIX="$HOME/${PREFIX#"~/"}" ;;
  /*)    ;;
  *)     PREFIX="$PWD/$PREFIX" ;;
esac
PREFIX="${PREFIX%/}"
SHARE_DIR="$(dirname "$PREFIX")/share/zipstar"

detect_host_platform() {
  local os arch
  case "$(uname -s)" in
    Darwin) os="darwin" ;;
    Linux)  os="linux" ;;
    *) die "unsupported OS $(uname -s); on Windows use scripts/install.ps1" ;;
  esac
  case "$(uname -m)" in
    arm64|aarch64) arch="arm64" ;;
    x86_64|amd64)  arch="amd64" ;;
    *) die "unsupported architecture $(uname -m)" ;;
  esac
  echo "${os}_${arch}"
}

PLAT="$(detect_host_platform)"
# Installed beside zipstar under a platform-specific name, so it never clobbers
# a system or Homebrew `unrar` that may already live in the same directory.
UNRAR_NAME="unrar-${PLAT/_/-}"

# sudo is used only when the target (or its nearest existing parent) is not
# writable by the current user.
needs_sudo() {
  local dir="$1"
  while [[ ! -e "$dir" ]]; do dir="$(dirname "$dir")"; done
  [[ ! -w "$dir" ]]
}
SUDO=""
if needs_sudo "$PREFIX" || needs_sudo "$SHARE_DIR"; then
  command -v sudo >/dev/null || die "$PREFIX is not writable and sudo is unavailable; pass --prefix"
  SUDO="sudo"
  echo "note: $PREFIX is not writable by $(id -un); using sudo"
fi
run() { $SUDO "$@"; }

rc_file() {
  case "$(basename "${SHELL:-}")" in
    zsh)  echo "${ZDOTDIR:-$HOME}/.zshrc" ;;
    bash) if [[ "$(uname -s)" == Darwin ]]; then echo "$HOME/.bash_profile"; else echo "$HOME/.bashrc"; fi ;;
    fish) echo "${XDG_CONFIG_HOME:-$HOME/.config}/fish/config.fish" ;;
    *)    echo "$HOME/.profile" ;;
  esac
}

remove_path_block() {
  local f="$1" tmp
  [[ -f "$f" ]] && grep -qF "$MARK_BEGIN" "$f" || return 0
  tmp="$(mktemp)"
  awk -v b="$MARK_BEGIN" -v e="$MARK_END" '$0==b{skip=1; next} $0==e{skip=0; next} !skip' "$f" >"$tmp"
  cat "$tmp" >"$f" # keep the original file's permissions
  rm -f "$tmp"
  echo "removed zipstar PATH entry from $f"
}

on_path() { [[ ":$PATH:" == *":$PREFIX:"* ]]; }

if [[ $UNINSTALL -eq 1 ]]; then
  run rm -f "$PREFIX/zipstar" "$PREFIX/$UNRAR_NAME"
  run rm -rf "$SHARE_DIR"
  for f in "${ZDOTDIR:-$HOME}/.zshrc" "$HOME/.bash_profile" "$HOME/.bashrc" "$HOME/.profile" \
           "${XDG_CONFIG_HOME:-$HOME/.config}/fish/config.fish"; do
    remove_path_block "$f"
  done
  echo "uninstalled zipstar from $PREFIX"
  exit 0
fi

command -v go >/dev/null || die "Go is required to build zipstar (https://go.dev/dl/)"

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

VERSION="$(git -C "$ROOT" describe --tags --always --dirty 2>/dev/null || echo dev)"
echo "building zipstar $VERSION for $PLAT..."
(cd "$ROOT" && go build -trimpath \
  -ldflags "-s -w -X github.com/fr13dman/zipstar-cli/internal/version.Version=$VERSION" \
  -o "$TMP/zipstar" ./cmd/zipstar)

UNRAR_BIN=""
if [[ $WITH_UNRAR -eq 1 ]]; then
  if [[ -n "$UNRAR_SRC" ]]; then
    [[ -f "$UNRAR_SRC" ]] || die "--unrar file not found: $UNRAR_SRC"
    UNRAR_BIN="$UNRAR_SRC"
  elif [[ -f "$ROOT/third_party/unrar/$PLAT/unrar" ]]; then
    UNRAR_BIN="$ROOT/third_party/unrar/$PLAT/unrar"
  elif "$ROOT/scripts/fetch-unrar.sh" "$PLAT" >/dev/null 2>&1; then
    UNRAR_BIN="$ROOT/third_party/unrar/$PLAT/unrar"
  else
    echo "warning: could not fetch UnRAR for $PLAT; zipstar will use its built-in RAR decoder" >&2
  fi
fi

run mkdir -p "$PREFIX"
run install -m 0755 "$TMP/zipstar" "$PREFIX/zipstar"
if [[ -n "$UNRAR_BIN" ]]; then
  run install -m 0755 "$UNRAR_BIN" "$PREFIX/$UNRAR_NAME"
  run mkdir -p "$SHARE_DIR"
  run install -m 0644 "$ROOT/third_party/unrar/LICENSE.txt" "$SHARE_DIR/UNRAR-LICENSE.txt"
fi

"$PREFIX/zipstar" --help >/dev/null || die "installed zipstar failed to run"

echo "installed zipstar $VERSION -> $PREFIX/zipstar"
if [[ -n "$UNRAR_BIN" ]]; then
  echo "installed UnRAR   -> $PREFIX/$UNRAR_NAME (license: $SHARE_DIR/UNRAR-LICENSE.txt)"
fi

if on_path; then
  echo "$PREFIX is already on PATH"
  resolved="$(command -v zipstar || true)"
  if [[ -n "$resolved" && "$resolved" != "$PREFIX/zipstar" ]]; then
    echo "warning: $resolved comes earlier on PATH and will run instead" >&2
  fi
elif [[ $EDIT_PATH -eq 0 ]]; then
  echo "$PREFIX is not on PATH; add it to run zipstar from anywhere"
else
  rc="$(rc_file)"
  if [[ -f "$rc" ]] && grep -qF "$MARK_BEGIN" "$rc"; then
    echo "PATH entry already present in $rc"
  else
    mkdir -p "$(dirname "$rc")"
    {
      echo ""
      echo "$MARK_BEGIN"
      if [[ "$rc" == *.fish ]]; then
        printf 'fish_add_path "%s"\n' "$PREFIX"
      else
        printf 'export PATH="%s:$PATH"\n' "$PREFIX"
      fi
      echo "$MARK_END"
    } >>"$rc"
    echo "added $PREFIX to PATH in $rc"
  fi
  echo "open a new terminal (or run: source $rc) to use zipstar"
fi
