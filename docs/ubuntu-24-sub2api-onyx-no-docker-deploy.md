# Ubuntu 24.04 裸机部署 Sub2API + Onyx Lite 指南

> 当前这台机器已经实际落地的环境信息、密钥、环境变量、systemd 配置和 Nginx 配置，统一记录在 [sub2api-onyx-runtime-environment-and-config.md](/app/ai/sub2api-all/sub2api/docs/sub2api-onyx-runtime-environment-and-config.md)。

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
2. 执行第 3 到第 4 节，完成系统依赖，并确认本机 PostgreSQL、Redis 已存在且可用；如果本机不存在，则按第 4 节新增安装。
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
  - 优先使用本机已有的 PostgreSQL 实例，分别创建 `sub2api` 与 `onyx` 数据库；如果本机没有 PostgreSQL，则在本机新增部署一个实例。
  - Redis 仅供 Sub2API 使用；如果本机没有 Redis，则在本机新增部署一个实例。
  - Onyx Lite 使用 PostgreSQL 作为 cache/auth/file store，不启动 Redis、Vespa、OpenSearch、MinIO、model server、background worker。
- Nginx 对外提供 HTTP/HTTPS 和反向代理；如果服务器已有自定义 Nginx 主配置，直接把 Sub2API 与 Onyx 的 `server` 块合并进去，不强制新增独立站点配置文件。

推荐本机端口规划：

| 服务 | 监听地址 | 用途 |
| --- | --- | --- |
| Sub2API | `127.0.0.1:8080` | Sub2API 后端与嵌入前端 |
| Onyx API | `127.0.0.1:8081` | Onyx FastAPI 后端 |
| Onyx Web | `127.0.0.1:3000` | Onyx Next.js 前端 |
| PostgreSQL | `127.0.0.1:5432` | 两个服务共用实例，分库分用户 |
| Redis | `127.0.0.1:6379` | Sub2API token/cache |
| Nginx | `0.0.0.0:80/443` | 对外入口 |

说明：

- 推荐让 Sub2API、Onyx API、Onyx Web 都只监听本机回环地址，再由 Nginx 暴露对外入口。
- 如果你需要临时排查“浏览器直连 `IP:8080` 不通”这类问题，可以把 Sub2API 的 `SERVER_HOST` 临时改成 `0.0.0.0`，但这不应作为默认生产形态。
- 如果没有可用 HTTPS 域名，也可以先用 `http://域名或IP:端口` 暴露 Onyx；此时 `WEB_DOMAIN`、Sub2API `onyx_base_url`、浏览器访问入口必须保持同一个外部 URL。

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

export PG_APP_HOST="127.0.0.1"
export PG_APP_PORT="5432"
export PG_SSLMODE="disable"

export REDIS_PASSWORD="replace-with-strong-redis-password"
export SUB2API_ADMIN_EMAIL="admin@example.com"
export SUB2API_ADMIN_PASSWORD="replace-with-strong-admin-password"
export SUB2API_JWT_SECRET="replace-with-64-random-chars"
export SUB2API_ONYX_SECRET="replace-with-64-random-chars"
export ONYX_USER_AUTH_SECRET="replace-with-64-random-chars"
export ONYX_DB_READONLY_PASSWORD="replace-with-strong-readonly-password"
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
  libpq-dev libxmlsec1-dev libxmlsec1-openssl libjemalloc2 \
  python3.12 python3.12-venv python3.12-dev
```

如果你明确知道本机还没有 PostgreSQL 或 Redis，也可以在这里一并安装：

```bash
sudo apt install -y postgresql postgresql-contrib redis-server
```

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

### 4.1 本机 PostgreSQL 检查、安装并初始化

本节只使用本机 PostgreSQL。若本机已安装，则直接复用；若本机未安装，则新增安装并初始化。

```bash
if ! dpkg -s postgresql >/dev/null 2>&1; then
  sudo apt update
  sudo apt install -y postgresql postgresql-contrib
fi

sudo systemctl enable --now postgresql
sudo systemctl status postgresql --no-pager
```

创建或更新 `sub2api` 与 `onyx` 数据库用户、数据库：

```bash
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
```

验证：

```bash
PGPASSWORD="${SUB2API_DB_PASSWORD}" psql \
  -h "${PG_APP_HOST}" -p "${PG_APP_PORT}" \
  -U "${SUB2API_DB_USER}" -d "${SUB2API_DB}" \
  -c "select current_database(), current_user;"
PGPASSWORD="${ONYX_DB_PASSWORD}" psql \
  -h "${PG_APP_HOST}" -p "${PG_APP_PORT}" \
  -U "${ONYX_DB_USER}" -d "${ONYX_DB}" \
  -c "select current_database(), current_user;"
```

### 4.2 本机 Redis 检查、安装并初始化

本节只使用本机 Redis。若本机已安装，则直接复用；若本机未安装，则新增安装并初始化。

```bash
if ! dpkg -s redis-server >/dev/null 2>&1; then
  sudo apt update
  sudo apt install -y redis-server
fi

sudo cp /etc/redis/redis.conf /etc/redis/redis.conf.bak.$(date +%F-%H%M%S)
sudo sed -i "s/^#\? requirepass .*/requirepass ${REDIS_PASSWORD}/" /etc/redis/redis.conf
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
sudo mkdir -p /etc/sub2api
sudo chown -R sub2api:sub2api "${SUB2API_APP_DIR}"
sudo chmod 750 "${SUB2API_APP_DIR}"
```

创建 `/etc/sub2api/config.yaml`：

```bash
sudo tee /etc/sub2api/config.yaml >/dev/null <<EOF
server:
  host: "127.0.0.1"
  port: 8080
  mode: "release"
  frontend_url: "https://${SUB2API_DOMAIN}"

database:
  host: "127.0.0.1"
  port: 5432
  user: "${SUB2API_DB_USER}"
  password: "${SUB2API_DB_PASSWORD}"
  dbname: "${SUB2API_DB}"
  sslmode: "${PG_SSLMODE}"

redis:
  host: "127.0.0.1"
  port: 6379
  password: "${REDIS_PASSWORD}"
  db: 0

jwt:
  secret: "${SUB2API_JWT_SECRET}"
  expire_hour: 24

run_mode: "standard"
EOF

sudo chown root:sub2api /etc/sub2api/config.yaml
sudo chmod 640 /etc/sub2api/config.yaml
```

说明：

- `config.yaml` 是 Sub2API README 中“源码部署”的主路径。
- 当前二进制没有 `--config` 启动参数，不要写 `ExecStart=/opt/sub2api/sub2api --config /etc/sub2api/config.yaml`。
- systemd 必须显式设置 `DATA_DIR=/etc/sub2api`，让启动阶段的 `setup.NeedsSetup()` 和主配置加载都指向同一份 `config.yaml`。
- 如需临时允许外部直连 `:8080` 调试，可把 `SERVER_HOST` 从 `127.0.0.1` 改成 `0.0.0.0`，排查结束后再改回。

如果你明确需要无人值守自动初始化，也可以走 `env + AUTO_SETUP=true` 备选路径；但本文主路径以 `config.yaml` 为准。

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
Environment=DATA_DIR=/etc/sub2api
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

sudo systemctl daemon-reload
sudo systemctl enable --now sub2api
sudo systemctl status sub2api --no-pager
```

### 5.7 验证 Sub2API

```bash
curl -i http://127.0.0.1:8080/
curl -s http://127.0.0.1:8080/api/v1/settings/public | jq .
```

如果你使用的是已有数据库，这里应直接进入主服务，而不是 setup wizard。

如果这是全新安装、数据库还是空的，按 README 的源码部署路径，首次会进入 setup wizard。此时通过浏览器完成初始化后，程序会写入 `/etc/sub2api/config.yaml` 并创建安装锁文件。完成初始化后重启：

```bash
sudo systemctl restart sub2api
```

如果你不想走浏览器向导，才改用 `env + AUTO_SETUP=true` 备选路径。

---

## 6. 部署 Onyx Lite

### 6.1 创建用户与目录

```bash
sudo useradd --system --home-dir /opt/onyx --shell /usr/sbin/nologin onyx || true
sudo mkdir -p /opt/onyx /etc/onyx /var/log/onyx /var/lib/onyx
sudo chown -R "$USER:$USER" /opt/onyx
sudo chown -R onyx:onyx /var/log/onyx
sudo chown -R onyx:onyx /var/lib/onyx
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
HOME=/var/lib/onyx
HF_HOME=/var/lib/onyx/.cache/huggingface
PYTHONPATH=${ONYX_DIR}/backend
LD_PRELOAD=libjemalloc.so.2

AUTH_TYPE=basic
WEB_DOMAIN=https://${ONYX_DOMAIN}
USER_AUTH_SECRET=${ONYX_USER_AUTH_SECRET}

POSTGRES_HOST=127.0.0.1
POSTGRES_PORT=5432
POSTGRES_USER=${ONYX_DB_USER}
POSTGRES_PASSWORD=${ONYX_DB_PASSWORD}
POSTGRES_DB=${ONYX_DB}
DB_READONLY_USER=db_readonly_user
DB_READONLY_PASSWORD=${ONYX_DB_READONLY_PASSWORD}

DISABLE_VECTOR_DB=true
FILE_STORE_BACKEND=postgres
CACHE_BACKEND=postgres
AUTH_BACKEND=postgres

LITELLM_LOCAL_MODEL_COST_MAP=true
ONYX_SKIP_LITELLM_INIT=true
ONYX_SKIP_LITELLM_OLLAMA_REGISTER=true

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

说明：

- `HOME` 和 `HF_HOME` 必须显式设置到可写目录。否则 Onyx 首次加载 tokenizer 或 Hugging Face 缓存时，可能默认落到 `/opt/onyx` 并触发权限错误。
- 如果你采用的是 IP + 端口入口，而不是域名 + HTTPS，这里的 `WEB_DOMAIN` 应替换成实际外部入口，例如 `http://108.187.32.100:81`。
- `LITELLM_LOCAL_MODEL_COST_MAP=true` 用于避免 LiteLLM 在启动或首次聊天时访问 GitHub 拉取模型成本表；裸机服务器 DNS 或外网不可用时必须设置。
- `ONYX_SKIP_LITELLM_INIT=true` 和 `ONYX_SKIP_LITELLM_OLLAMA_REGISTER=true` 用于 Lite 部署路径跳过不需要的 LiteLLM 附加初始化与 Ollama 模型注册；当前 Sub2API 集成只走 OpenAI-compatible 聊天链路，不依赖这两段初始化。
- `USER_AUTH_SECRET` 必须稳定，不能每次重启变化；否则已登录 cookie 会失效。
- `DB_READONLY_USER` 与 `DB_READONLY_PASSWORD` 是 Onyx 启动时读取的只读数据库账号配置。若该用户不存在，按下一小节创建。

### 6.5 执行 Onyx 数据库迁移

如果 `/etc/onyx/onyx.env` 配置了 `DB_READONLY_USER`，先创建或更新只读数据库账号：

```bash
sudo -u postgres psql -d "${ONYX_DB}" <<SQL
DO \$\$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'db_readonly_user') THEN
    CREATE ROLE db_readonly_user LOGIN PASSWORD '${ONYX_DB_READONLY_PASSWORD}';
  ELSE
    ALTER ROLE db_readonly_user WITH PASSWORD '${ONYX_DB_READONLY_PASSWORD}';
  END IF;
END
\$\$;
GRANT CONNECT ON DATABASE ${ONYX_DB} TO db_readonly_user;
GRANT USAGE ON SCHEMA public TO db_readonly_user;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO db_readonly_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO db_readonly_user;
SQL
```

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
ReadWritePaths=/var/log/onyx /var/lib/onyx ${ONYX_DIR}/backend

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

### 6.7 构建并运行 Onyx Web

```bash
cd "${ONYX_DIR}/web"
npm ci
NEXT_PRIVATE_STANDALONE=true \
NEXT_TELEMETRY_DISABLED=1 \
INTERNAL_URL=http://127.0.0.1:8081 \
WEB_DOMAIN=https://${ONYX_DOMAIN} \
npm run build
```

如果构建阶段因为内存不足被杀掉，例如 `Killed`、退出码 `137` 或 Next 构建长时间卡死，按下面方式处理：

```bash
sudo fallocate -l 4G /swapfile_onyx_build
sudo chmod 600 /swapfile_onyx_build
sudo mkswap /swapfile_onyx_build
sudo swapon /swapfile_onyx_build

cd "${ONYX_DIR}/web"
NEXT_PRIVATE_STANDALONE=true \
NEXT_TELEMETRY_DISABLED=1 \
INTERNAL_URL=http://127.0.0.1:8081 \
WEB_DOMAIN=https://${ONYX_DOMAIN} \
NODE_OPTIONS=--max-old-space-size=3072 \
npx next build --webpack
```

构建完成后，把静态资源和 `public` 目录放进 standalone 运行目录：

```bash
cd "${ONYX_DIR}/web"
mkdir -p .next/standalone/.next
cp -a .next/static .next/standalone/.next/static
cp -a public .next/standalone/public
```

如果前面为了构建临时创建了 swap，确认系统内存压力恢复正常后可以清理：

```bash
sudo swapoff /swapfile_onyx_build
sudo rm -f /swapfile_onyx_build
```

创建 `/etc/onyx/web.env`：

```bash
sudo tee /etc/onyx/web.env >/dev/null <<EOF
NODE_ENV=production
NEXT_TELEMETRY_DISABLED=1
INTERNAL_URL=http://127.0.0.1:8081
WEB_DOMAIN=https://${ONYX_DOMAIN}
HOSTNAME=127.0.0.1
PORT=3000
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
Wants=network-online.target onyx-api.service

[Service]
Type=simple
User=onyx
Group=onyx
WorkingDirectory=${ONYX_DIR}/web/.next/standalone
EnvironmentFile=/etc/onyx/web.env
ExecStart=/usr/bin/node server.js
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=onyx-web

NoNewPrivileges=true
PrivateTmp=true
ProtectHome=true
ReadOnlyPaths=${ONYX_DIR}/web/.next/standalone

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

Onyx exchange 会使用用户从 Sub2API 页面点击 Onyx 时携带过去的 API Key 绑定信息。不要在 Onyx 侧单独手填 API Key，也不要为了验证而直接读取明文 key；需要确认时，直接在 Sub2API 页面验证该 key 是否可调用模型。

可供 Onyx 使用的 API Key 应满足：

- `status = active`
- 未过期
- 未超额；如果当前 Sub2API 版本把 `quota = 0` 定义为不限额，则 `quota = 0` 也应视为可用
- 具备目标模型所在分组或渠道权限

推荐通过 Sub2API Web UI 登录后创建 API Key。不要直接手写 `api_keys` 表，除非已经确认当前版本表结构。

Onyx 聊天页的模型列表应来自当前用户 API Key 对应的 Sub2API OpenAI-compatible 模型接口，即：

```text
GET https://${SUB2API_DOMAIN}/v1/models
```

不要用 LiteLLM 的模型成本表或 Onyx 内置模型枚举作为 Sub2API 模型列表来源。

### 7.3 Onyx 侧校验环境变量

```bash
sudo systemctl show onyx-api --property=EnvironmentFiles
sudo grep -E 'SUB2API|DISABLE_VECTOR_DB|CACHE_BACKEND|AUTH_BACKEND|FILE_STORE_BACKEND|WEB_DOMAIN' /etc/onyx/onyx.env
sudo systemctl restart onyx-api onyx-web
```

---

## 8. 配置 Nginx

### 8.1 Sub2API 站点

如果服务器已有统一 Nginx 配置文件，例如面板或自定义安装的 Nginx 使用 `/app/data/nginx/conf/nginx.conf`，不要再新增独立站点文件。直接把本节和 8.2 的 `server` 块合并进现有 `http { ... }` 配置即可。验证和重载也要使用实际运行中的 Nginx 二进制，例如：

```bash
/app/data/nginx/sbin/nginx -t
/app/data/nginx/sbin/nginx -s reload
```

如果使用 Ubuntu apt 安装的默认 Nginx，再按下面的 `/etc/nginx/sites-available/*` 方式创建站点。

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

如果这台机器已经有自定义 Nginx 在跑，不要再强行切回 Ubuntu 默认 `/etc/nginx/sites-enabled/*` 布局。把下面的 `server` 块合并进现有主配置，再执行对应 Nginx 二进制的 `-t` 和 reload。

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

### 8.3 非 HTTPS 或非 80/443 入口

如果部署初期只使用 `http://域名或IP:端口`，例如 `http://onyx.example.com:81`，需要保持三处完全一致：

- `/etc/onyx/onyx.env` 的 `WEB_DOMAIN`
- `/etc/onyx/web.env` 的 `WEB_DOMAIN`
- Sub2API settings 里的 `onyx_base_url`

修改后重启：

```bash
sudo systemctl restart sub2api onyx-api onyx-web
```

这种方式可以用于内网或临时验证；生产环境仍建议配置 HTTPS。

### 8.4 配置 HTTPS

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

### 9.2 本机端口

```bash
ss -lntp | grep -E ':(8080|8081|3000|5432|6379|80|443)\b'
```

期望：

- `127.0.0.1:8080` 有 Sub2API。
- `127.0.0.1:8081` 有 Onyx API。
- `127.0.0.1:3000` 有 Onyx Web。
- `127.0.0.1:5432` 有 PostgreSQL。
- `127.0.0.1:6379` 有 Redis。
- `0.0.0.0:80` 和 `0.0.0.0:443` 由 Nginx 监听。

如果你为了临时排查把 Sub2API 暴露到了 `0.0.0.0:8080`，这里只会多出一个外网可见监听；生产形态仍建议回到 `127.0.0.1:8080`。

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

### 9.7 验证 Onyx 模型列表来自 Sub2API

页面登录后，Onyx 获取 persona 可用模型时应返回 `Sub2API` provider，且 Sub2API 日志应出现 `/v1/models`。

服务端可用同等链路验证：

```bash
sudo journalctl -u sub2api --since "5 minutes ago" --no-pager | grep '/v1/models'
```

期望看到：

```text
path": "/v1/models", "method": "GET", ... "status_code": 200
```

Onyx API 返回的 provider 应类似：

```json
{
  "id": -2001,
  "name": "sub2api",
  "provider": "openai_compatible",
  "provider_display_name": "Sub2API"
}
```

### 9.8 验证 Onyx 聊天调用 Sub2API

1. 打开 `https://sub2api.example.com`。
2. 使用 Sub2API 用户登录。
3. 确认侧边栏出现 `Onyx` 菜单。
4. 点击 `Onyx`。
5. 浏览器跳转到 `https://onyx.example.com` 并进入 Onyx 已登录页面。
6. 在 Onyx 聊天页发起文本聊天。
7. 如果已配置生图模型，继续验证 Image Generation。

发起文本聊天后，Sub2API 日志应出现 `/v1/chat/completions`：

```bash
sudo journalctl -u sub2api --since "5 minutes ago" --no-pager | grep '/v1/chat/completions'
```

期望看到：

```text
path": "/v1/chat/completions", "method": "POST", ... "status_code": 200
```

如果 Onyx 页面长时间无回复，但 Sub2API 日志没有 `/v1/chat/completions`，问题在 Onyx 调用 Sub2API 之前；优先看 Onyx API 日志和 LiteLLM 初始化相关配置。

如果 Sub2API 已收到 `/v1/chat/completions` 但返回失败，再检查 Sub2API 中该用户 API Key 是否可用，以及上游模型账号是否已配置。

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
NEXT_PRIVATE_STANDALONE=true \
NEXT_TELEMETRY_DISABLED=1 \
INTERNAL_URL=http://127.0.0.1:8081 \
WEB_DOMAIN=https://${ONYX_DOMAIN} \
NODE_OPTIONS=--max-old-space-size=3072 \
npx next build --webpack
mkdir -p .next/standalone/.next
cp -a .next/static .next/standalone/.next/static
cp -a public .next/standalone/public

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

说明 Onyx 需要兼容 Sub2API 的标准响应包装 `{code,message,data}`。确认当前部署版本支持该响应格式后重启 Onyx API。

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

### 11.5 Onyx 模型列表没有显示 Sub2API 模型

先确认当前用户是从 Sub2API 页面点击 Onyx 进入，而不是直接打开 Onyx 域名。正常 exchange 后，Onyx DB 应写入用户级 Sub2API 凭证：

```bash
sudo -u postgres psql -d "${ONYX_DB}" -c "
select user_id, sub2api_user_id, api_key_id, api_base_url, text_model_name, updated_at
from sub2api_user_credential
order by updated_at desc
limit 5;
"
```

再确认 Sub2API 模型接口本身可用：

```bash
sudo journalctl -u sub2api --since "10 minutes ago" --no-pager | grep '/v1/models'
```

如果 Onyx providers 接口没有触发 `/v1/models`，通常是 exchange 未完成、cookie 失效、用户级凭证未写入，或当前部署版本未启用 Sub2API provider。

如果 `/v1/models` 返回 401，说明传入的 Sub2API API Key 不可用或不是从页面跳转绑定的 key。回到 Sub2API 页面验证该 key，不要在 Onyx 侧手填 key。

### 11.6 Onyx Web 页面 404 或 API 调用 404

确认 Nginx `/api` 转发时执行了：

```nginx
rewrite ^/api(/.*)$ $1 break;
proxy_pass http://127.0.0.1:8081;
```

Onyx 后端路由本身不带 `/api` 前缀，Nginx 必须去掉该前缀。

### 11.7 Redis NOAUTH

Sub2API Redis 配置中的 password 必须与 `/etc/redis/redis.conf` 的 `requirepass` 一致：

```bash
redis-cli -a "${REDIS_PASSWORD}" ping
sudo journalctl -u sub2api -n 100 --no-pager
```

### 11.8 Nginx 502

检查本地端口：

```bash
curl -i http://127.0.0.1:8080/
curl -i http://127.0.0.1:8081/health
curl -i http://127.0.0.1:3000/
```

哪个失败就优先查看对应服务日志。

### 11.9 本机 PostgreSQL 未启动或拒绝连接

现象：

```text
connection refused
server closed the connection unexpectedly
```

先确认本机 PostgreSQL 服务状态和监听端口：

```bash
sudo systemctl status postgresql --no-pager
sudo journalctl -u postgresql -n 100 --no-pager
ss -lntp | grep ':5432\b'
```

再确认数据库本身和应用用户可登录：

```bash
sudo -u postgres pg_isready -h 127.0.0.1 -p 5432
PGPASSWORD="${SUB2API_DB_PASSWORD}" psql \
  -h "${PG_APP_HOST}" -p "${PG_APP_PORT}" \
  -U "${SUB2API_DB_USER}" -d "${SUB2API_DB}" \
  -c "select current_database(), current_user;"
```

如果 PostgreSQL 正常但应用仍连不上，检查 Sub2API 和 Onyx 配置是否都指向本机：

```text
host: 127.0.0.1
port: 5432
sslmode: disable
```

如果系统重启后 PostgreSQL 慢于应用启动，确认 Sub2API 与 Onyx API 的 systemd `[Unit]` 已依赖 `postgresql.service`，然后执行：

```bash
sudo systemctl daemon-reload
sudo systemctl restart postgresql sub2api onyx-api
```

### 11.10 Sub2API 误进入 setup wizard

现象：

```text
Setup wizard available at http://...
```

但你明明已经写好了 `/etc/sub2api/config.yaml`，数据库里也已有数据。

处理：

- 确认 systemd 是否显式设置了 `DATA_DIR=/etc/sub2api`。
- 确认 `/etc/sub2api/config.yaml` 的权限允许 `sub2api` 用户读取。
- 不要给二进制传 `--config`，当前版本并不支持这个参数。

推荐 unit 片段：

```ini
[Service]
Environment=DATA_DIR=/etc/sub2api
ExecStart=/opt/sub2api/sub2api
```

改完后执行：

```bash
sudo systemctl daemon-reload
sudo systemctl restart sub2api
sudo journalctl -u sub2api -n 100 --no-pager
```

### 11.11 Onyx API 报 Hugging Face 缓存权限错误

现象：

```text
PermissionError: [Errno 13] Permission denied: '/opt/onyx'
```

处理：

```bash
sudo grep -E '^(HOME|HF_HOME)=' /etc/onyx/onyx.env
```

如果没有下面两行，就补上后重启：

```text
HOME=/var/lib/onyx
HF_HOME=/var/lib/onyx/.cache/huggingface
```

```bash
sudo systemctl restart onyx-api
sudo journalctl -u onyx-api -n 100 --no-pager
```

### 11.12 Onyx API 启动或首次聊天长时间卡住

现象：

- `curl http://127.0.0.1:8081/health` 长时间无法连接，或启动很久才监听。
- Onyx 聊天请求长时间无回复。
- Sub2API 日志没有 `/v1/chat/completions`，说明请求尚未到达 Sub2API。

先确认 LiteLLM 相关离线/跳过配置：

```bash
sudo grep -E 'LITELLM_LOCAL_MODEL_COST_MAP|ONYX_SKIP_LITELLM' /etc/onyx/onyx.env
```

期望至少包含：

```text
LITELLM_LOCAL_MODEL_COST_MAP=true
ONYX_SKIP_LITELLM_INIT=true
ONYX_SKIP_LITELLM_OLLAMA_REGISTER=true
```

如果缺失，补上后重启：

```bash
sudo systemctl restart onyx-api
sudo journalctl -u onyx-api -n 100 --no-pager
```

`LITELLM_LOCAL_MODEL_COST_MAP=true` 用来避免访问 GitHub 拉取模型成本表；后两项用于 Lite 部署跳过不需要的 LiteLLM 附加初始化和 Ollama 模型注册。当前 Sub2API 集成只需要 OpenAI-compatible 聊天调用。

### 11.13 Onyx Web 构建被 OOM 杀掉

现象：

```text
Killed
exit code 137
```

处理：

- 增加临时 swap，再重新构建。
- 构建时显式设置 `NODE_OPTIONS=--max-old-space-size=3072`。
- 优先使用 `npx next build --webpack`，与本次成功部署路径保持一致。

### 11.14 生图工具未显示或不可用

先确认 Sub2API 与 Onyx 两侧都具备前提条件：

- Sub2API 已配置可用上游生图渠道。
- 当前用户至少有一条可用的 Sub2API API Key。
- `onyx_default_image_model` 已配置，例如 `gpt-image-2`。

如果这些都满足，但 Onyx 前端仍不显示 Image Generation，检查当前部署版本是否支持 Sub2API 用户级图片凭证作为 Image Generation 可用性来源。

### 11.15 Onyx migration 缺少角色权限

部分环境下，Onyx migration 可能要求数据库用户临时具备更高权限。如果 `alembic upgrade head` 日志明确提示角色或权限不足，先为 `onyx` 用户临时补权，迁移完成后再收回：

```bash
sudo -u postgres psql -c "ALTER ROLE ${ONYX_DB_USER} CREATEROLE;"
cd "${ONYX_DIR}/backend"
set -a
. /etc/onyx/onyx.env
set +a
"${ONYX_DIR}/.venv/bin/alembic" -c alembic.ini upgrade head
sudo -u postgres psql -c "ALTER ROLE ${ONYX_DB_USER} NOCREATEROLE;"
```

---

## 附录：给服务器执行 agent 的最短检查清单

1. 不安装 Docker，不执行任何 `docker` 命令。
2. PostgreSQL 只使用本机单实例，创建 `sub2api`、`onyx` 两个数据库；本机没有 PostgreSQL 时先安装再初始化。
3. Redis 只给 Sub2API 用；Onyx Lite 使用 PostgreSQL backend。
4. Sub2API 源码部署主路径使用 `/etc/sub2api/config.yaml`，并在 systemd 里显式设置 `DATA_DIR=/etc/sub2api`。
5. Sub2API 默认监听 `127.0.0.1:8080`，仅在临时排障时才改成 `0.0.0.0:8080`。
6. 不要给 `sub2api` 二进制传 `--config`，当前版本不支持该参数。
7. Onyx API 监听 `127.0.0.1:8081`。
8. Onyx Web 监听 `127.0.0.1:3000`。
9. Nginx 对外暴露两个域名；已有统一 Nginx 主配置时，直接合并 `server` 块，不必新建独立配置文件。
10. Onyx 域名的 `/api/*` 必须去掉 `/api` 前缀后转发到 Onyx API。
11. 两边 exchange secret 必须完全一致。
12. Sub2API `api_base_url` 应为 `https://${SUB2API_DOMAIN}/v1`。
13. Onyx `SUB2API_BASE_URL` 应为 `http://127.0.0.1:8080`。
14. Onyx `HOME` 和 `HF_HOME` 应指向 `/var/lib/onyx` 这类服务用户可写目录。
15. Onyx Lite 裸机部署建议设置 `LITELLM_LOCAL_MODEL_COST_MAP=true`，避免启动或首次聊天时访问外网模型成本表。
16. Onyx Web 应从 `.next/standalone` 运行，而不是 `npm run start`。
17. 验证成功标准是 launch 返回 exchange URL、exchange 返回 `302 /chat`、Onyx DB 写入 `sub2api_user_credential`。
18. 模型列表成功标准是 Onyx providers 返回 `Sub2API` provider，Sub2API 日志出现 `/v1/models 200`。
19. 聊天成功标准是 Onyx 聊天返回内容，Sub2API 日志出现 `/v1/chat/completions 200`。
