#!/usr/bin/env bash
# Download official UnRAR binaries from rarlab.com into third_party/unrar/<os>_<arch>/.
# Also writes release-asset-style copies under dist/unrar/ for packaging.
#
# Usage:
#   ./scripts/fetch-unrar.sh              # detect host OS/arch
#   ./scripts/fetch-unrar.sh --all        # all supported release platforms
#   ./scripts/fetch-unrar.sh darwin_arm64 # one or more explicit platforms
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="${ROOT}/third_party/unrar"
DIST="${ROOT}/dist/unrar"
BASE="https://www.rarlab.com/rar"
VERSION="${UNRAR_PKG_VERSION:-723}"

detect_host_platform() {
  local os arch
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  arch="$(uname -m)"
  case "$os" in
    darwin) os="darwin" ;;
    linux)  os="linux" ;;
    mingw*|msys*|cygwin*|windows*) os="windows" ;;
    *)
      echo "error: unsupported host OS from uname: $(uname -s)" >&2
      exit 1
      ;;
  esac
  case "$arch" in
    arm64|aarch64) arch="arm64" ;;
    x86_64|amd64)  arch="amd64" ;;
    *)
      echo "error: unsupported host arch from uname: $(uname -m)" >&2
      exit 1
      ;;
  esac
  echo "${os}_${arch}"
}

mkdir -p "$OUT" "$DIST"

fetch_tar() {
  local plat="$1" url="$2" bin_name="${3:-unrar}"
  local tmp dir dest
  tmp="$(mktemp -d)"
  echo "→ ${plat}: ${url}"
  curl -fsSL -o "${tmp}/pkg.tgz" "$url"
  tar -xzf "${tmp}/pkg.tgz" -C "$tmp"
  # Packages unpack to rar/unrar
  if [[ ! -f "${tmp}/rar/${bin_name}" ]]; then
    echo "error: ${bin_name} not found in package for ${plat}" >&2
    find "$tmp" -maxdepth 3 -type f -print >&2
    rm -rf "$tmp"
    exit 1
  fi
  dir="${OUT}/${plat}"
  mkdir -p "$dir"
  dest="${dir}/${bin_name}"
  cp "${tmp}/rar/${bin_name}" "$dest"
  chmod +x "$dest"
  # Keep license from package when present
  if [[ -f "${tmp}/rar/license.txt" ]]; then
    cp "${tmp}/rar/license.txt" "${OUT}/LICENSE.txt"
  fi
  # Release asset naming: unrar-<goos>-<goarch>
  local asset="${DIST}/unrar-${plat/_/-}"
  cp "$dest" "$asset"
  chmod +x "$asset"
  echo "  installed ${dest}"
  echo "  asset     ${asset}"
  rm -rf "$tmp"
}

fetch_win() {
  local plat="windows_amd64"
  local url="${BASE}/unrarw64.exe"
  local dir="${OUT}/${plat}"
  mkdir -p "$dir"
  echo "→ ${plat}: ${url}"
  curl -fsSL -o "${dir}/unrar.exe" "$url"
  cp "${dir}/unrar.exe" "${DIST}/unrar-windows-amd64.exe"
  echo "  installed ${dir}/unrar.exe"
  echo "  asset     ${DIST}/unrar-windows-amd64.exe"
}

ALL_PLATFORMS=(darwin_arm64 darwin_amd64 linux_amd64 windows_amd64)
PLATFORMS=()

if [[ $# -eq 0 ]]; then
  host="$(detect_host_platform)"
  echo "detected host platform: ${host}"
  PLATFORMS=("$host")
else
  for arg in "$@"; do
    case "$arg" in
      --all|-a)
        PLATFORMS=("${ALL_PLATFORMS[@]}")
        break
        ;;
      -h|--help)
        cat <<'EOF'
Usage: fetch-unrar.sh [--all] [platform...]

With no arguments, detects the current OS/arch (uname) and fetches that binary.
  --all, -a     fetch every supported release platform
  platform      one of: darwin_arm64 darwin_amd64 linux_amd64 windows_amd64
EOF
        exit 0
        ;;
      *)
        PLATFORMS+=("$arg")
        ;;
    esac
  done
fi

for plat in "${PLATFORMS[@]}"; do
  case "$plat" in
    darwin_arm64)  fetch_tar "$plat" "${BASE}/rarmacos-arm-${VERSION}.tar.gz" ;;
    darwin_amd64)  fetch_tar "$plat" "${BASE}/rarmacos-x64-${VERSION}.tar.gz" ;;
    linux_amd64)   fetch_tar "$plat" "${BASE}/rarlinux-x64-${VERSION}.tar.gz" ;;
    windows_amd64) fetch_win ;;
    linux_arm64)
      echo "error: linux_arm64 is not published by RARLab in this script's package set; use ZIPSTAR_UNRAR or rardecode fallback" >&2
      exit 1
      ;;
    *)
      echo "unknown platform: $plat (expected darwin_arm64|darwin_amd64|linux_amd64|windows_amd64 or --all)" >&2
      exit 1
      ;;
  esac
done

echo "done."
