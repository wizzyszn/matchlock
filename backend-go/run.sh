#!/usr/bin/env bash
set -a
. "$(dirname "$0")/.env"
set +a
exec "$(dirname "$0")/bin/keeper"
