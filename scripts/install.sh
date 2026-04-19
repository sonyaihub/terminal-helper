#!/bin/sh
# wut installer.
#
# Download + install the latest release binary from GitHub. Supports macOS and
# Linux on amd64/arm64. Falls back to `go install` from source if a Go
# toolchain is available and no matching release asset is found.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/sonyaihub/wut/main/scripts/install.sh | sh
#
# Env overrides:
#   TH_VERSION=v0.1.2     install a specific tag (default: latest)
#   TH_INSTALL_DIR=/path  override the install location (default: /usr/local/bin
#                          when writable, else $HOME/.local/bin)
#
# This script is POSIX sh and avoids bashisms.

set -eu

OWNER="sonyaihub"
REPO="wut"
BIN="wut"

log()  { printf "==> %s\n" "$*" >&2; }
warn() { printf "!!  %s\n" "$*" >&2; }
die()  { printf "xx  %s\n" "$*" >&2; exit 1; }

need() {
  command -v "$1" >/dev/null 2>&1 || die "required tool not found: $1"
}

# ---- detect platform ------------------------------------------------------

uname_s=$(uname -s)
uname_m=$(uname -m)

case "$uname_s" in
  Darwin) os="Darwin" ;;
  Linux)  os="Linux"  ;;
  *)      die "unsupported OS: $uname_s" ;;
esac

case "$uname_m" in
  x86_64|amd64) arch="x86_64" ;;
  arm64|aarch64) arch="arm64" ;;
  *) die "unsupported arch: $uname_m" ;;
esac

# ---- pick version ---------------------------------------------------------

need curl
version=${TH_VERSION:-}
if [ -z "$version" ]; then
  # Resolve "latest" via GitHub's redirect — no auth, no rate-limit worries.
  log "resolving latest release…"
  version=$(curl -fsSLI -o /dev/null -w '%{url_effective}' \
    "https://github.com/$OWNER/$REPO/releases/latest" \
    | sed 's#.*/tag/##')
  [ -n "$version" ] || die "could not resolve latest release"
fi

# Strip a leading "v" for the tarball version (goreleaser drops it in .Version).
version_no_v=${version#v}

asset="${BIN}_${os}_${arch}.tar.gz"
url="https://github.com/$OWNER/$REPO/releases/download/$version/$asset"

log "installing $BIN $version ($os/$arch)"

# ---- pick install dir -----------------------------------------------------

if [ -n "${TH_INSTALL_DIR:-}" ]; then
  install_dir="$TH_INSTALL_DIR"
  sudo=""
elif [ -w /usr/local/bin ]; then
  install_dir="/usr/local/bin"
  sudo=""
elif command -v sudo >/dev/null 2>&1; then
  install_dir="/usr/local/bin"
  sudo="sudo"
else
  install_dir="$HOME/.local/bin"
  sudo=""
fi
mkdir -p "$install_dir" 2>/dev/null || $sudo mkdir -p "$install_dir"

# ---- download + verify ----------------------------------------------------

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

log "downloading $asset"
if ! curl -fsSL -o "$tmp/$asset" "$url"; then
  warn "release asset not found at $url"
  if command -v go >/dev/null 2>&1; then
    log "falling back to go install from source"
    GOBIN="$tmp" go install "github.com/$OWNER/$REPO/cmd/$BIN@$version"
    $sudo install -m 0755 "$tmp/$BIN" "$install_dir/$BIN"
    log "installed via go install → $install_dir/$BIN"
    version_no_v=""  # let the binary print its own version
  else
    die "download failed and no Go toolchain available for source fallback"
  fi
else
  # Checksum verify.
  sums_url="https://github.com/$OWNER/$REPO/releases/download/$version/checksums.txt"
  if curl -fsSL -o "$tmp/checksums.txt" "$sums_url"; then
    expected=$(grep " $asset\$" "$tmp/checksums.txt" | awk '{print $1}')
    if [ -n "$expected" ]; then
      if command -v sha256sum >/dev/null 2>&1; then
        actual=$(sha256sum "$tmp/$asset" | awk '{print $1}')
      else
        actual=$(shasum -a 256 "$tmp/$asset" | awk '{print $1}')
      fi
      if [ "$expected" != "$actual" ]; then
        die "checksum mismatch for $asset (expected $expected, got $actual)"
      fi
      log "sha256 verified"
    else
      warn "checksums.txt has no entry for $asset — skipping verification"
    fi
  else
    warn "checksums.txt not found — skipping verification"
  fi

  tar -xzf "$tmp/$asset" -C "$tmp"
  $sudo install -m 0755 "$tmp/$BIN" "$install_dir/$BIN"
fi

# ---- confirm install + nudge toward setup --------------------------------

if ! command -v "$BIN" >/dev/null 2>&1; then
  warn "$install_dir is not on your \$PATH."
  warn "add this to your shell rc file:"
  warn "  export PATH=\"$install_dir:\$PATH\""
  warn ""
  warn "then run: $install_dir/$BIN doctor"
  exit 0
fi

installed_version=$("$BIN" version 2>/dev/null || echo "unknown")
log "installed $BIN $installed_version → $install_dir/$BIN"

cat >&2 <<NEXT

Next steps:

  1. Pick a harness and default mode:
       wut setup
  2. Wire the shell hook:
       wut install-hook
  3. Open a new shell, then verify:
       wut doctor

NEXT
