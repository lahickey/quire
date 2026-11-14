#!/bin/sh
# Retakes the screenshots in content/docs/shots from the examples. Needs chromium.
set -eu
site=$(cd "$(dirname "$0")" && pwd)
root=$site/..
tmp=$(mktemp -d)
trap 'kill $pids 2>/dev/null; rm -rf "$tmp"' EXIT
pids=

serve() {
	(cd "$root/examples/$1" && go build -o "$tmp/$1" . && exec "$tmp/$1" --drafts --addr "127.0.0.1:$2") >"$tmp/$1.log" 2>&1 &
	pids="$pids $!"
	until curl -sf "http://127.0.0.1:$2/_quire/health" >/dev/null; do sleep 0.2; done
}

shot() {
	chromium --headless --disable-gpu --hide-scrollbars --window-size="1200,$3" --screenshot="$tmp/$1.png" "http://127.0.0.1:$2" >/dev/null 2>&1
	(cd "$root" && go run ./cmd/quire ingest "$site/content/docs/shots" "$1" "$tmp/$1.png")
}

serve 2-pictures 8111
serve 3-themed 8112
shot listing 8111/posts 520
shot article 8111/posts/ridge 1000
shot tagged 8112/writing/one-binary 1000
shot badges 8112/ 900
shot draft 8112/writing/unfinished 620
