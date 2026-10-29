#!/usr/bin/env bash
set -euo pipefail

usage() {
	echo "usage: $0 <base-url> <output-dir>" >&2
	echo "example: $0 http://127.0.0.1:8212 /tmp/palrest-capture" >&2
}

if [[ $# -ne 2 ]]; then
	usage
	exit 1
fi

base="${1%/}"
out="$2"
mkdir -p "$out"

read -rsp "Palworld admin password: " password
printf '\n'

for endpoint in info players settings metrics game-data; do
	curl -fsS --user "admin:$password" "$base/v1/api/$endpoint" -o "$out/$endpoint.json"
	printf 'captured %s\n' "$endpoint"
done

printf 'Fixtures written to %s\n' "$out"
printf 'Review them before copying into testdata/contract/<palworld-version>/.\n'
