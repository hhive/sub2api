#!/usr/bin/env bash
set -euo pipefail

VARS_FILE="/root/sub2api-onyx-vars.env"

if [ ! -f "$VARS_FILE" ]; then
  cat >"$VARS_FILE" <<EOF
export SUB2API_DOMAIN="sub2api.localhost"
export ONYX_DOMAIN="onyx.localhost"

export SUB2API_SRC_DIR="/app/ai/sub2api-all/sub2api"
export ONYX_DIR="/app/ai/sub2api-all/onyx"
export SUB2API_APP_DIR="/opt/sub2api"

export SUB2API_DB="sub2api"
export SUB2API_DB_USER="sub2api"
export SUB2API_DB_PASSWORD="$(openssl rand -hex 16)"

export ONYX_DB="onyx"
export ONYX_DB_USER="onyx"
export ONYX_DB_PASSWORD="$(openssl rand -hex 16)"

export PG_APP_HOST="127.0.0.1"
export PG_APP_PORT="5432"
export PG_SSLMODE="disable"

export REDIS_PASSWORD="$(openssl rand -hex 16)"
export SUB2API_ADMIN_EMAIL="admin@example.com"
export SUB2API_ADMIN_PASSWORD="$(openssl rand -base64 24 | tr -d '\n' | tr '/+' 'AB' | cut -c1-24)"
export SUB2API_JWT_SECRET="$(openssl rand -hex 32)"
export SUB2API_ONYX_SECRET="$(openssl rand -hex 32)"
EOF
  chmod 600 "$VARS_FILE"
fi

. "$VARS_FILE"

systemctl enable --now postgresql
systemctl enable --now redis-server

if ! sudo -u postgres psql -tAc "SELECT 1 FROM pg_roles WHERE rolname='${SUB2API_DB_USER}'" | grep -q 1; then
  sudo -u postgres psql -c "CREATE USER ${SUB2API_DB_USER} WITH PASSWORD '${SUB2API_DB_PASSWORD}';"
else
  sudo -u postgres psql -c "ALTER USER ${SUB2API_DB_USER} WITH PASSWORD '${SUB2API_DB_PASSWORD}';"
fi

if ! sudo -u postgres psql -tAc "SELECT 1 FROM pg_database WHERE datname='${SUB2API_DB}'" | grep -q 1; then
  sudo -u postgres psql -c "CREATE DATABASE ${SUB2API_DB} OWNER ${SUB2API_DB_USER};"
fi

if ! sudo -u postgres psql -tAc "SELECT 1 FROM pg_roles WHERE rolname='${ONYX_DB_USER}'" | grep -q 1; then
  sudo -u postgres psql -c "CREATE USER ${ONYX_DB_USER} WITH PASSWORD '${ONYX_DB_PASSWORD}';"
else
  sudo -u postgres psql -c "ALTER USER ${ONYX_DB_USER} WITH PASSWORD '${ONYX_DB_PASSWORD}';"
fi

if ! sudo -u postgres psql -tAc "SELECT 1 FROM pg_database WHERE datname='${ONYX_DB}'" | grep -q 1; then
  sudo -u postgres psql -c "CREATE DATABASE ${ONYX_DB} OWNER ${ONYX_DB_USER};"
fi

cp /etc/redis/redis.conf "/etc/redis/redis.conf.bak.$(date +%F-%H%M%S)"
sed -i "s/^#\? requirepass .*/requirepass ${REDIS_PASSWORD}/" /etc/redis/redis.conf
if ! grep -q '^requirepass ' /etc/redis/redis.conf; then
  echo "requirepass ${REDIS_PASSWORD}" >> /etc/redis/redis.conf
fi
systemctl restart redis-server

install -d -m 750 /etc/sub2api /etc/onyx
install -d -m 755 /opt/sub2api /var/log/sub2api /var/log/onyx
id -u sub2api >/dev/null 2>&1 || useradd --system --home-dir /opt/sub2api --shell /usr/sbin/nologin sub2api
id -u onyx >/dev/null 2>&1 || useradd --system --home-dir /opt/onyx --shell /usr/sbin/nologin onyx
chown -R sub2api:sub2api /opt/sub2api /var/log/sub2api
chown -R onyx:onyx /var/log/onyx

echo "Variables saved to $VARS_FILE"
echo "Admin email: ${SUB2API_ADMIN_EMAIL}"
echo "Admin password: ${SUB2API_ADMIN_PASSWORD}"
