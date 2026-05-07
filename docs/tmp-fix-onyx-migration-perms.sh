#!/usr/bin/env bash
set -euo pipefail

. /root/sub2api-onyx-vars.env

sudo -u postgres psql <<EOF
ALTER ROLE ${ONYX_DB_USER} CREATEROLE;
EOF

cd "${ONYX_DIR}/backend"
set -a
. /etc/onyx/onyx.env
set +a
"${ONYX_DIR}/.venv/bin/alembic" -c alembic.ini upgrade head

sudo -u postgres psql <<EOF
ALTER ROLE ${ONYX_DB_USER} NOCREATEROLE;
EOF
