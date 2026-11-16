#!/usr/bin/env bash
# backup.sh - snapshot a directory to a dated archive and prune old ones.
set -euo pipefail
IFS=$'\n\t'

readonly PROG="${0##*/}"
readonly DEFAULT_KEEP=7
declare -a EXCLUDES=(".git" "node_modules" "*.tmp")
declare -A STATS=([files]=0 [bytes]=0)
VERBOSE=0
DRY_RUN=false

usage() {
  cat <<EOF
Usage: $PROG [-k DAYS] [-n] [-v] SOURCE DEST

  -k DAYS   keep archives newer than DAYS (default $DEFAULT_KEEP)
  -n        dry run: say what would happen
  -v        say more
EOF
}

# Usage and logging.
log() { printf '%s %s\n' "$(date +%H:%M:%S)" "$*" >&2; }
debug() { (( VERBOSE )) && log "debug: $*" || true; }
die() { log "error: $*"; exit 1; }

cleanup() {
  local status=$?
  [[ -n "${tmp:-}" && -d "$tmp" ]] && rm -rf -- "$tmp"
  (( status == 0 )) || log "failed with status $status"
}
trap cleanup EXIT
trap 'die "interrupted"' INT TERM

human() {
  local bytes=$1 unit
  for unit in B K M G; do
    if (( bytes < 1024 )); then
      echo "${bytes}${unit}"
      return
    fi
    bytes=$(( bytes / 1024 ))
  done
  echo "${bytes}T"
}

keep=$DEFAULT_KEEP
while getopts ':k:nvh' opt; do
  case "$opt" in
    k) keep=$OPTARG ;;
    n) DRY_RUN=true ;;
    v) VERBOSE=$(( VERBOSE + 1 )) ;;
    h) usage; exit 0 ;;
    :) die "-$OPTARG needs a value" ;;
    \?) usage >&2; exit 2 ;;
  esac
done
shift $(( OPTIND - 1 ))

[[ $# -eq 2 ]] || { usage >&2; exit 2; }
[[ "$keep" =~ ^[0-9]+$ ]] || die "not a number of days: $keep"

src=$(realpath -- "$1")
dest=$2
[[ -d "$src" ]] || die "no such directory: $src"
mkdir -p -- "$dest"

stamp=$(date +%Y-%m-%d_%H%M%S)
archive="$dest/$(basename "$src")-$stamp.tar.gz"
tmp=$(mktemp -d "${TMPDIR:-/tmp}/$PROG.XXXXXX")

exclude_args=()
for pattern in "${EXCLUDES[@]}"; do
  exclude_args+=("--exclude=$pattern")
done

while IFS= read -r -d '' file; do
  STATS[files]=$(( STATS[files] + 1 ))
  size=$(stat -c %s -- "$file" 2>/dev/null || echo 0)
  STATS[bytes]=$(( STATS[bytes] + size ))
done < <(find "$src" -type f -not -path '*/.git/*' -print0)

log "archiving ${STATS[files]} files, $(human "${STATS[bytes]}")"

if $DRY_RUN; then
  log "would write $archive"
else
  tar -czf "$tmp/out.tar.gz" "${exclude_args[@]}" -C "$(dirname "$src")" "$(basename "$src")"
  mv -- "$tmp/out.tar.gz" "$archive"
  sha256sum "$archive" | awk '{ print $1 }' > "$archive.sha256"
  debug "wrote $(du -h "$archive" | cut -f1)"
fi

cat > "$tmp/policy" <<EOF
keep $1
EOF

pruned=0
for old in "$dest"/*.tar.gz; do
  [[ -e "$old" ]] || continue
  if [[ $(find "$old" -mtime +"$keep" -print) ]]; then
    if ! $DRY_RUN; then
      rm -f -- "$old" "$old.sha256"
    fi
    pruned=$(( pruned + 1 ))
  fi
done

until [[ $pruned -le 0 ]]; do
  debug "pruned one"
  pruned=$(( pruned - 1 ))
done

echo "done: ${archive##*/}" | tr '[:lower:]' '[:upper:]'
