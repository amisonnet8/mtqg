#!/usr/bin/env bash
# Apply one mutation to a file, run a test command, say whether the tests noticed,
# and put the file back. See SKILL.md.
#
# usage: mutate.sh <name> <file> <old text> <new text> -- <test command>...
#
# Exit code: 0 the tests failed (the mutation was killed), 1 the tests passed (it
# survived), 2 a mistake in the use, or the tests fail before the mutation, 3 the
# mutation did not compile (it says nothing about the tests).
# MUTATE_BASELINE=0 skips the run of the tests before the mutation.
set -uo pipefail

usage() {
  echo "usage: mutate.sh <name> <file> <old text> <new text> -- <test command>..." >&2
  exit 2
}

[[ $# -ge 6 ]] || usage
name=$1
file=$2
old=$3
new=$4
shift 4
[[ $1 == -- ]] || usage
shift

[[ -f $file ]] || { echo "error: no such file: $file" >&2; exit 2; }
[[ -n $old ]] || { echo "error: the text to replace is empty" >&2; exit 2; }
[[ $old != "$new" ]] || { echo "error: the mutation changes nothing" >&2; exit 2; }

# The file is kept aside and put back however this ends (a failed test, a signal).
# The copy is removed once the file is known to be the same as it was.
backup=$(mktemp)
cp -p "$file" "$backup"
finish() {
  cp -p "$backup" "$file"
  if cmp -s "$backup" "$file"; then
    rm -f "$backup"
  else
    echo "error: $file was not put back; the original is in $backup" >&2
    return 1
  fi
}
trap finish EXIT
trap 'exit 130' INT TERM

# The text of the file with its last line feeds, which $(...) would drop.
content=$(cat "$file"; printf x)
content=${content%x}
[[ $content == *"$old"* ]] || { echo "error: the text to replace is not in $file" >&2; exit 2; }

if [[ ${MUTATE_BASELINE:-1} != 0 ]] && ! "$@" >/dev/null 2>&1; then
  echo "error: the tests fail before the mutation, so nothing they say means anything" >&2
  exit 2
fi

printf '%s' "${content/"$old"/"$new"}" >"$file"
out=$("$@" 2>&1)
code=$?
trap - EXIT
finish || exit 2

if [[ $code -eq 0 ]]; then
  echo "SURVIVED: $name"
  exit 1
fi
if grep -qE 'build failed|setup failed' <<<"$out"; then
  echo "not compiled: $name (change the mutation: this says nothing about the tests)"
  grep -m3 -E '\.go:[0-9]+' <<<"$out" | sed 's/^/  /'
  exit 3
fi
echo "killed:   $name ($(grep -c -e '--- FAIL' <<<"$out") failing tests)"
exit 0
