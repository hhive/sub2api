# Ubuntu 24.04 裸机部署 Sub2API + Onyx Lite 指南

> 目标：在一台全新的 Ubuntu 24.04 服务器上，不使用 Docker，部署 Sub2API 与 Onyx Lite，并通过 Nginx 统一反向代理。

## 目录

0. 给服务器执行 agent 的执行方式
1. 部署目标与拓扑
2. 约定变量
3. 系统依赖
4. 中间件初始化
5. 部署 Sub2API
6. 部署 Onyx Lite
7. 配置两边集成
8. 配置 Nginx
9. 启动与验证
10. 运维命令
11. 常见故障

---

## 0. 给服务器执行 agent 的执行方式

服务器上的大模型或运维 agent 按阶段执行，不要一次性全量粘贴整篇文档：

1. 先执行第 2 节，替换域名、密码和密钥变量。
2. 执行第 3 到第 4 节，完成系统依赖、PostgreSQL、Redis。
3. 执行第 5 节，只部署 Sub2API。第 5.7 节验证通过后再继续。
4. 执行第 6 节，只部署 Onyx Lite。第 6.8 节验证通过后再继续。
5. 执行第 7 节，配置 Sub2API 与 Onyx 的 shared secret 和 URL。
6. 执行第 8 节，配置 Nginx 和 HTTPS。
7. 执行第 9 节，完成端到端验证。

每个阶段的停止条件：

- 任一 `systemctl status` 显示服务失败时，先看该服务日志，不继续下一阶段。
- 任一 `curl -i` 本机端口验证失败时，先修复本机服务，不配置 Nginx。
- launch/exchange 未返回 `302 /chat` 和 `fastapiusersauth` cookie 前，不继续验证聊天。
- 不执行任何 Docker 安装、Docker Compose、`docker run` 或 `docker compose` 命令。

---

## 1. 部署目标与拓扑

本指南面向一台全新的 Ubuntu 24.04 服务器，要求：

- 不使用 Docker / Docker Compose。
- Sub2API 与 Onyx Lite 部署在同一台机器。
- 尽量复用中间件：
  - 复用同一个 PostgreSQL 实例，分别创建 `sub2api` 与 `onyx` 数据库。
  - Redis 仅供 Sub2API 使用。
  - Onyx Lite 使用 PostgreSQL 作为 cache/auth/file store，不启动 Redis、Vespa、OpenSearch、MinIO、model server、background worker。
- Nginx 对外提供 HTTPS 和反向代理。

推荐本机端口规划：

| 服务 | 监听地址 | 用途 |
| --- | --- | --- |
| Sub2API | `127.0.0.1:8080` | Sub2API 后端与嵌入前端 |
| Onyx API | `127.0.0.1:8081` | Onyx FastAPI 后端 |
| Onyx Web | `127.0.0.1:3000` | Onyx Next.js 前端 |
| PostgreSQL | `127.0.0.1:5432` | 两个服务共用实例，分库分用户 |
| Redis | `127.0.0.1:6379` | Sub2API token/cache |
| Nginx | `0.0.0.0:80/443` | 对外入口 |

跨服务器使用 PostgreSQL 时，推荐让 PostgreSQL 服务器继续只监听 `127.0.0.1:5432`，在应用服务器上用 SSH tunnel 映射成本机端口：

```text
Sub2API/Onyx 应用服务器 127.0.0.1:15432
  -> SSH tunnel
PostgreSQL 服务器 127.0.0.1:5432
```

应用服务仍然连接 `127.0.0.1`，但端口改为 tunnel 本地端口。不要为了跨服务器访问而直接把 PostgreSQL 暴露到公网。

推荐域名：

| 域名 | Nginx 反代目标 |
| --- | --- |
| `sub2api.example.com` | `http://127.0.0.1:8080` |
| `onyx.example.com` | `/api/*` 到 `http://127.0.0.1:8081`，其余到 `http://127.0.0.1:3000` |

Onyx 的官方 Docker nginx 会把 `/api/*` 转发到后端时去掉 `/api` 前缀。裸机 Nginx 也必须保持这个行为。

---

## 2. 约定变量

执行前先替换这些变量。建议服务器上的大模型或运维 agent 先生成一份自己的变量表，再逐条执行命令。

```bash
export SUB2API_DOMAIN="sub2api.example.com"
export ONYX_DOMAIN="onyx.example.com"

export SUB2API_REPO="https://github.com/Wei-Shaw/sub2api.git"
export ONYX_REPO="https://github.com/onyx-dot-app/onyx.git"

export SRC_ROOT="/opt/src"
export SUB2API_SRC_DIR="/opt/src/sub2api"
export ONYX_DIR="/opt/onyx"
export SUB2API_APP_DIR="/opt/sub2api"

export SUB2API_DB="sub2api"
export SUB2API_DB_USER="sub2api"
export SUB2API_DB_PASSWORD="replace-with-strong-sub2api-db-password"

export ONYX_DB="onyx"
export ONYX_DB_USER="onyx"
export ONYX_DB_PASSWORD="replace-with-strong-onyx-db-password"

# 单机部署使用 127.0.0.1:5432。
# 跨服务器 PostgreSQL 使用 SSH tunnel 时，保持 host 为 127.0.0.1，端口改为本机转发端口。
export PG_APP_HOST="127.0.0.1"
export PG_APP_PORT="5432"
export PG_SSLMODE="disable"

# 跨服务器 PostgreSQL 可选变量。仅在第 4.2 节启用 SSH tunnel 时使用。
export PG_TUNNEL_LOCAL_PORT="15432"
export PG_REMOTE_SSH_HOST="108.187.32.84"
export PG_REMOTE_SSH_USER="root"
export PG_REMOTE_DB_HOST="127.0.0.1"
export PG_REMOTE_DB_PORT="5432"

export REDIS_PASSWORD="replace-with-strong-redis-password"
export SUB2API_ADMIN_EMAIL="admin@example.com"
export SUB2API_ADMIN_PASSWORD="replace-with-strong-admin-password"
export SUB2API_JWT_SECRET="replace-with-64-random-chars"
export SUB2API_ONYX_SECRET="replace-with-64-random-chars"
```

生成随机密钥示例：

```bash
openssl rand -hex 32
```

注意：

- Onyx 会拒绝 `.local` 等保留域邮箱，管理员邮箱不要使用 `admin@sub2api.local`。
- 本文示例使用 `example.com` 占位，生产环境必须替换成真实域名。

---

## 3. 系统依赖

### 3.1 安装基础包

```bash
sudo apt update
sudo apt install -y \
  git curl wget ca-certificates gnupg lsb-release build-essential pkg-config cmake \
  nginx certbot python3-certbot-nginx jq \
  postgresql postgresql-contrib redis-server \
  libpq-dev libxmlsec1-dev libxmlsec1-openssl libjemalloc2 \
  python3.12 python3.12-venv python3.12-dev
```

如果 PostgreSQL 已部署在另一台服务器，应用服务器不需要运行本机 PostgreSQL 服务，但仍建议安装 `postgresql-client` 或保留上面的 `postgresql` 包用于 `psql`、`pg_isready` 验证。第 4.1 节的建库命令应在 PostgreSQL 服务器上执行。

### 3.2 安装 Node.js 24

Onyx Web 当前使用 Node 24 依赖；Sub2API 前端也可复用这个 Node。

```bash
curl -fsSL https://deb.nodesource.com/setup_24.x | sudo -E bash -
sudo apt install -y nodejs
node -v
npm -v
```

### 3.3 安装 pnpm

```bash
sudo npm install -g pnpm
pnpm -v
```

### 3.4 安装 Go

Sub2API `go.mod` 当前声明 `go 1.26.2`。如果 Ubuntu apt 源中的 Go 版本低于该版本，请安装官方 tarball。

```bash
go version || true
```

如果版本不足：

```bash
cd /tmp
wget https://go.dev/dl/go1.26.2.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.26.2.linux-amd64.tar.gz
echo 'export PATH=/usr/local/go/bin:$PATH' | sudo tee /etc/profile.d/go.sh
source /etc/profile.d/go.sh
go version
```

### 3.5 安装 uv

Onyx Python 依赖建议用 `uv` 安装。

```bash
curl -LsSf https://astral.sh/uv/install.sh | sh
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
export PATH="$HOME/.local/bin:$PATH"
uv --version
```

---

## 4. 中间件初始化

### 4.1 PostgreSQL 创建两个数据库

单机部署时，在当前应用服务器执行本节。跨服务器 PostgreSQL 时，在 PostgreSQL 服务器执行本节，应用服务器只执行第 4.2 节创建 SSH tunnel。

```bash
sudo systemctl enable --now postgresql
sudo -u postgres psql <<SQL
CREATE USER ${SUB2API_DB_USER} WITH PASSWORD '${SUB2API_DB_PASSWORD}';
CREATE DATABASE ${SUB2API_DB} OWNER ${SUB2API_DB_USER};
CREATE USER ${ONYX_DB_USER} WITH PASSWORD '${ONYX_DB_PASSWORD}';
CREATE DATABASE ${ONYX_DB} OWNER ${ONYX_DB_USER};
SQL
```

验证：

```bash
PGPASSWORD="${SUB2API_DB_PASSWORD}" psql -h 127.0.0.1 -U "${SUB2API_DB_USER}" -d "${SUB2API_DB}" -c "select current_database(), current_user;"
PGPASSWORD="${ONYX_DB_PASSWORD}" psql -h 127.0.0.1 -U "${ONYX_DB_USER}" -d "${ONYX_DB}" -c "select current_database(), current_user;"
```

### 4.2 可选：跨服务器 PostgreSQL SSH tunnel

如果 PostgreSQL 不在当前应用服务器上，且远端 PostgreSQL 只监听 `127.0.0.1:5432`，不要开放 PostgreSQL 公网端口。建议在应用服务器上创建一个长期运行的 SSH tunnel，把远端数据库映射到本机端口。

启用该模式前，调整变量：

```bash
export PG_APP_HOST="127.0.0.1"
export PG_APP_PORT="${PG_TUNNEL_LOCAL_PORT}"
export PG_SSLMODE="require"
```

在应用服务器上生成 tunnel 专用 SSH key：

```bash
sudo install -d -m 700 /etc/sub2api
sudo ssh-keygen -t ed25519 -f /etc/sub2api/pgsql-tunnel_ed25519 -N "" -C "sub2api-pgsql-tunnel"
sudo chmod 600 /etc/sub2api/pgsql-tunnel_ed25519
sudo cat /etc/sub2api/pgsql-tunnel_ed25519.pub
```

把上一步输出的公钥加入 PostgreSQL 服务器 `${PG_REMOTE_SSH_USER}` 用户的 `~/.ssh/authorized_keys`。确认应用服务器可以免密登录：

```bash
sudo ssh -i /etc/sub2api/pgsql-tunnel_ed25519 \
  -o BatchMode=yes \
  -o StrictHostKeyChecking=accept-new \
  "${PG_REMOTE_SSH_USER}@${PG_REMOTE_SSH_HOST}" \
  "pg_isready -h ${PG_REMOTE_DB_HOST} -p ${PG_REMOTE_DB_PORT}"
```

创建 SSH tunnel 服务：

```bash
sudo tee /etc/systemd/system/pgsql-ssh-tunnel.service >/dev/null <<EOF
[Unit]
Description=PostgreSQL SSH Tunnel
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/bin/ssh \
  -N \
  -i /etc/sub2api/pgsql-tunnel_ed25519 \
  -o BatchMode=yes \
  -o ExitOnForwardFailure=yes \
  -o ServerAliveInterval=30 \
  -o ServerAliveCountMax=3 \
  -o StrictHostKeyChecking=accept-new \
  -L 127.0.0.1:${PG_TUNNEL_LOCAL_PORT}:${PG_REMOTE_DB_HOST}:${PG_REMOTE_DB_PORT} \
  ${PG_REMOTE_SSH_USER}@${PG_REMOTE_SSH_HOST}
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable --now pgsql-ssh-tunnel
sudo systemctl status pgsql-ssh-tunnel --no-pager
```

验证应用服务器本机 tunnel 端口：

```bash
ss -lntp | grep ":${PG_TUNNEL_LOCAL_PORT}\b"
PGPASSWORD="${SUB2API_DB_PASSWORD}" psql \
  "host=${PG_APP_HOST} port=${PG_APP_PORT} user=${SUB2API_DB_USER} dbname=${SUB2API_DB} sslmode=${PG_SSLMODE}" \
  -c "select current_database(), current_user;"
PGPASSWORD="${ONYX_DB_PASSWORD}" psql \
  "host=${PG_APP_HOST} port=${PG_APP_PORT} user=${ONYX_DB_USER} dbname=${ONYX_DB} sslmode=${PG_SSLMODE}" \
  -c "select current_database(), current_user;"
```

启用 tunnel 后，后续 Sub2API 与 Onyx 的数据库配置都使用 `${PG_APP_HOST}:${PG_APP_PORT}`，并把 systemd 依赖从 `postgresql.service` 调整为 `pgsql-ssh-tunnel.service`。SSH tunnel 已提供传输加密；Sub2API 可继续设置 `sslmode=require`，Onyx 只需要把 PostgreSQL host/port 指向 tunnel 本地端口。如果 PostgreSQL 就在本机，跳过本节，继续使用 `127.0.0.1:5432`。

### 4.3 Redis 仅供 Sub2API 使用

编辑 Redis 配置：

```bash
sudo cp /etc/redis/redis.conf /etc/redis/redis.conf.bak.$(date +%F-%H%M%S)
sudo sed -i "s/^# requirepass .*/requirepass ${REDIS_PASSWORD}/" /etc/redis/redis.conf
if ! grep -q '^requirepass ' /etc/redis/redis.conf; then
  echo "requirepass ${REDIS_PASSWORD}" | sudo tee -a /etc/redis/redis.conf
fi
sudo systemctl enable --now redis-server
sudo systemctl restart redis-server
redis-cli -a "${REDIS_PASSWORD}" ping
```

期望输出：

```text
PONG
```

---

## 5. 部署 Sub2API

### 5.1 创建用户与目录

```bash
sudo useradd --system --home-dir /opt/sub2api --shell /usr/sbin/nologin sub2api || true
sudo mkdir -p "${SRC_ROOT}" "${SUB2API_APP_DIR}" /var/log/sub2api
sudo chown -R "$USER:$USER" "${SRC_ROOT}"
sudo chown -R sub2api:sub2api /var/log/sub2api
```

### 5.2 拉取源码

```bash
if [ ! -d "${SUB2API_SRC_DIR}/.git" ]; then
  git clone "${SUB2API_REPO}" "${SUB2API_SRC_DIR}"
fi
cd "${SUB2API_SRC_DIR}"
git pull --ff-only
```

如果需要部署当前开发分支或私有 fork，在这里切换分支：

```bash
git status --short --branch
```

### 5.3 构建前端

Sub2API 前端构建产物会写入后端嵌入目录，后端需要用 `-tags embed` 重新编译。

```bash
cd "${SUB2API_SRC_DIR}/frontend"
pnpm install --frozen-lockfile
pnpm run build
```

### 5.4 构建后端

```bash
cd "${SUB2API_SRC_DIR}/backend"
go build -tags embed -o sub2api ./cmd/server
sudo install -m 0755 sub2api "${SUB2API_APP_DIR}/sub2api"
sudo chown sub2api:sub2api "${SUB2API_APP_DIR}/sub2api"
```

### 5.5 写入配置文件

```bash
sudo mkdir -p /etc/sub2api "${SUB2API_APP_DIR}/data"
sudo cp "${SUB2API_SRC_DIR}/deploy/config.example.yaml" /etc/sub2api/config.yaml
sudo chown -R sub2api:sub2api "${SUB2API_APP_DIR}" "${SUB2API_APP_DIR}/data"
sudo chmod 750 "${SUB2API_APP_DIR}" "${SUB2API_APP_DIR}/data"
sudo chown root:sub2api /etc/sub2api/config.yaml
sudo chmod 640 /etc/sub2api/config.yaml
```

编辑 `/etc/sub2api/config.yaml`，至少确认这些值：

```yaml
server:
  host: "127.0.0.1"
  port: 8080
  mode: "release"
  frontend_url: "https://sub2api.example.com"

database:
  host: "127.0.0.1"
  port: 5432
  user: "sub2api"
  password: "replace-with-strong-sub2api-db-password"
  dbname: "sub2api"
  sslmode: "disable"

redis:
  host: "127.0.0.1"
  port: 6379
  password: "replace-with-strong-redis-password"
  db: 0

jwt:
  secret: "replace-with-64-random-chars"
```

如果配置文件字段结构与当前版本略有差异，以 `deploy/config.example.yaml` 中的实际 key 为准，保持上面的语义不变。

如果第 4.2 节启用了跨服务器 PostgreSQL SSH tunnel，数据库配置改为：

```yaml
database:
  host: "127.0.0.1"
  port: 15432
  user: "sub2api"
  password: "replace-with-strong-sub2api-db-password"
  dbname: "sub2api"
  sslmode: "require"
```

### 5.6 创建 systemd 服务

```bash
sudo tee /etc/systemd/system/sub2api.service >/dev/null <<'EOF'
[Unit]
Description=Sub2API
After=network-online.target postgresql.service redis-server.service
Wants=network-online.target postgresql.service redis-server.service

[Service]
Type=simple
User=sub2api
Group=sub2api
WorkingDirectory=/opt/sub2api
ExecStart=/opt/sub2api/sub2api --config /etc/sub2api/config.yaml
Restart=always
RestartSec=5
Environment=DATA_DIR=/opt/sub2api/data
Environment=GIN_MODE=release
StandardOutput=journal
StandardError=journal
SyslogIdentifier=sub2api

NoNewPrivileges=true
PrivateTmp=true
ProtectHome=true
ReadWritePaths=/opt/sub2api /var/log/sub2api

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable --now sub2api
sudo systemctl status sub2api --no-pager
```

如果第 4.2 节启用了跨服务器 PostgreSQL SSH tunnel，把 `[Unit]` 中的 PostgreSQL 依赖改为 tunnel 服务：

```ini
After=network-online.target pgsql-ssh-tunnel.service redis-server.service
Wants=network-online.target pgsql-ssh-tunnel.service redis-server.service
```

如果当前二进制不支持 `--config` 参数，改为把配置文件复制到工作目录：

```bash
sudo cp /etc/sub2api/config.yaml /opt/sub2api/config.yaml
sudo chown sub2api:sub2api /opt/sub2api/config.yaml
sudo sed -i 's#ExecStart=/opt/sub2api/sub2api --config /etc/sub2api/config.yaml#ExecStart=/opt/sub2api/sub2api#' /etc/systemd/system/sub2api.service
sudo systemctl daemon-reload
sudo systemctl restart sub2api
```

### 5.7 验证 Sub2API

```bash
curl -i http://127.0.0.1:8080/
curl -s http://127.0.0.1:8080/api/v1/settings/public | jq .
```

首次部署可能进入安装向导。可以通过浏览器访问 `https://sub2api.example.com` 完成初始化，也可以按当前版本支持的自动初始化方式写入管理员账户。完成初始化后重启：

```bash
sudo systemctl restart sub2api
```

---

## 6. 部署 Onyx Lite

### 6.1 创建用户与目录

```bash
sudo useradd --system --home-dir /opt/onyx --shell /usr/sbin/nologin onyx || true
sudo mkdir -p /opt/onyx /etc/onyx /var/log/onyx
sudo chown -R "$USER:$USER" /opt/onyx
sudo chown -R onyx:onyx /var/log/onyx
```

### 6.2 拉取源码

```bash
if [ ! -d "${ONYX_DIR}/.git" ]; then
  git clone "${ONYX_REPO}" "${ONYX_DIR}"
fi
cd "${ONYX_DIR}"
git pull --ff-only
git status --short --branch
```

### 6.3 安装 Onyx 后端 Python 依赖

```bash
cd "${ONYX_DIR}"
uv sync --frozen --group backend --group ee
```

如果服务器资源较小，可以先只安装 backend 依赖：

```bash
uv sync --frozen --group backend
```

但如果当前源码导入了 `ee` 目录或企业版相关模块，仍需安装 `--group ee`。

### 6.4 写入 Onyx 环境文件

创建 `/etc/onyx/onyx.env`：

```bash
sudo tee /etc/onyx/onyx.env >/dev/null <<EOF
PYTHONPATH=${ONYX_DIR}/backend
LD_PRELOAD=libjemalloc.so.2

AUTH_TYPE=basic
WEB_DOMAIN=https://${ONYX_DOMAIN}

POSTGRES_HOST=127.0.0.1
POSTGRES_PORT=5432
POSTGRES_USER=${ONYX_DB_USER}
POSTGRES_PASSWORD=${ONYX_DB_PASSWORD}
POSTGRES_DB=${ONYX_DB}

DISABLE_VECTOR_DB=true
FILE_STORE_BACKEND=postgres
CACHE_BACKEND=postgres
AUTH_BACKEND=postgres

SUB2API_INTEGRATION_ENABLED=true
SUB2API_BASE_URL=http://127.0.0.1:8080
SUB2API_EXCHANGE_SECRET=${SUB2API_ONYX_SECRET}
SUB2API_DEFAULT_TEXT_MODEL=gpt-5.5
SUB2API_DEFAULT_IMAGE_MODEL=gpt-image-2
SUB2API_ONYX_REDIRECT_PATH=/chat

API_SERVER_PROTOCOL=http
API_SERVER_HOST=127.0.0.1
API_SERVER_PORT=8081
EOF

sudo chmod 640 /etc/onyx/onyx.env
sudo chown root:onyx /etc/onyx/onyx.env
```

如果第 4.2 节启用了跨服务器 PostgreSQL SSH tunnel，Onyx 数据库变量改为：

```env
POSTGRES_HOST=127.0.0.1
POSTGRES_PORT=15432
POSTGRES_USER=${ONYX_DB_USER}
POSTGRES_PASSWORD=${ONYX_DB_PASSWORD}
POSTGRES_DB=${ONYX_DB}
```

### 6.5 执行 Onyx 数据库迁移

```bash
cd "${ONYX_DIR}/backend"
set -a
. /etc/onyx/onyx.env
set +a
"${ONYX_DIR}/.venv/bin/alembic" -c alembic.ini upgrade head
```

如果出现 `Multiple head revisions are present`，先检查 migration head：

```bash
"${ONYX_DIR}/.venv/bin/alembic" -c alembic.ini heads
```

当前 Sub2API 集成 migration 应该接在唯一当前 head 后面，不应形成第二个 head。

### 6.6 创建 Onyx API systemd 服务

```bash
sudo tee /etc/systemd/system/onyx-api.service >/dev/null <<EOF
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
```

确认源码目录对 `onyx` 服务用户可读。不要把整个源码目录递归 chown 给服务用户，否则后续 `git pull` 和构建会不方便；只要目录是常规 `755`、文件是常规 `644` 即可。

```bash
sudo chmod o+rx /opt /opt/onyx || true
sudo systemctl daemon-reload
sudo systemctl enable --now onyx-api
sudo systemctl status onyx-api --no-pager
```

如果第 4.2 节启用了跨服务器 PostgreSQL SSH tunnel，把 `[Unit]` 中的 PostgreSQL 依赖改为 tunnel 服务：

```ini
After=network-online.target pgsql-ssh-tunnel.service
Wants=network-online.target pgsql-ssh-tunnel.service
```

### 6.7 构建并运行 Onyx Web

```bash
cd "${ONYX_DIR}/web"
npm ci
NEXT_PRIVATE_STANDALONE=true NEXT_TELEMETRY_DISABLED=1 npm run build
```

创建 `/etc/onyx/web.env`：

```bash
sudo tee /etc/onyx/web.env >/dev/null <<EOF
NODE_ENV=production
NEXT_TELEMETRY_DISABLED=1
INTERNAL_URL=http://127.0.0.1:8081
WEB_DOMAIN=https://${ONYX_DOMAIN}
EOF

sudo chmod 640 /etc/onyx/web.env
sudo chown root:onyx /etc/onyx/web.env
```

创建 systemd 服务：

```bash
sudo tee /etc/systemd/system/onyx-web.service >/dev/null <<EOF
[Unit]
Description=Onyx Web
After=network-online.target onyx-api.service
Wants=network-online.target

[Service]
Type=simple
User=onyx
Group=onyx
WorkingDirectory=${ONYX_DIR}/web
EnvironmentFile=/etc/onyx/web.env
ExecStart=/usr/bin/npm run start -- -p 3000 -H 127.0.0.1
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=onyx-web

NoNewPrivileges=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable --now onyx-web
sudo systemctl status onyx-web --no-pager
```

### 6.8 验证 Onyx 本机端口

```bash
curl -i http://127.0.0.1:8081/health
curl -i http://127.0.0.1:3000/
```

---

## 7. 配置两边集成

### 7.1 Sub2API 侧 Onyx settings

推荐优先在 Sub2API 管理后台配置：

- 启用 Onyx 集成。
- Onyx base URL：`https://onyx.example.com`
- Onyx menu label：`Onyx`
- Exchange secret：与 `/etc/onyx/onyx.env` 中 `SUB2API_EXCHANGE_SECRET` 完全一致。
- Launch token TTL：`60`
- Default redirect path：`/chat`
- Default text model：`gpt-5.5`
- Default image model：`gpt-image-2`
- API base URL：`https://sub2api.example.com/v1`

如果需要由服务器上的 agent 直接写库，可在 Sub2API 初始化完成后执行：

```bash
sudo -u postgres psql -d "${SUB2API_DB}" <<SQL
INSERT INTO settings (key, value, updated_at) VALUES
  ('onyx_enabled', 'true', now()),
  ('onyx_base_url', 'https://${ONYX_DOMAIN}', now()),
  ('onyx_menu_label', 'Onyx', now()),
  ('onyx_exchange_secret', '${SUB2API_ONYX_SECRET}', now()),
  ('onyx_launch_token_ttl_seconds', '60', now()),
  ('onyx_default_redirect_path', '/chat', now()),
  ('onyx_default_text_model', 'gpt-5.5', now()),
  ('onyx_default_image_model', 'gpt-image-2', now()),
  ('api_base_url', 'https://${SUB2API_DOMAIN}/v1', now())
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value, updated_at = now();
SQL
```

重启 Sub2API：

```bash
sudo systemctl restart sub2api
```

### 7.2 创建可供 Onyx 使用的 Sub2API API Key

Onyx exchange 会选择当前 Sub2API 用户下第一条符合条件的 API Key：

- `status = active`
- 未过期
- `quota > 0`
- `quota_used < quota`

推荐通过 Sub2API Web UI 登录后创建 API Key。不要直接手写 `api_keys` 表，除非已经确认当前版本表结构。

### 7.3 Onyx 侧校验环境变量

```bash
sudo systemctl show onyx-api --property=EnvironmentFiles
sudo grep -E 'SUB2API|DISABLE_VECTOR_DB|CACHE_BACKEND|AUTH_BACKEND|FILE_STORE_BACKEND|WEB_DOMAIN' /etc/onyx/onyx.env
sudo systemctl restart onyx-api onyx-web
```

---

## 8. 配置 Nginx

### 8.1 Sub2API 站点

创建 `/etc/nginx/sites-available/sub2api.conf`：

```bash
sudo tee /etc/nginx/sites-available/sub2api.conf >/dev/null <<EOF
server {
    listen 80;
    server_name ${SUB2API_DOMAIN};

    client_max_body_size 512m;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_set_header X-Forwarded-Host \$host;
        proxy_set_header X-Forwarded-Port \$server_port;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_buffering off;
        proxy_read_timeout 300s;
        proxy_send_timeout 300s;
    }
}
EOF
```

### 8.2 Onyx 站点

创建 `/etc/nginx/sites-available/onyx.conf`。重点是 `/api/*` 需要去掉 `/api` 前缀再转发给 Onyx API。

```bash
sudo tee /etc/nginx/sites-available/onyx.conf >/dev/null <<EOF
map \$http_upgrade \$connection_upgrade {
    default upgrade;
    '' close;
}

server {
    listen 80;
    server_name ${ONYX_DOMAIN};

    client_max_body_size 5g;

    location ~ ^/(api|openapi.json)(/.*)?$ {
        rewrite ^/api(/.*)$ \$1 break;

        proxy_pass http://127.0.0.1:8081;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_set_header X-Forwarded-Host \$host;
        proxy_set_header X-Forwarded-Port \$server_port;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection \$connection_upgrade;
        proxy_buffering off;
        proxy_read_timeout 300s;
        proxy_send_timeout 300s;
    }

    location /scim {
        proxy_pass http://127.0.0.1:8081;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_set_header X-Forwarded-Host \$host;
        proxy_set_header X-Forwarded-Port \$server_port;
        proxy_buffering off;
        proxy_read_timeout 300s;
        proxy_send_timeout 300s;
    }

    location / {
        proxy_pass http://127.0.0.1:3000;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_set_header X-Forwarded-Host \$host;
        proxy_set_header X-Forwarded-Port \$server_port;
        proxy_read_timeout 300s;
        proxy_send_timeout 300s;
    }
}
EOF
```

启用站点：

```bash
sudo ln -sf /etc/nginx/sites-available/sub2api.conf /etc/nginx/sites-enabled/sub2api.conf
sudo ln -sf /etc/nginx/sites-available/onyx.conf /etc/nginx/sites-enabled/onyx.conf
sudo nginx -t
sudo systemctl reload nginx
```

### 8.3 配置 HTTPS

域名 DNS 已解析到服务器后执行：

```bash
sudo certbot --nginx -d "${SUB2API_DOMAIN}" -d "${ONYX_DOMAIN}"
sudo nginx -t
sudo systemctl reload nginx
```

---

## 9. 启动与验证

### 9.1 服务状态

```bash
sudo systemctl status postgresql redis-server sub2api onyx-api onyx-web nginx --no-pager
```

如果第 4.2 节启用了跨服务器 PostgreSQL SSH tunnel，改为检查：

```bash
sudo systemctl status pgsql-ssh-tunnel redis-server sub2api onyx-api onyx-web nginx --no-pager
```

### 9.2 本机端口

```bash
ss -lntp | grep -E ':(8080|8081|3000|5432|6379|80|443)\b'
```

期望：

- `127.0.0.1:8080` 有 Sub2API。
- `127.0.0.1:8081` 有 Onyx API。
- `127.0.0.1:3000` 有 Onyx Web。
- 单机 PostgreSQL 时，`127.0.0.1:5432` 有 PostgreSQL。
- 跨服务器 PostgreSQL SSH tunnel 时，`127.0.0.1:15432` 有 tunnel 监听。
- `0.0.0.0:80` 和 `0.0.0.0:443` 由 Nginx 监听。

跨服务器 PostgreSQL SSH tunnel 的端口检查命令：

```bash
ss -lntp | grep ":${PG_TUNNEL_LOCAL_PORT}\b"
PGPASSWORD="${SUB2API_DB_PASSWORD}" psql \
  "host=${PG_APP_HOST} port=${PG_APP_PORT} user=${SUB2API_DB_USER} dbname=${SUB2API_DB} sslmode=${PG_SSLMODE}" \
  -c "select current_database(), current_user;"
```

### 9.3 Sub2API public settings

```bash
curl -s "https://${SUB2API_DOMAIN}/api/v1/settings/public" | jq '.data | {api_base_url, onyx_enabled, onyx_menu_label, onyx_launch_path}'
```

期望：

```json
{
  "api_base_url": "https://sub2api.example.com/v1",
  "onyx_enabled": true,
  "onyx_menu_label": "Onyx",
  "onyx_launch_path": "/api/v1/onyx/launch"
}
```

### 9.4 Sub2API login + Onyx launch

```bash
LOGIN_JSON=$(curl -sS -X POST "https://${SUB2API_DOMAIN}/api/v1/auth/login" \
  -H 'Content-Type: application/json' \
  -d "{\"email\":\"${SUB2API_ADMIN_EMAIL}\",\"password\":\"${SUB2API_ADMIN_PASSWORD}\"}")

TOKEN=$(echo "$LOGIN_JSON" | jq -r '.data.access_token')

LAUNCH_JSON=$(curl -sS -X POST "https://${SUB2API_DOMAIN}/api/v1/onyx/launch" \
  -H "Authorization: Bearer ${TOKEN}")

echo "$LAUNCH_JSON" | jq .
EXCHANGE_URL=$(echo "$LAUNCH_JSON" | jq -r '.data.redirect_url')
echo "$EXCHANGE_URL"
```

期望 `redirect_url` 类似：

```text
https://onyx.example.com/api/sub2api/exchange?token=...
```

### 9.5 验证 Onyx exchange

```bash
curl -i -k -L --max-redirs 0 "$EXCHANGE_URL"
```

期望：

- HTTP `302`
- `Location: /chat`
- `Set-Cookie` 包含 `fastapiusersauth`

### 9.6 验证 Onyx 已写入用户级 Sub2API 凭证

```bash
sudo -u postgres psql -d "${ONYX_DB}" -c "
select u.email,
       c.sub2api_user_id,
       c.api_key_id,
       c.api_base_url,
       c.text_model_name,
       c.image_model_name,
       c.updated_at
from public.user u
join sub2api_user_credential c on c.user_id = u.id
order by c.updated_at desc
limit 5;
"
```

期望：

- `api_base_url = https://sub2api.example.com/v1`
- `text_model_name = gpt-5.5`
- `image_model_name = gpt-image-2`

### 9.7 浏览器验证

1. 打开 `https://sub2api.example.com`。
2. 使用 Sub2API 用户登录。
3. 确认侧边栏出现 `Onyx` 菜单。
4. 点击 `Onyx`。
5. 浏览器跳转到 `https://onyx.example.com` 并进入 Onyx 已登录页面。
6. 在 Onyx 聊天页发起文本聊天。
7. 如果已配置生图模型，继续验证 Image Generation。

如果文本聊天或生图失败，先检查 Sub2API 中该用户 API Key 是否可用，以及上游模型账号是否已配置。

---

## 10. 运维命令

### 10.1 查看日志

```bash
sudo journalctl -u sub2api -f
sudo journalctl -u onyx-api -f
sudo journalctl -u onyx-web -f
sudo tail -f /var/log/nginx/access.log /var/log/nginx/error.log
```

### 10.2 重启服务

```bash
sudo systemctl restart sub2api
sudo systemctl restart onyx-api onyx-web
sudo systemctl reload nginx
```

### 10.3 更新 Sub2API

```bash
cd "${SUB2API_SRC_DIR}"
git pull --ff-only
cd frontend
pnpm install --frozen-lockfile
pnpm run build
cd ../backend
go build -tags embed -o sub2api ./cmd/server
sudo systemctl stop sub2api
sudo install -m 0755 sub2api "${SUB2API_APP_DIR}/sub2api"
sudo chown sub2api:sub2api "${SUB2API_APP_DIR}/sub2api"
sudo systemctl start sub2api
```

### 10.4 更新 Onyx

```bash
cd "${ONYX_DIR}"
git pull --ff-only
uv sync --frozen --group backend --group ee
cd backend
set -a
. /etc/onyx/onyx.env
set +a
"${ONYX_DIR}/.venv/bin/alembic" -c alembic.ini upgrade head

cd "${ONYX_DIR}/web"
npm ci
NEXT_PRIVATE_STANDALONE=true NEXT_TELEMETRY_DISABLED=1 npm run build

sudo systemctl restart onyx-api onyx-web
```

### 10.5 备份

备份 PostgreSQL：

```bash
sudo -u postgres pg_dump -Fc "${SUB2API_DB}" > "/root/sub2api-$(date +%F).dump"
sudo -u postgres pg_dump -Fc "${ONYX_DB}" > "/root/onyx-$(date +%F).dump"
```

备份配置：

```bash
sudo tar czf "/root/sub2api-onyx-config-$(date +%F).tar.gz" \
  /etc/sub2api /etc/onyx /etc/nginx/sites-available/sub2api.conf /etc/nginx/sites-available/onyx.conf
```

---

## 11. 常见故障

### 11.1 Onyx API 启动时报 multiple heads

现象：

```text
FAILED: Multiple head revisions are present for given argument 'head'
```

处理：

```bash
cd "${ONYX_DIR}/backend"
set -a
. /etc/onyx/onyx.env
set +a
"${ONYX_DIR}/.venv/bin/alembic" -c alembic.ini heads
```

如果看到多个 head，需要检查最近新增 migration 的 `down_revision`。Sub2API 集成 migration 应接到当前唯一 head 后，而不是接到旧 revision。

### 11.2 Onyx exchange 返回 502

先看日志：

```bash
sudo journalctl -u onyx-api -n 200 --no-pager
```

如果日志包含：

```text
Sub2API exchange endpoint returned an invalid payload
```

说明 Onyx client 需要兼容 Sub2API 的标准响应包装 `{code,message,data}`。确认当前代码包含该修复后重启 Onyx API。

### 11.3 Onyx exchange 返回 400，邮箱非法

Onyx 邮箱校验会拒绝 `.local` 等保留域名。把 Sub2API 用户邮箱改成真实域名格式，例如：

```bash
sudo -u postgres psql -d "${SUB2API_DB}" -c "update users set email='admin@example.com', updated_at=now() where email='admin@sub2api.local';"
```

然后重新登录 Sub2API 并点击 Onyx。

### 11.4 Onyx 能登录但聊天调用失败

检查 Onyx 保存的用户级凭证：

```bash
sudo -u postgres psql -d "${ONYX_DB}" -c "select api_base_url, text_model_name, image_model_name from sub2api_user_credential order by updated_at desc limit 5;"
```

如果 `api_base_url` 为空，修复 Sub2API settings：

```bash
sudo -u postgres psql -d "${SUB2API_DB}" -c "insert into settings (key,value,updated_at) values ('api_base_url','https://${SUB2API_DOMAIN}/v1',now()) on conflict (key) do update set value=excluded.value, updated_at=now();"
```

然后重新 exchange。

### 11.5 Onyx Web 页面 404 或 API 调用 404

确认 Nginx `/api` 转发时执行了：

```nginx
rewrite ^/api(/.*)$ $1 break;
proxy_pass http://127.0.0.1:8081;
```

Onyx 后端路由本身不带 `/api` 前缀，Nginx 必须去掉该前缀。

### 11.6 Redis NOAUTH

Sub2API Redis 配置中的 password 必须与 `/etc/redis/redis.conf` 的 `requirepass` 一致：

```bash
redis-cli -a "${REDIS_PASSWORD}" ping
sudo journalctl -u sub2api -n 100 --no-pager
```

### 11.7 Nginx 502

检查本地端口：

```bash
curl -i http://127.0.0.1:8080/
curl -i http://127.0.0.1:8081/health
curl -i http://127.0.0.1:3000/
```

哪个失败就优先查看对应服务日志。

### 11.8 跨服务器 PostgreSQL SSH tunnel 断开

现象：

```text
connection refused
connection timed out
server closed the connection unexpectedly
```

先确认 tunnel 服务和本地转发端口：

```bash
sudo systemctl status pgsql-ssh-tunnel --no-pager
sudo journalctl -u pgsql-ssh-tunnel -n 100 --no-pager
ss -lntp | grep ":${PG_TUNNEL_LOCAL_PORT}\b"
```

再确认 SSH 免密与远端 PostgreSQL 本机可用：

```bash
sudo ssh -i /etc/sub2api/pgsql-tunnel_ed25519 \
  -o BatchMode=yes \
  "${PG_REMOTE_SSH_USER}@${PG_REMOTE_SSH_HOST}" \
  "pg_isready -h ${PG_REMOTE_DB_HOST} -p ${PG_REMOTE_DB_PORT}"
```

如果 tunnel 正常但应用仍连不上，检查应用配置是否连接本机 tunnel 端口，而不是远端公网地址：

```text
host: 127.0.0.1
port: 15432
sslmode: require
```

如果系统重启后应用先启动、tunnel 后启动，确认 Sub2API 与 Onyx API 的 systemd `[Unit]` 已依赖 `pgsql-ssh-tunnel.service`，然后执行：

```bash
sudo systemctl daemon-reload
sudo systemctl restart pgsql-ssh-tunnel sub2api onyx-api
```

---

## 附录：给服务器执行 agent 的最短检查清单

1. 不安装 Docker，不执行任何 `docker` 命令。
2. PostgreSQL 单机部署时只装一个实例，创建 `sub2api`、`onyx` 两个数据库；跨服务器部署时通过 SSH tunnel 暴露为应用服务器本机端口。
3. Redis 只给 Sub2API 用；Onyx Lite 使用 PostgreSQL backend。
4. Sub2API 监听 `127.0.0.1:8080`。
5. Onyx API 监听 `127.0.0.1:8081`。
6. Onyx Web 监听 `127.0.0.1:3000`。
7. Nginx 对外暴露两个域名。
8. Onyx 域名的 `/api/*` 必须去掉 `/api` 前缀后转发到 Onyx API。
9. 两边 exchange secret 必须完全一致。
10. Sub2API `api_base_url` 应为 `https://${SUB2API_DOMAIN}/v1`。
11. Onyx `SUB2API_BASE_URL` 应为 `http://127.0.0.1:8080`。
12. 验证成功标准是 launch 返回 exchange URL、exchange 返回 `302 /chat`、Onyx DB 写入 `sub2api_user_credential`。
