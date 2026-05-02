# Onyx 集成进度交接（2026-05-01）

## 当前总体状态

`sub2api` 侧的 Onyx launch token 服务已经完成第一阶段骨架，`CreateLaunch` 与 `ConsumeLaunch` 的服务层主链路已经打通，并补了一批围绕一次性 token 语义、API Key 二次校验和 exchange payload 的单元测试。

截至 2026-05-02，sub2api 前端 Sidebar 入口也已完成最小闭环：前端会根据 public settings 中的 Onyx 开关显示菜单，点击后调用 `POST /api/v1/onyx/launch`，并跳转到后端返回的 `redirect_url`。

今天的推进重点是两部分：

1. 修好 Claude Code 的子agent/worktree 环境，后续可以继续按子agent规则推进。
2. 继续用 TDD 推进 `OnyxLaunchService.ConsumeLaunch`，把 token 读取、校验、单次消费语义补到当前最小闭环。
3. 继续用 TDD 补齐 `ConsumeLaunch` 的 API Key 二次校验与完整 exchange payload。

---

## 今天已完成内容

### 1. Claude Code 子agent环境已配置完成

已完成以下全局配置：

- `C:\Users\jax\.claude\settings.json`
  - 新增 `WorktreeCreate` / `WorktreeRemove` hooks
- `C:\Users\jax\.claude\hooks\worktree-create.ps1`
- `C:\Users\jax\.claude\hooks\worktree-remove.ps1`

作用：即使当前工作目录不是 git 仓库，也可以让 Claude Code 的 Agent 以 `worktree` 隔离方式运行，满足你要求的“复杂任务按子agent拆分推进”。

### 2. `CreateLaunch` 已完成的能力

文件：`sub2api/backend/internal/service/onyx_launch_service.go`

已完成：

- 校验 Onyx 配置可用性：
  - `OnyxEnabled`
  - `OnyxBaseURL`
  - `OnyxExchangeSecret`
- 按规则选择当前用户第一条可用 API Key：
  - `status == active`
  - 未过期
  - `quota_used < quota`
  - `quota = 0` 不算可用
  - 按 `CreatedAt` 升序，`ID` 作为稳定 tie-breaker
- 生成随机 launch token
- 存储 launch token 与 `user_id/api_key_id` 绑定
- 按 Onyx base URL 组装跳转地址：
  - `/api/sub2api/exchange?token=...`

### 3. `ConsumeLaunch` 已完成的能力

文件：`sub2api/backend/internal/service/onyx_launch_service.go`

目前已完成：

- `TrimSpace` 后再处理 token
- 空 token / 全空白 token 直接返回 unauthorized
- `tokenStore == nil` 时返回 unauthorized
- `GetLaunchToken` 返回错误时透传该错误
- `GetLaunchToken` 返回 `nil` 数据时返回 unauthorized
- token 数据中 `UserID <= 0` 或 `APIKeyID <= 0` 时返回 unauthorized
- 只要 token 成功取出，就会立即执行删除
- 即使 token 绑定数据非法，也会先删除 token，保证不能重试
- 删除成功后，必须重新读取 token 绑定的 API Key 并校验：
  - API Key 仍属于 token 绑定的 sub2api user
  - `status == active`
  - 未过期
  - `quota_used < quota`
  - `quota = 0` 不算可用
- `apiKeyRepo == nil` 时返回 service unavailable，避免绕过二次校验
- 成功路径返回完整 exchange payload：
  - `user_id`
  - `email`
  - `username`
  - `api_key_id`
  - `api_key`
  - `api_base_url`
  - `text_model_name`
  - `image_model_name`
- 如果 `DeleteLaunchToken` 返回错误，当前实现会直接返回该错误

### 4. 已补齐的关键单元测试

文件：`sub2api/backend/internal/service/onyx_launch_service_test.go`

今天和此前连续补齐的关键测试包括：

- `TestOnyxLaunchService_SelectFirstEligibleAPIKey_PicksEarliestCreatedEligibleKey`
- `TestOnyxLaunchService_SelectFirstEligibleAPIKey_ReturnsConflictWhenNoEligibleKeyExists`
- `TestOnyxLaunchService_CreateLaunch_ReturnsServiceUnavailableWhenOnyxBaseURLMissing`
- `TestOnyxLaunchService_CreateLaunch_ReturnsServiceUnavailableWhenOnyxExchangeSecretMissing`
- `TestOnyxLaunchService_CreateLaunch_ReturnsConflictWhenNoEligibleAPIKeyExists`
- `TestOnyxLaunchService_CreateLaunch_ReturnsRedirectURLWithLaunchToken`
- `TestOnyxLaunchService_CreateLaunch_StoresLaunchTokenWithTTLAndBinding`
- `TestOnyxLaunchService_ConsumeLaunch_ReturnsUnauthorizedWhenTokenEmpty`
- `TestOnyxLaunchService_ConsumeLaunch_TrimsTokenAndReturnsPayloadWhenTokenDataValid`
- `TestOnyxLaunchService_ConsumeLaunch_ReturnsUnauthorizedWhenTokenDataMissing`
- `TestOnyxLaunchService_ConsumeLaunch_ReturnsUnauthorizedWhenTokenDataHasInvalidIDs`
- `TestOnyxLaunchService_ConsumeLaunch_InvalidatesTokenAfterSuccessfulConsumption`
- `TestOnyxLaunchService_ConsumeLaunch_DeletesTokenWhenTokenDataHasInvalidIDs`
- `TestOnyxLaunchService_ConsumeLaunch_ReturnsConflictWhenBoundAPIKeyNoLongerEligible`
- `TestOnyxLaunchService_ConsumeLaunch_ReturnsExchangePayloadForEligibleBoundAPIKey`
- `TestOnyxLaunchService_ConsumeLaunch_ReturnsServiceUnavailableWhenAPIKeyRepoMissing`
- `TestOnyxLaunchTokenStore_RedisError`
- `TestOnyxHandler_Launch_ReturnsRedirectURLForAuthenticatedUser`
- `TestOnyxHandler_Launch_ReturnsUnauthorizedWithoutAuthenticatedUser`
- `TestOnyxHandler_Exchange_ReturnsUnauthorizedWhenSecretMissing`
- `TestOnyxHandler_Exchange_ReturnsPayloadWhenSecretValid`
- `TestOnyxHandler_Exchange_ReturnsBadRequestWhenTokenEmpty`

### 5. Onyx Image Generation 用户级 sub2api credential 已接入

文件：`onyx/backend/onyx/tools/tool_constructor.py`

已完成：

- 新增 `_get_user_sub2api_image_generation_config(...)`
- 构造 ImageGenerationTool 时优先读取当前 Onyx 用户的 `Sub2APIUserCredential`
- 存在用户级 credential 时使用：
  - provider：`openai_compatible`
  - model：`credential.image_model_name`
  - api_key：用户级 sub2api API Key
  - api_base：用户级 sub2api API base URL
  - deployment_name：`credential.image_model_name`
- 用户级 credential 不存在时保持原有全局 Image Generation config fallback

新增测试：

- `onyx/backend/tests/unit/onyx/tools/test_sub2api_image_generation_config.py`
  - 覆盖用户级 image credential 配置生成
  - 覆盖 `_construct_tools_impl` 构造 ImageGenerationTool 时优先使用用户级 sub2api credential

### 6. Onyx LLM 单测收集阶段挂起问题已修复

问题：

- `backend/tests/unit/onyx/llm/test_sub2api_user_credentials.py` 通过 pytest 运行时会在测试收集阶段挂起。
- `faulthandler` 定位到 `backend/tests/unit/onyx/llm/conftest.py` import `onyx.llm.litellm_singleton.config` 时，Python 先执行 `onyx.llm.litellm_singleton.__init__`。
- `__init__` 会立即调用 `initialize_litellm()`，其中 `register_ollama_models()` 触发 LiteLLM 尝试连接 Ollama/网络，在当前环境中阻塞。

已完成：

- `onyx/backend/onyx/llm/litellm_singleton/__init__.py`
  - 新增 `ONYX_SKIP_LITELLM_INIT=true` 环境开关，仅在显式设置时跳过初始化。
  - 默认生产行为不变，仍会执行 `initialize_litellm()` 和 `apply_monkey_patches()`。
- `onyx/backend/tests/unit/onyx/llm/conftest.py`
  - 在导入 `litellm_singleton.config` 前设置 `ONYX_SKIP_LITELLM_INIT=true`。
  - LLM 单测仍会按原 fixture 加载 model metadata enrichments，并清理 parser cache。

验证：

- `pytest -q backend/tests/unit/onyx/llm/test_sub2api_user_credentials.py` 通过。
- LLM、Image Generation、sub2api client/api 相关 7 个单测一起运行通过。

Docker Postgres 验证：

- 已通过 Docker 启动临时 Postgres：
  - 容器名：`onyx-sub2api-test-postgres`
  - 镜像：`postgres:16-alpine`
  - 端口：`127.0.0.1:5432`
  - 用户/密码/库：`postgres/password/postgres`
- 已执行 `alembic upgrade heads` 初始化 schema。
- `pytest -q backend/tests/external_dependency_unit/db/test_sub2api_user_credentials.py` 已通过。
- LLM、Image Generation、sub2api client/api、DB credential helper 相关 8 个测试一起运行通过。

### 7. Onyx sub2api exchange client 错误映射已细化

文件：`onyx/backend/onyx/server/sub2api/client.py`

已完成：

- sub2api exchange 返回 401 时映射为 `INVALID_TOKEN`，提示 launch link 无效或过期。
- sub2api exchange 返回 409 或 `ONYX_NO_ELIGIBLE_API_KEY` 时映射为 `CONFLICT`，提示用户没有可用 API Key。
- sub2api exchange 返回 503 时映射为 `SERVICE_UNAVAILABLE`，提示 sub2api 集成未配置。
- 未识别的上游错误仍保留 `BAD_GATEWAY` fallback，并保留上游 HTTP status override。

测试：

- `onyx/backend/tests/unit/onyx/server/sub2api/test_client.py`
  - 新增 401、409、503 参数化覆盖。
- LLM、Image Generation、sub2api client/api、DB credential helper 相关 10 个测试一起运行通过。

### 8. sub2api 侧目标测试已复验

已通过：

- `go -C sub2api/backend test -p=1 -tags unit ./internal/service -run TestOnyxLaunchService`
- `go -C sub2api/backend test -p=1 -tags unit ./internal/handler -run TestOnyxHandler`
- `npm run test:run -- src/components/layout/__tests__/AppSidebar.spec.ts src/api/__tests__/onyx.spec.ts`
- `npm run typecheck`

### 9. Onyx 自动创建/复用用户 helper 测试已补齐

文件：`onyx/backend/tests/unit/onyx/server/sub2api/test_api.py`

新增覆盖：

- `upsert_onyx_user_from_sub2api` 会复用已有 web login 用户，并规范化 email。
- sub2api 用户在 Onyx 不存在时会自动创建 `STANDARD` / `BASIC` / verified web 用户。
- 已存在的非 web login 用户会被拒绝。
- 空 email 会被拒绝。

验证：

- LLM、Image Generation、sub2api client/api、DB credential helper 相关 14 个测试一起运行通过。
- `ruff check backend/tests/unit/onyx/server/sub2api/test_api.py backend/onyx/server/sub2api/api.py backend/onyx/server/sub2api/client.py backend/tests/unit/onyx/server/sub2api/test_client.py` 通过。

### 10. Onyx exchange endpoint 拒绝路径与公开路由测试已补齐

文件：`onyx/backend/tests/unit/onyx/server/sub2api/test_api.py`

新增覆盖：

- `SUB2API_INTEGRATION_ENABLED=false` 时拒绝 exchange。
- 空 launch token 时拒绝 exchange。
- exchange 成功但 Onyx 用户 inactive 时拒绝登录，且不保存 credential、不 commit。
- exchange 成功但 Onyx 用户不是 web login account type 时拒绝登录，且不保存 credential、不 commit。
- `/sub2api/exchange` 被 `auth_check.PUBLIC_ENDPOINT_SPECS` 识别为 public endpoint。

验证：

- LLM、Image Generation、sub2api client/api、DB credential helper 相关 19 个测试一起运行通过。
- `ruff check backend/tests/unit/onyx/server/sub2api/test_api.py` 通过。

---

## 当前完成状态与剩余工作

### 1. sub2api 后端 Onyx launch / exchange API 已完成代码接入

已落地：

- Redis 版 `OnyxLaunchTokenStore`
- `OnyxLaunchService` 构造函数和 Wire provider
- 用户侧 `OnyxHandler.Launch`
- 服务端侧 `OnyxHandler.Exchange`
- exchange endpoint 的 `X-Sub2API-Onyx-Secret` 共享密钥校验
- exchange handler 对 `ConsumeLaunch` payload 的 HTTP 返回封装
- handler 聚合 / router / `wire_gen.go` 注入
- `POST /api/v1/onyx/launch`
- `POST /api/v1/onyx/exchange`
- exchange endpoint 的共享密钥校验
- exchange handler 对 `ConsumeLaunch` payload 的 HTTP 返回封装

已验证：

- `TestOnyxLaunchService`
- `TestOnyxHandler`

### 2. Redis token store 真实存储层已完成 Docker 验证

已新增独立 Redis integration 测试文件：

- `sub2api/backend/internal/repository/onyx_launch_token_store_redis_integration_test.go`

该测试使用 `redis_integration` build tag，不再复用仓库通用 `integration` harness，因此不会启动 Postgres/testcontainers。默认连接 `127.0.0.1:6379`，也可通过 `ONYX_REDIS_ADDR` 指定 Redis；Redis 不可用时会快速 skip。

2026-05-02 已验证：

- 已启动 Docker Redis 容器 `sub2api-redis-test`，镜像 `redis:7-alpine`，端口 `127.0.0.1:6379`。
- 已执行 `go -C sub2api/backend test -v -p=1 -tags redis_integration ./internal/repository -run TestOnyxLaunchTokenStore_RedisIntegration`。
- `TestOnyxLaunchTokenStore_RedisIntegration` 与 `TestOnyxLaunchTokenStore_RedisIntegration_InvalidJSONReturnsError` 均通过。

### 3. sub2api 前端 Sidebar Onyx 菜单已完成代码接入

已落地：

- public settings 中 Onyx 菜单配置前端消费
- `launchOnyx()` 前端 API 封装
- Sidebar 用户主菜单与管理员“我的账户”菜单中的 Onyx 入口
- 点击后调用 `/api/v1/onyx/launch`
- 成功后跳转后端返回的 `redirect_url`
- loading 防重复点击
- 409 无可用 API Key、503 未配置、通用错误 toast 文案
- Onyx 菜单 feature flag 统一接入 `FeatureFlags.onyx`

已验证：

- `AppSidebar.spec.ts`
- `onyx.spec.ts`
- `npm run typecheck`

### 4. Onyx 端 exchange / 自动登录链路已完成代码接入

- `onyx/backend/onyx/server/sub2api/models.py`
  - 定义 sub2api exchange 响应模型
  - 将 sub2api 扁平 payload 转换为 Onyx 内部 `user + credential` 结构
- `onyx/backend/onyx/server/sub2api/client.py`
  - 调用 sub2api `/api/v1/onyx/exchange`
  - 使用 `X-Sub2API-Onyx-Secret` 共享密钥 header
  - 解析完整 payload
  - 将上游 HTTP/网络/响应结构错误转换为 `OnyxError`
- `onyx/backend/onyx/configs/app_configs.py`
  - 新增 `SUB2API_INTEGRATION_ENABLED`
  - 新增 `SUB2API_BASE_URL`
  - 新增 `SUB2API_EXCHANGE_SECRET`
  - 新增 `SUB2API_DEFAULT_TEXT_MODEL`
  - 新增 `SUB2API_DEFAULT_IMAGE_MODEL`
  - 新增 `SUB2API_ONYX_REDIRECT_PATH`
- `onyx/backend/onyx/db/models.py`
  - 新增 `Sub2APIUserCredential`
  - API Key 使用 `EncryptedString`
  - `user_id` 唯一，保证每个 Onyx 用户一条当前 credential
- `onyx/backend/onyx/db/sub2api_user_credentials.py`
  - `get_sub2api_credential_for_user`
  - `upsert_sub2api_credential_for_user`
  - `delete_sub2api_credential_for_user`
- `onyx/backend/alembic/versions/4f2b7c8d9e10_add_sub2api_user_credential.py`
  - 新增 `sub2api_user_credential` 表 migration
- `onyx/backend/onyx/server/sub2api/api.py`
  - 新增 `GET /api/sub2api/exchange?token=...`
  - 校验 Onyx 侧 sub2api 集成开关
  - 调用 sub2api exchange client 消费 launch token
  - 自动创建或复用 Onyx web login 用户
  - 保存/覆盖用户级 sub2api credential
  - 复用 `auth_backend.login()` 写入 Onyx 登录 cookie
  - redirect 到 `SUB2API_ONYX_REDIRECT_PATH`
- `onyx/backend/onyx/main.py`
  - 挂载 sub2api router
- `onyx/backend/onyx/server/auth_check.py`
  - 将 `/sub2api/exchange` 标记为 public endpoint，该路由自身通过 launch token + server-to-server exchange 完成安全校验
- `onyx/backend/tests/unit/onyx/server/sub2api/test_api.py`
  - 覆盖成功 exchange、自动创建/复用用户、inactive 用户拒绝、非 web login 用户拒绝、空 token、集成关闭、public endpoint 白名单
- `onyx/backend/tests/unit/onyx/server/sub2api/test_client.py`
  - 覆盖 shared secret 调用、配置缺失、401/409/503 上游错误映射
- `onyx/backend/tests/external_dependency_unit/db/test_sub2api_user_credentials.py`
  - 通过 Docker Postgres + Alembic schema 验证 credential upsert 覆盖旧 key

当前聊天与图片生成的代码层接入已完成：

- 聊天 LLM：`get_llm_for_persona(...)` 在传入 `db_session` 时优先使用用户级 sub2api credential。
- Image Generation：`construct_tools(...)` 构造 ImageGenerationTool 时优先使用用户级 sub2api credential。

已验证：

- LLM、Image Generation、sub2api client/api、DB credential helper 相关 19 个测试一起运行通过。
- 2026-05-02 复跑通过：
  - `go -C sub2api/backend test -p=1 -tags unit ./internal/service -run TestOnyxLaunchService`
  - `go -C sub2api/backend test -p=1 -tags unit ./internal/handler -run TestOnyxHandler`
  - `go -C sub2api/backend test -p=1 ./cmd/server -run TestDoesNotExist`
  - `npm run test:run -- src/components/layout/__tests__/AppSidebar.spec.ts src/api/__tests__/onyx.spec.ts`
  - `npm run typecheck`
  - `npm run build`
  - `python -m pytest -q backend\tests\unit\onyx\server\sub2api\test_api.py backend\tests\unit\onyx\server\sub2api\test_client.py backend\tests\unit\onyx\llm\test_sub2api_user_credentials.py backend\tests\unit\onyx\tools\test_sub2api_image_generation_config.py`
  - `python -m pytest -q backend\tests\external_dependency_unit\db\test_sub2api_user_credentials.py`
  - `python -m py_compile ...`
  - `ruff check ...`

### 5. 当前真正剩余工作

- 端到端本地联调仍未执行：
  - sub2api 管理后台配置 Onyx 地址、启用菜单、配置 shared secret。
  - sub2api 用户创建符合条件的 API Key。
  - 点击 sub2api Sidebar Onyx 菜单。
  - 验证 Onyx 自动登录/自动创建用户/写入用户级 credential。
  - 验证 Onyx 文本聊天使用当前用户 sub2api API Key。
  - 验证 Onyx Image Generation 使用当前用户 sub2api API Key 和 image model。
  - 换另一个 sub2api 用户重复验证用户级 API Key 隔离。
- 生产部署/容器联调仍未执行：
  - 按 `onyx/docs/local-deployment-notes.md` 或实际部署方式重建 Onyx web/api 相关服务。
  - 确认 Onyx 容器访问宿主机 sub2api 时的 `SUB2API_BASE_URL`，Windows Docker 通常使用 `host.docker.internal`。

---

## 当前对下一步的判断

代码层主链路已经基本补齐：sub2api launch/exchange、sub2api Sidebar、Onyx exchange/自动登录、用户级 credential、聊天 LLM、Image Generation 都已有实现与目标测试覆盖。

下一阶段最窄的合理扩展是端到端联调：

1. 确认 sub2api 和 Onyx 服务都能本地启动。
2. 配置两边 shared secret、Onyx base URL、默认模型。
3. 使用浏览器验证 Sidebar 菜单、跳转、自动登录、聊天和生图 API Key 隔离。

---

## 关键文件清单

### 已有核心实现

- `sub2api/backend/internal/service/onyx_launch_service.go`
- `sub2api/backend/internal/service/onyx_launch_service_test.go`
- `sub2api/backend/internal/repository/onyx_launch_token_store.go`
- `sub2api/backend/internal/repository/onyx_launch_token_store_test.go`
- `sub2api/backend/internal/repository/onyx_launch_token_store_redis_integration_test.go`
- `sub2api/backend/internal/handler/onyx_handler.go`
- `sub2api/backend/internal/handler/onyx_handler_test.go`
- `sub2api/backend/internal/server/routes/onyx.go`
- `sub2api/frontend/src/api/onyx.ts`
- `sub2api/frontend/src/api/__tests__/onyx.spec.ts`
- `sub2api/frontend/src/components/layout/AppSidebar.vue`
- `sub2api/frontend/src/components/layout/__tests__/AppSidebar.spec.ts`
- `sub2api/frontend/src/utils/featureFlags.ts`
- `sub2api/frontend/src/i18n/locales/zh.ts`
- `sub2api/frontend/src/i18n/locales/en.ts`
- `sub2api/docs/onyx-integration-development-plan.md`
- `onyx/backend/onyx/server/sub2api/models.py`
- `onyx/backend/onyx/server/sub2api/client.py`
- `onyx/backend/onyx/server/sub2api/api.py`
- `onyx/backend/onyx/db/sub2api_user_credentials.py`
- `onyx/backend/onyx/db/models.py`
- `onyx/backend/alembic/versions/4f2b7c8d9e10_add_sub2api_user_credential.py`
- `onyx/backend/onyx/llm/factory.py`
- `onyx/backend/onyx/chat/process_message.py`
- `onyx/backend/onyx/tools/tool_constructor.py`
- `onyx/backend/onyx/llm/litellm_singleton/__init__.py`
- `onyx/backend/tests/unit/onyx/llm/conftest.py`
- `onyx/backend/tests/unit/onyx/llm/test_sub2api_user_credentials.py`
- `onyx/backend/tests/unit/onyx/tools/test_sub2api_image_generation_config.py`
- `onyx/backend/tests/unit/onyx/server/sub2api/test_client.py`
- `onyx/backend/tests/unit/onyx/server/sub2api/test_api.py`
- `onyx/backend/tests/external_dependency_unit/db/test_sub2api_user_credentials.py`

### Claude Code 环境配置

- `C:\Users\jax\.claude\settings.json`
- `C:\Users\jax\.claude\hooks\worktree-create.ps1`
- `C:\Users\jax\.claude\hooks\worktree-remove.ps1`
- `e:\Study\AI\claude\.claude\rules\sub-agent-rule.mdc`

---

## 下次继续时可直接复用的测试命令

在 Windows 当前环境下，建议继续使用下面这种方式跑 Go 目标测试，避免临时目录与资源问题：

```bash
GOMAXPROCS=1 GOTMPDIR="E:/Study/AI/claude/.tmp/go-build" GOCACHE="E:/Study/AI/claude/.tmp/gocache" go -C "E:/Study/AI/claude/sub2api/backend" test -p=1 -tags unit ./internal/service -run TestOnyxLaunchService_...
```

本轮新增验证命令：

```bash
GOMAXPROCS=1 GOTMPDIR="E:/Study/AI/claude/.tmp/go-build" GOCACHE="E:/Study/AI/claude/.tmp/gocache" go -C "E:/Study/AI/claude/sub2api/backend" test -p=1 -tags unit ./internal/handler -run TestOnyxHandler
GOMAXPROCS=1 GOTMPDIR="E:/Study/AI/claude/.tmp/go-build" GOCACHE="E:/Study/AI/claude/.tmp/gocache" go -C "E:/Study/AI/claude/sub2api/backend" test -p=1 -tags unit ./internal/repository -run TestOnyxLaunchTokenStore_RedisError
GOMAXPROCS=1 GOTMPDIR="E:/Study/AI/claude/.tmp/go-build" GOCACHE="E:/Study/AI/claude/.tmp/gocache" go -C "E:/Study/AI/claude/sub2api/backend" test -v -p=1 -tags redis_integration ./internal/repository -run TestOnyxLaunchTokenStore_RedisIntegration
GOMAXPROCS=1 GOTMPDIR="E:/Study/AI/claude/.tmp/go-build" GOCACHE="E:/Study/AI/claude/.tmp/gocache" go -C "E:/Study/AI/claude/sub2api/backend" test -p=1 ./cmd/server -run TestDoesNotExist
cd E:/Study/AI/claude/sub2api/frontend
cmd /c npm run test:run -- src/components/layout/__tests__/AppSidebar.spec.ts src/api/__tests__/onyx.spec.ts
cmd /c npm run typecheck
cd E:/Study/AI/claude/onyx
cmd /c .venv311\Scripts\python.exe -m pytest -q backend\tests\unit\onyx\server\sub2api\test_client.py
cmd /c .venv311\Scripts\python.exe -m pytest -q backend\tests\unit\onyx\server\sub2api\test_api.py backend\tests\unit\onyx\server\sub2api\test_client.py
cmd /c .venv311\Scripts\python.exe -m py_compile backend\onyx\server\sub2api\api.py backend\onyx\server\sub2api\client.py backend\onyx\server\sub2api\models.py backend\onyx\db\sub2api_user_credentials.py backend\alembic\versions\4f2b7c8d9e10_add_sub2api_user_credential.py backend\onyx\main.py backend\onyx\server\auth_check.py
cmd /c .venv311\Scripts\python.exe -m ruff check backend\onyx\server\sub2api backend\onyx\db\sub2api_user_credentials.py backend\tests\unit\onyx\server\sub2api backend\tests\external_dependency_unit\db\test_sub2api_user_credentials.py backend\alembic\versions\4f2b7c8d9e10_add_sub2api_user_credential.py backend\onyx\main.py backend\onyx\server\auth_check.py
```

已完成的 Docker Postgres 验证：

```bash
docker run --name onyx-sub2api-test-postgres -e POSTGRES_PASSWORD=password -e POSTGRES_DB=postgres -p 5432:5432 -d postgres:16-alpine
cd E:/Study/AI/claude/onyx/backend
cmd /c ..\.venv311\Scripts\python.exe -m alembic -c alembic.ini upgrade heads
cd E:/Study/AI/claude/onyx
cmd /c .venv311\Scripts\python.exe -m pytest -q backend\tests\external_dependency_unit\db\test_sub2api_user_credentials.py
```

该测试已通过。当前 Docker Postgres 容器名为 `onyx-sub2api-test-postgres`，仍可继续复用。

Redis integration 已完成验证：

```bash
docker run --name sub2api-redis-test -p 6379:6379 -d redis:7-alpine
GOMAXPROCS=1 GOTMPDIR="E:/Study/AI/claude/.tmp/go-build" GOCACHE="E:/Study/AI/claude/.tmp/gocache" ONYX_REDIS_ADDR="127.0.0.1:6379" go -C "E:/Study/AI/claude/sub2api/backend" test -v -p=1 -tags redis_integration ./internal/repository -run TestOnyxLaunchTokenStore_RedisIntegration
```

该测试已通过。当前 Docker Redis 容器名为 `sub2api-redis-test`，仍可继续复用。

---

## 建议的下次起手点

下次继续时，建议直接从这里开始：

- 启动 sub2api 与 Onyx，配置 shared secret 和 Onyx base URL。
- 使用浏览器验证 Sidebar 菜单、跳转、Onyx 自动登录、聊天与 Image Generation 的用户级 API Key 隔离。

这样最不容易跳步，也最贴合当前代码状态。
