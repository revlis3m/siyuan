#!/usr/bin/env bash
# Runs the mods checks against a throwaway SiYuan started from the "mods-preview" build
# (the server + editor that the workflow "Mods preview bundle" publishes on every push to mods).
#   ./run.sh            download the current preview build, start it, run api.mjs then ui.mjs
#   ./run.sh api        only the API checks        ./run.sh ui     only the browser check
#   KEEP=1 ./run.sh     leave the test SiYuan running afterwards (http://127.0.0.1:6816, code test1234)
# Needs: gh (logged in, or GH_TOKEN), node 22+, the "playwright" package and Chrome.
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
work="${MODS_TESTS_DIR:-${TMPDIR:-/tmp}/mods-tests}"
port="${PORT:-6816}"
repo="${MODS_REPO:-revlis3m/siyuan}"
which="${1:-all}"

mkdir -p "$work/bundle"
asset=$(gh api "repos/$repo/releases/tags/mods-preview" -q '.assets[] | select(.name=="mods-preview-linux-amd64.tar.gz") | "\(.id) \(.updated_at)"')
id="${asset%% *}"
if [ "$(cat "$work/bundle.version" 2>/dev/null)" != "$asset" ]; then
    echo "downloading the preview build (${asset#* })"
    gh api -H "Accept: application/octet-stream" "repos/$repo/releases/assets/$id" > "$work/bundle.tar.gz"
    rm -rf "$work/bundle" && mkdir -p "$work/bundle"
    tar xzf "$work/bundle.tar.gz" -C "$work/bundle"
    echo "$asset" > "$work/bundle.version"
fi

rm -rf "$work/ws" "$work/home" && mkdir -p "$work/ws" "$work/home"
"$work/bundle/kernel" serve --wd "$work/bundle" --workspace "$work/ws" --home-dir "$work/home" \
    --port "$port" --accessAuthCode test1234 --lang en > "$work/kernel.log" 2>&1 &
kernel=$!
stop() { [ -n "${KEEP:-}" ] || kill "$kernel" 2>/dev/null || true; }
trap stop EXIT
for _ in $(seq 1 60); do
    curl -fsS -m 2 -X POST "http://127.0.0.1:$port/api/system/version" >/dev/null 2>&1 && break
    sleep 1
done
curl -fsS -m 2 -X POST "http://127.0.0.1:$port/api/system/version" >/dev/null || { echo "the test SiYuan did not start; see $work/kernel.log"; exit 1; }

export SIYUAN_URL="http://127.0.0.1:$port" SIYUAN_WORKSPACE="$work/ws" SIYUAN_ACCESS_CODE=test1234
status=0
if [ "$which" = all ] || [ "$which" = api ]; then node "$here/api.mjs" || status=1; fi
if [ "$which" = all ] || [ "$which" = ui ]; then node "$here/ui.mjs" || status=1; fi
exit $status
