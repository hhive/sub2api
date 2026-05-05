#!/usr/bin/env bash
set -euo pipefail

. /root/sub2api-onyx-vars.env

install -m 0755 "/app/ai/sub2api-all/sub2api/backend/sub2api" "${SUB2API_APP_DIR}/sub2api"
chown sub2api:sub2api "${SUB2API_APP_DIR}/sub2api"
chown -R sub2api:sub2api /etc/sub2api

cat >/etc/sub2api/sub2api.env <<EOF
DATA_DIR=/etc/sub2api
AUTO_SETUP=true
DATABASE_HOST=${PG_APP_HOST}
DATABASE_PORT=${PG_APP_PORT}
DATABASE_USER=${SUB2API_DB_USER}
DATABASE_PASSWORD=${SUB2API_DB_PASSWORD}
DATABASE_DBNAME=${SUB2API_DB}
DATABASE_SSLMODE=${PG_SSLMODE}
REDIS_HOST=127.0.0.1
REDIS_PORT=6379
REDIS_PASSWORD=${REDIS_PASSWORD}
REDIS_DB=0
ADMIN_EMAIL=${SUB2API_ADMIN_EMAIL}
ADMIN_PASSWORD=${SUB2API_ADMIN_PASSWORD}
SERVER_HOST=127.0.0.1
SERVER_PORT=8080
SERVER_MODE=release
JWT_SECRET=${SUB2API_JWT_SECRET}
GIN_MODE=release
EOF
chmod 640 /etc/sub2api/sub2api.env
chown sub2api:sub2api /etc/sub2api/sub2api.env

cat >/etc/systemd/system/sub2api.service <<'EOF'
[Unit]
Description=Sub2API
After=network-online.target postgresql.service redis-server.service
Wants=network-online.target postgresql.service redis-server.service

[Service]
Type=simple
User=sub2api
Group=sub2api
WorkingDirectory=/opt/sub2api
EnvironmentFile=/etc/sub2api/sub2api.env
ExecStart=/opt/sub2api/sub2api
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=sub2api
NoNewPrivileges=true
PrivateTmp=true
ProtectHome=true
ReadWritePaths=/opt/sub2api /var/log/sub2api /etc/sub2api

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable --now sub2api
systemctl status sub2api --no-pager
