#!/bin/sh
set -e

# Render assigns PORT dynamically (defaults to 10000 if unset)
export PORT="${PORT:-10000}"

envsubst '${PORT}' < /etc/nginx/templates/default.conf.template > /etc/nginx/http.d/default.conf

exec supervisord -c /etc/supervisord.conf
