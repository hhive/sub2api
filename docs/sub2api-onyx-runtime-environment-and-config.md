# Sub2API + Onyx 现网环境与配置总表

> 记录时间：2026-05-03
>
> 用途：集中保存这台机器当前部署的环境信息、服务配置、环境变量、密钥、端口、systemd 与 Nginx 配置。
>
> 注意：本文档包含明文密码、密钥和管理员账号，只适合保存在受控环境中，不要提交到公开仓库。

## 1. 部署概览

- 操作系统：Ubuntu 24.04
- 部署方式：裸机，无 Docker
- 外部入口：
  - Sub2API：`http://108.187.32.100/`
  - Onyx：`http://108.187.32.100:81/`
- Sub2API：`0.0.0.0:8080`，由 Nginx `:80` 反代
- Onyx API：`127.0.0.1:8081`
- Onyx Web：`127.0.0.1:3000`
- PostgreSQL：`127.0.0.1:5432`
- Redis：`127.0.0.1:6379`

## 2. 当前访问与状态

- 外部浏览器访问地址：
  - Sub2API：`http://108.187.32.100/`
  - Onyx：`http://108.187.32.100:81/`
- Sub2API 公开设置中的 `api_base_url`：`http://108.187.32.100/v1`
- Sub2API 存储的 `onyx_base_url`：`http://108.187.32.100:81`
- `sub2api.service`：运行中
- `onyx-api.service`：运行中
- `onyx-web.service`：运行中
- UFW：已放行 `22/tcp`、`80/tcp`、`81/tcp`、`443/tcp`

当前端口监听：

```text
0.0.0.0:80        nginx
0.0.0.0:81        nginx
0.0.0.0:8080      sub2api
127.0.0.1:8081    onyx-api
127.0.0.1:3000    onyx-web
127.0.0.1:5432    postgresql
127.0.0.1:6379    redis
```

## 3. 路径与目录

```bash
SUB2API_SRC_DIR=/app/ai/sub2api-all/sub2api
ONYX_DIR=/app/ai/sub2api-all/onyx
SUB2API_APP_DIR=/opt/sub2api
```

关键配置文件路径：

```text
/root/sub2api-onyx-vars.env
/etc/sub2api/config.yaml
/etc/sub2api/sub2api.env
/etc/onyx/onyx.env
/etc/onyx/web.env
/etc/systemd/system/sub2api.service
/etc/systemd/system/onyx-api.service
/etc/systemd/system/onyx-web.service
/app/data/nginx/conf/nginx.conf
```

## 4. 统一变量与密钥

来自 `/root/sub2api-onyx-vars.env`：

```bash
export SUB2API_DOMAIN="sub2api.localhost"
export ONYX_DOMAIN="onyx.localhost"

export SUB2API_SRC_DIR="/app/ai/sub2api-all/sub2api"
export ONYX_DIR="/app/ai/sub2api-all/onyx"
export SUB2API_APP_DIR="/opt/sub2api"

export SUB2API_DB="sub2api"
export SUB2API_DB_USER="sub2api"
export SUB2API_DB_PASSWORD="8d41d4c31c3355f6fa82b28dce613fcf"

export ONYX_DB="onyx"
export ONYX_DB_USER="onyx"
export ONYX_DB_PASSWORD="409f55c212faf7fa28adb0e1981dcae4"

export PG_APP_HOST="127.0.0.1"
export PG_APP_PORT="5432"
export PG_SSLMODE="disable"

export REDIS_PASSWORD="ecd54cf6cc74021c70c2566cb7abfcc3"
export SUB2API_ADMIN_EMAIL="admin@example.com"
export SUB2API_ADMIN_PASSWORD="gnwkoWAxDEiOhgr4yrcGeEBA"
export SUB2API_JWT_SECRET="ce175c6a9ef27cbae1285c7c027ab65eb16532ced066c1e20a21f41219e4af55"
export SUB2API_ONYX_SECRET="1914d828a7188b5701d1bb979d43ac3d841f655564f77345d554f030f076d7e6"
```

## 5. Sub2API 配置文件

文件：`/etc/sub2api/config.yaml`

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  mode: "release"
  frontend_url: "http://108.187.32.100"

database:
  host: "127.0.0.1"
  port: 5432
  user: "sub2api"
  password: "8d41d4c31c3355f6fa82b28dce613fcf"
  dbname: "sub2api"
  sslmode: "disable"

redis:
  host: "127.0.0.1"
  port: 6379
  password: "ecd54cf6cc74021c70c2566cb7abfcc3"
  db: 0

jwt:
  secret: "ce175c6a9ef27cbae1285c7c027ab65eb16532ced066c1e20a21f41219e4af55"
  expire_hour: 24

run_mode: "standard"
```

说明：

- `sub2api.service` 当前通过 `DATA_DIR=/etc/sub2api` 指向这份 `config.yaml`。
- `/etc/sub2api/sub2api.env` 仍保留为回退备份，但不再被当前 systemd unit 引用。

## 6. Onyx API 环境文件

文件：`/etc/onyx/onyx.env`

```bash
HOME=/tmp/onyx
HF_HOME=/tmp/onyx/.cache/huggingface
PYTHONPATH=/app/ai/sub2api-all/onyx/backend
LD_PRELOAD=libjemalloc.so.2

AUTH_TYPE=basic
WEB_DOMAIN=http://108.187.32.100:81

POSTGRES_HOST=127.0.0.1
POSTGRES_PORT=5432
POSTGRES_USER=onyx
POSTGRES_PASSWORD=409f55c212faf7fa28adb0e1981dcae4
POSTGRES_DB=onyx

DISABLE_VECTOR_DB=true
FILE_STORE_BACKEND=postgres
CACHE_BACKEND=postgres
AUTH_BACKEND=postgres

SUB2API_INTEGRATION_ENABLED=true
SUB2API_BASE_URL=http://127.0.0.1:8080
SUB2API_LLM_BASE_URL=http://127.0.0.1:8080/v1
SUB2API_EXCHANGE_SECRET=1914d828a7188b5701d1bb979d43ac3d841f655564f77345d554f030f076d7e6
SUB2API_DEFAULT_TEXT_MODEL=gpt-5.5
SUB2API_DEFAULT_IMAGE_MODEL=gpt-image-2
SUB2API_ONYX_REDIRECT_PATH=/chat

API_SERVER_PROTOCOL=http
API_SERVER_HOST=127.0.0.1
API_SERVER_PORT=8081
```

说明：

- `SUB2API_BASE_URL` 只用于 Onyx 后端调用 Sub2API 的 launch token exchange 接口。
- `SUB2API_LLM_BASE_URL` 只用于 Onyx 聊天模型请求，当前同机部署固定走本机 `127.0.0.1`，避免绕公网入口。
- Sub2API 公开设置中的 `api_base_url` 面向外部客户端和 API Key 配置展示，不应改成本机 `127.0.0.1`。

## 7. Onyx Web 环境文件

文件：`/etc/onyx/web.env`

```bash
NODE_ENV=production
NEXT_TELEMETRY_DISABLED=1
INTERNAL_URL=http://127.0.0.1:8081
WEB_DOMAIN=http://108.187.32.100:81
HOSTNAME=127.0.0.1
PORT=3000
```

## 8. systemd 配置

### 8.1 Sub2API

文件：`/etc/systemd/system/sub2api.service`

```ini
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
```

### 8.2 Onyx API

文件：`/etc/systemd/system/onyx-api.service`

```ini
[Unit]
Description=Onyx API Lite
After=network-online.target postgresql.service
Wants=network-online.target postgresql.service

[Service]
Type=simple
User=onyx
Group=onyx
WorkingDirectory=/app/ai/sub2api-all/onyx/backend
EnvironmentFile=/etc/onyx/onyx.env
ExecStartPre=/app/ai/sub2api-all/onyx/.venv/bin/alembic -c /app/ai/sub2api-all/onyx/backend/alembic.ini upgrade head
ExecStart=/app/ai/sub2api-all/onyx/.venv/bin/uvicorn onyx.main:app --host 127.0.0.1 --port 8081
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=onyx-api
NoNewPrivileges=true
PrivateTmp=true
ReadWritePaths=/var/log/onyx /app/ai/sub2api-all/onyx/backend

[Install]
WantedBy=multi-user.target
```

### 8.3 Onyx Web

文件：`/etc/systemd/system/onyx-web.service`

```ini
[Unit]
Description=Onyx Web
After=network-online.target onyx-api.service
Wants=network-online.target onyx-api.service

[Service]
Type=simple
User=onyx
Group=onyx
WorkingDirectory=/app/ai/sub2api-all/onyx/web/.next/standalone
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
ReadOnlyPaths=/app/ai/sub2api-all/onyx/web/.next/standalone

[Install]
WantedBy=multi-user.target
```

## 9. Nginx 配置

文件：`/app/data/nginx/conf/nginx.conf`

```nginx
#user  nobody;
worker_processes  1;

events {
    worker_connections  1024;
}

http {
    include       mime.types;
    default_type  application/octet-stream;
    sendfile        on;
    keepalive_timeout  65;

    server {
        listen       80;
        server_name  localhost _;

        location / {
            proxy_pass         http://127.0.0.1:8080;
            proxy_http_version 1.1;
            proxy_set_header   Host $host;
            proxy_set_header   X-Real-IP $remote_addr;
            proxy_set_header   X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header   X-Forwarded-Proto $scheme;
            proxy_set_header   Connection "";
            proxy_buffering    off;
        }

        error_page   500 502 503 504  /50x.html;
        location = /50x.html {
            root   html;
        }
    }

    server {
        listen       81;
        server_name  localhost _;

        location /api/ {
            rewrite ^/api/(.*)$ /$1 break;
            proxy_pass         http://127.0.0.1:8081;
            proxy_http_version 1.1;
            proxy_set_header   Host $host;
            proxy_set_header   X-Real-IP $remote_addr;
            proxy_set_header   X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header   X-Forwarded-Proto $scheme;
            proxy_set_header   Connection "";
            proxy_buffering    off;
        }

        location = /openapi.json {
            proxy_pass         http://127.0.0.1:8081/openapi.json;
            proxy_http_version 1.1;
            proxy_set_header   Host $host;
            proxy_set_header   X-Real-IP $remote_addr;
            proxy_set_header   X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header   X-Forwarded-Proto $scheme;
            proxy_set_header   Connection "";
            proxy_buffering    off;
        }

        location / {
            proxy_pass         http://127.0.0.1:3000;
            proxy_http_version 1.1;
            proxy_set_header   Host $host;
            proxy_set_header   X-Real-IP $remote_addr;
            proxy_set_header   X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header   X-Forwarded-Proto $scheme;
            proxy_set_header   Connection "";
            proxy_buffering    off;
        }

        error_page   500 502 503 504  /50x.html;
        location = /50x.html {
            root   html;
        }
    }
}
```

## 10. 数据库与中间件

PostgreSQL：

- Host：`127.0.0.1`
- Port：`5432`
- Sub2API DB：`sub2api`
- Sub2API User：`sub2api`
- Onyx DB：`onyx`
- Onyx User：`onyx`

Redis：

- Host：`127.0.0.1`
- Port：`6379`
- Password：`ecd54cf6cc74021c70c2566cb7abfcc3`
- DB：`0`

## 11. 当前公开设置

当前 `GET /api/v1/settings/public` 关键字段：

```json
{
  "api_base_url": "http://108.187.32.100/v1",
  "onyx_base_url": "http://108.187.32.100:81",
  "onyx_enabled": true,
  "onyx_menu_label": "Onyx",
  "onyx_launch_path": "/api/v1/onyx/launch"
}
```

`api_base_url` 是外部客户端使用的公开 OpenAI-compatible endpoint。Onyx 服务端内部聊天调用由 `/etc/onyx/onyx.env` 中的 `SUB2API_LLM_BASE_URL` 控制。

## 12. 当前已知未完成项

- 还没有配置 HTTPS。
- Onyx 通过 Nginx `:81` 对外暴露，内部服务本身仍只监听本机。
- Sub2API 的 Onyx 跳转链路还缺“可用 API Key”初始化数据。
- 当前 `onyx/launch` 会返回 `ONYX_NO_ELIGIBLE_API_KEY`，原因是 `api_keys` 表中没有可用 key。
- 临时构建与部署脚本仍保留在 `docs/tmp-*` 下，尚未清理。
