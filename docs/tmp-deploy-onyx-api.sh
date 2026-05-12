#!/usr/bin/env bash
set -euo pipefail

. /root/sub2api-onyx-vars.env

cat >/etc/onyx/onyx.env <<EOF
PYTHONPATH=${ONYX_DIR}/backend
LD_PRELOAD=libjemalloc.so.2

AUTH_TYPE=basic
WEB_DOMAIN=http://127.0.0.1:3000

POSTGRES_HOST=${PG_APP_HOST}
POSTGRES_PORT=${PG_APP_PORT}
POSTGRES_USER=${ONYX_DB_USER}
POSTGRES_PASSWORD=${ONYX_DB_PASSWORD}
POSTGRES_DB=${ONYX_DB}

DISABLE_VECTOR_DB=true
FILE_STORE_BACKEND=postgres
CACHE_BACKEND=postgres
AUTH_BACKEND=postgres

SUB2API_INTEGRATION_ENABLED=true
SUB2API_BASE_URL=http://127.0.0.1:8080
SUB2API_LLM_BASE_URL=http://127.0.0.1:8080/v1
SUB2API_EXCHANGE_SECRET=${SUB2API_ONYX_SECRET}
SUB2API_DEFAULT_TEXT_MODEL=gpt-5.5
SUB2API_DEFAULT_IMAGE_MODEL=gpt-image-2
SUB2API_ONYX_REDIRECT_PATH=/chat

API_SERVER_PROTOCOL=http
API_SERVER_HOST=127.0.0.1
API_SERVER_PORT=8081
EOF

chmod 640 /etc/onyx/onyx.env
chown root:onyx /etc/onyx/onyx.env

cd "${ONYX_DIR}/backend"
set -a
. /etc/onyx/onyx.env
set +a
"${ONYX_DIR}/.venv/bin/alembic" -c alembic.ini upgrade head

cat >/etc/systemd/system/onyx-api.service <<EOF
[Unit]
Description=Onyx API Lite
After=network-online.target postgresql.service
Wants=network-online.target postgresql.service

[Service]
Type=simple
User=onyx
Group=onyx
WorkingDirectory=${ONYX_DIR}/backend
EnvironmentFile=/etc/onyx/onyx.env
ExecStartPre=${ONYX_DIR}/.venv/bin/alembic -c ${ONYX_DIR}/backend/alembic.ini upgrade head
ExecStart=${ONYX_DIR}/.venv/bin/uvicorn onyx.main:app --host 127.0.0.1 --port 8081
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=onyx-api
NoNewPrivileges=true
PrivateTmp=true
ReadWritePaths=/var/log/onyx ${ONYX_DIR}/backend

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable --now onyx-api
systemctl status onyx-api --no-pager
