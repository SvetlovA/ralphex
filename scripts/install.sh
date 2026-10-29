#!/bin/sh
# Install the latest fork release on Linux or macOS, without requiring Go or root.
set -eu

fail() {
    printf 'ralphex installer: %s\n' "$*" >&2
    exit 1
}

for tool in curl tar awk install sed grep mktemp mkdir rm; do
    command -v "$tool" >/dev/null 2>&1 || fail "required command not found: $tool"
done

case "$(uname -s)" in
    Linux) os=linux ;;
    Darwin) os=darwin ;;
    *) fail "unsupported OS: $(uname -s)" ;;
esac
case "$(uname -m)" in
    x86_64|amd64) arch=amd64 ;;
    arm64|aarch64) arch=arm64 ;;
    *) fail "unsupported architecture: $(uname -m)" ;;
esac

if command -v sha256sum >/dev/null 2>&1; then
    checksum() { sha256sum "$1" | awk '{print $1}'; }
elif command -v shasum >/dev/null 2>&1; then
    checksum() { shasum -a 256 "$1" | awk '{print $1}'; }
else
    fail 'sha256sum or shasum is required'
fi

repository=https://github.com/svtlv/ralphex
tag=${RALPHEX_VERSION:-}
if [ -z "$tag" ]; then
    release=$(curl -fsSL --retry 3 https://api.github.com/repos/svtlv/ralphex/releases/latest)
    tag=$(printf '%s\n' "$release" | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')
fi
printf '%s\n' "$tag" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+-windows\.[1-9][0-9]*$' || fail "invalid fork release tag: $tag"
version=${tag#v}
archive=ralphex_${version}_${os}_${arch}.tar.gz
install_dir=${RALPHEX_INSTALL_DIR:-"$HOME/.local/bin"}

temporary_dir=$(mktemp -d)
trap 'rm -rf "$temporary_dir"' EXIT
trap 'exit 1' HUP INT TERM

printf 'Downloading ralphex %s for %s/%s...\n' "$version" "$os" "$arch"
curl -fsSL --retry 3 "$repository/releases/download/$tag/$archive" -o "$temporary_dir/$archive"
curl -fsSL --retry 3 "$repository/releases/download/$tag/ralphex_${version}_checksums.txt" -o "$temporary_dir/checksums.txt"
expected=$(awk -v asset="$archive" '$2 == asset {print $1}' "$temporary_dir/checksums.txt")
printf '%s\n' "$expected" | grep -Eq '^[a-fA-F0-9]{64}$' || fail "missing or invalid checksum for $archive"
actual=$(checksum "$temporary_dir/$archive")
[ "$actual" = "$expected" ] || fail "checksum mismatch for $archive"

tar -xzf "$temporary_dir/$archive" -C "$temporary_dir" ralphex
[ -f "$temporary_dir/ralphex" ] && [ ! -L "$temporary_dir/ralphex" ] || fail 'archive does not contain a regular ralphex binary'
mkdir -p "$install_dir"
install -m 0755 "$temporary_dir/ralphex" "$install_dir/ralphex"
printf 'Installed %s to %s/ralphex\n' "$tag" "$install_dir"
case ":$PATH:" in
    *":$install_dir:"*) printf 'Verify with: ralphex --version\n' ;;
    *) printf 'Add %s to your shell PATH, then run: ralphex --version\n' "$install_dir" ;;
esac
