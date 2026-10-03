#!/usr/bin/env bash
# Applies every migration up, rolls every one back, then applies them up
# again — all inside a throwaway schema, so the real data is never touched.
# Catches a missing/broken .down.sql and migrations that can't be re-run.
#
#   TEST_DATABASE_URL=postgres://… scripts/test-migrations.sh
set -euo pipefail

: "${TEST_DATABASE_URL:?set TEST_DATABASE_URL}"
cd "$(dirname "$0")/.."

schema="it_mig_$(date +%s)_$$"
psql() { command psql "$TEST_DATABASE_URL" -q -X -v ON_ERROR_STOP=1 "$@"; }
cleanup() { PGOPTIONS= psql -c "DROP SCHEMA IF EXISTS $schema CASCADE" >/dev/null 2>&1 || true; }
trap cleanup EXIT

psql -c "CREATE SCHEMA $schema" >/dev/null
export PGOPTIONS="-c search_path=$schema,public -c client_min_messages=warning"

tables() { psql -At -c "SELECT count(*) FROM information_schema.tables WHERE table_schema = '$schema'"; }

run() { # run <label> <files...>
	local label=$1; shift
	for f in "$@"; do
		if ! psql -f "$f" >/dev/null; then
			echo "FAIL ($label): $f" >&2
			exit 1
		fi
	done
}

ups=$(ls migrations/*.up.sql | sort -V)
downs=$(ls migrations/*.down.sql | sort -Vr)

# Rollbacks run inside a throwaway schema, but extensions, other schemas and
# the database are shared by everyone — a down file must never drop those.
if grep -niE '^\s*DROP\s+(EXTENSION|SCHEMA|DATABASE)' migrations/*.down.sql; then
	echo "FAIL: a down migration drops a database-wide object (see above)" >&2
	exit 1
fi

for up in $ups; do
	[[ -f "${up%.up.sql}.down.sql" ]] || { echo "FAIL: $up has no .down.sql" >&2; exit 1; }
done

run up $ups
after_up=$(tables)
run down $downs
after_down=$(tables)
[[ "$after_down" == 0 ]] || { echo "FAIL: $after_down tables left after rolling everything back" >&2; exit 1; }
run re-up $ups
[[ "$(tables)" == "$after_up" ]] || { echo "FAIL: re-applying gave $(tables) tables, first run gave $after_up" >&2; exit 1; }

echo "migrations OK: $(echo "$ups" | wc -l) up, all rolled back cleanly, re-applied ($after_up tables)"
