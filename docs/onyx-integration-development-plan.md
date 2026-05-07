# sub2api 集成 Onyx 开发计划

## 目标

在 sub2api 的 Sidebar 主菜单中新增 Onyx 入口。用户点击后，sub2api 选择该用户第一条有额度的 API Key，通过安全的一次性登录交换流程跳转到 Onyx。Onyx 自动创建或登录对应用户，保存该用户独立的 sub2api API Key，并让 Onyx 聊天 LLM 和 Image Generation 都使用该用户自己的 key，而不是全局管理员 key。

## 已确认需求

- 菜单位置：sub2api Sidebar 主菜单。
- API Key 选择规则：选择当前用户第一条满足以下条件的 key：
  - `status === active`
  - `expires_at` 为空，或晚于当前时间
  - `quota_used < quota`
- `quota = 0` 虽然在 sub2api 现有注释中表示 unlimited，但本次需求明确使用 `quota_used < quota`，因此 `quota = 0` 不算有额度。
- Onyx 端行为：自动登录、自动创建用户、打开已登录页面。
- API Key 用途：Onyx 聊天 LLM 和 Image Generation 都使用该用户自己的 sub2api API Key。
- 每个 sub2api 用户进入 Onyx 后使用自己独立的 API Key。
- URL 只允许携带短期、单次使用 launch token，不允许携带 API Key 明文。
- launch token 默认有效期保持 60 秒。
- sub2api 需要有配置 Onyx 地址的地方。

## 总体架构

采用“sub2api 后端生成一次性 launch token，Onyx 后端消费 token 并写登录 cookie”的方案。

sub2api 前端不直接把 API Key 放到 URL query 中。用户点击 Sidebar 的 Onyx 菜单后，前端请求 sub2api 后端接口。sub2api 后端按用户身份查找第一条符合条件的 API Key，并生成短期、单次使用的 launch token。浏览器跳转到 Onyx 的 exchange endpoint。Onyx 后端调用 sub2api 后端验证并消费 launch token，拿到用户信息和 API Key，然后自动创建或更新 Onyx 用户、保存用户级 sub2api credential、写入 Onyx 登录 cookie，并 redirect 到 Onyx 已登录页面。

launch token 只用于登录交接，不用于后续聊天。后续聊天和生图依赖 Onyx 登录 cookie，以及 Onyx 数据库中加密保存的用户级 sub2api API Key。

## sub2api 后端改动

### `sub2api/backend/internal/service/domain_constants.go`

新增 Onyx 集成相关系统设置 key：

- Onyx 集成是否启用。
- Onyx base URL。
- Onyx Sidebar 菜单 label。
- sub2api 与 Onyx exchange 使用的共享密钥。
- launch token TTL，默认 60 秒。
- Onyx 默认跳转路径，默认 `/chat` 或 Onyx 首页。

这些设置复用现有 settings 系统，不新增数据库表。

### `sub2api/backend/internal/handler/dto/settings.go`

在 `SystemSettings` 和 `PublicSettings` 中增加 Onyx 配置字段。

公开给前端的字段只包含：

- 是否显示 Onyx 菜单。
- Onyx 菜单 label。
- launch API 路径或前端固定调用标识。

不得公开共享密钥。

### `sub2api/backend/internal/service/setting_service.go`

扩展 settings 默认值、读取逻辑和 public settings 注入逻辑。

`GetPublicSettings()` 应返回是否启用 Onyx Sidebar 菜单。

`GetPublicSettingsForInjection()` 也需要把公开配置注入 HTML，避免前端刷新首屏时缺配置。

### `sub2api/backend/internal/handler/admin/setting_handler.go`

在 admin settings 更新接口中允许管理员配置 Onyx 集成参数。

校验规则：

- Onyx base URL 必须是合法 URL。
- launch token TTL 必须为正数，默认 60 秒。
- 共享密钥不能为空，除非 Onyx 菜单未启用。
- 不记录共享密钥明文。
- 不把共享密钥返回给 public settings。

### `sub2api/backend/internal/service/onyx_launch_service.go`

新增 `OnyxLaunchService`，负责生成和消费 Onyx launch token。

核心方法：

- `CreateLaunch(ctx context.Context, userID int64) (*OnyxLaunchResult, error)`
- `ConsumeLaunch(ctx context.Context, token string) (*OnyxLaunchPayload, error)`
- `selectFirstEligibleAPIKey(ctx context.Context, userID int64) (*APIKey, error)`

`selectFirstEligibleAPIKey` 必须严格执行：

- `status === active`
- `expires_at` 为空或晚于当前时间
- `quota_used < quota`

排序规则：

- 按创建时间升序，确保“第一条”稳定。

launch token 设计：

- 默认 60 秒有效。
- 单次使用。
- token payload 绑定 sub2api user id、email、username、api key id。
- token 消费时重新读取 API Key，避免 token 生成后 key 被禁用、过期或额度耗尽。
- token 不包含 API Key 明文。
- 优先使用 Redis 存储一次性 token，因为当前 sub2api router 已传入 `redisClient`。

### `sub2api/backend/internal/handler/onyx_handler.go`

新增用户侧 Onyx handler。

接口：

- `POST /api/v1/onyx/launch`
- `POST /api/v1/onyx/exchange`

`POST /api/v1/onyx/launch`：

- 由 sub2api 登录用户调用。
- 通过 `jwtAuth` 获取当前用户。
- 调用 `OnyxLaunchService.CreateLaunch`。
- 返回跳转 URL，例如 `${onyx_base_url}/api/sub2api/exchange?token=...`。
- URL query 中只允许出现 launch token。

`POST /api/v1/onyx/exchange`：

- 由 Onyx 后端调用。
- 输入 launch token。
- 校验 token 是否存在、未过期、未消费。
- 重新校验 API Key 是否仍符合有额度规则。
- 返回 Onyx 所需用户信息和 API Key。
- 成功后立即消费 token。

返回数据字段：

- sub2api user id。
- email。
- username。
- api key id。
- api key 明文。
- API base URL，指向 sub2api OpenAI-compatible endpoint。
- 推荐文本模型。
- 推荐图片模型。

错误处理：

- 无可用 API Key：409。
- token 无效或过期：401。
- token 已消费：401。
- key 状态变化导致不再可用：409。
- 配置缺失：503。

### `sub2api/backend/internal/handler/handlers.go`

把 `OnyxHandler` 加入 handlers 聚合结构，并完成依赖初始化。

### `sub2api/backend/internal/server/routes/onyx_routes.go`

新增 Onyx route 注册文件。

- `/api/v1/onyx/launch` 使用 `jwtAuth`。
- `/api/v1/onyx/exchange` 使用 launch token 和服务端共享签名校验，不使用普通用户 JWT。

### `sub2api/backend/internal/server/router.go`

在 `registerRoutes` 中挂载 `RegisterOnyxRoutes`。

## sub2api 前端改动

### `sub2api/frontend/src/types/index.ts`

新增 public settings 中 Onyx 菜单配置类型字段：

- `onyx_enabled: boolean`
- `onyx_menu_label: string`
- `onyx_launch_path: string`

### `sub2api/frontend/src/api/onyx.ts`

新增前端 API 方法：

- `launchOnyx(): Promise<{ redirect_url: string }>`

该方法调用：

- `POST /api/v1/onyx/launch`

错误处理：

- 409：提示用户需要先创建一条 active、未过期、未耗尽额度的 API Key。
- 503：提示管理员未启用 Onyx。
- 其他错误：显示通用跳转失败提示。

### `sub2api/frontend/src/components/layout/AppSidebar.vue`

在用户 Sidebar 主菜单中新增 Onyx 菜单项。

位置：

- 放在 API Keys 或 Chat 附近，作为用户主功能入口。

行为：

- 用户点击 Onyx 菜单项时调用 `launchOnyx()`。
- 成功后使用 `window.location.href = redirect_url` 跳转。
- 点击期间显示 loading 状态，避免重复点击生成多个 token。
- 出错时显示 toast。

不修改 Header，因为菜单位置已确认是 Sidebar 主菜单。

### sub2api locale 文件

新增 Onyx 菜单和错误提示文案：

- Onyx
- 正在打开 Onyx
- 没有可用 API Key
- Onyx 尚未配置
- 打开 Onyx 失败

## Onyx 后端改动

### `onyx/backend/alembic/versions/*_sub2api_user_credentials.py`

新增用户级 sub2api credential 表。

表名：

- `sub2api_user_credential`

字段：

- `id`
- `user_id`
- `sub2api_user_id`
- `sub2api_api_key_id`
- `api_key`
- `api_base`
- `text_model_name`
- `image_model_name`
- `created_at`
- `updated_at`
- `last_used_at`

约束：

- `user_id` 外键指向 Onyx user 表。
- `user_id` 唯一，保证每个 Onyx 用户只有一条当前 sub2api credential。
- `api_key` 使用 Onyx 已有 `EncryptedString()` 存储。
- `sub2api_user_id + sub2api_api_key_id` 建索引。

不把 sub2api API Key 存到全局 `LLMProvider` 表，避免不同用户互相覆盖。

### `onyx/backend/onyx/db/models.py`

新增 ORM model：`Sub2APIUserCredential`。

字段与 migration 保持一致。

关系：

- 与 `User` 建立一对一或多对一关系。
- 不强依赖 `LLMProvider`、`ModelConfiguration`、`ImageGenerationConfig`。

### `onyx/backend/onyx/db/sub2api_user_credentials.py`

新增 DB helper。

核心函数：

- `get_sub2api_credential_for_user(db_session: Session, user_id: UUID | str) -> Sub2APIUserCredential | None`
- `upsert_sub2api_credential_for_user(db_session: Session, user: User, payload: Sub2APIExchangePayload) -> Sub2APIUserCredential`
- `delete_sub2api_credential_for_user(db_session: Session, user_id: UUID | str) -> None`

行为：

- upsert 时覆盖旧 API Key。
- 不记录 API Key 明文。
- `updated_at` 每次 exchange 成功后更新。

### `onyx/backend/onyx/server/sub2api/models.py`

定义 Onyx 与 sub2api exchange 相关 Pydantic model。

模型包括：

- `Sub2APILaunchExchangeRequest`
- `Sub2APILaunchExchangeResponse`
- `Sub2APIExchangeUser`
- `Sub2APICredentialPayload`

字段包括：

- token。
- sub2api user id。
- email。
- username。
- api key id。
- api key。
- api base。
- text model name。
- image model name。

### `onyx/backend/onyx/server/sub2api/client.py`

封装 Onyx 调用 sub2api 的 exchange client。

核心函数：

- `exchange_sub2api_launch_token(token: str) -> Sub2APILaunchExchangeResponse`

配置来源：

- sub2api base URL。
- shared secret。
- request timeout。

错误处理：

- sub2api 返回 401：Onyx 返回登录链接已失效。
- sub2api 返回 409：Onyx 返回用户没有可用 API Key。
- sub2api 返回 503：Onyx 返回 sub2api 集成未配置。
- 网络超时：Onyx 返回临时不可用。

### `onyx/backend/onyx/server/sub2api/api.py`

新增 Onyx exchange router。

路由：

- `GET /api/sub2api/exchange?token=...`

处理流程：

1. 校验 token query 存在。
2. 调用 sub2api exchange client 消费 token。
3. 根据返回的 email 查找 Onyx 用户。
4. 如果用户不存在，自动创建 Onyx web login 用户。
5. 如果用户存在但 inactive，拒绝登录。
6. 如果用户存在但 account type 不允许 web login，拒绝登录。
7. upsert 用户级 sub2api credential。
8. 使用 Onyx 现有 cookie auth backend 写登录 cookie。
9. redirect 到 Onyx 已登录页面。

自动创建用户逻辑参考：

- `onyx/backend/onyx/auth/users.py`
- `onyx/backend/onyx/server/saml.py`

必须复用现有用户创建流程，确保 role、verified、password、domain/invite 校验逻辑一致。

### `onyx/backend/onyx/main.py`

新增 sub2api router import，并通过 `include_router_with_global_prefix_prepended` 挂载。

### `onyx/backend/onyx/configs/app_configs.py`

新增环境配置项：

- `SUB2API_INTEGRATION_ENABLED`
- `SUB2API_BASE_URL`
- `SUB2API_EXCHANGE_SECRET`
- `SUB2API_DEFAULT_TEXT_MODEL`
- `SUB2API_DEFAULT_IMAGE_MODEL`
- `SUB2API_ONYX_REDIRECT_PATH`

默认值：

- 集成默认关闭。
- text model 可默认使用 `gpt-5.5`，允许环境变量覆盖。
- image model 可默认使用 `gpt-image-2`，允许环境变量覆盖。

## Onyx 聊天 LLM 用户级 API Key

### `onyx/backend/onyx/llm/factory.py`

修改 `get_llm_for_persona(...)`。

计划行为：

- 当用户存在 sub2api credential 时，优先构造 OpenAI-compatible LLM：
  - provider 使用 `openai`
  - api_key 使用用户级 credential
  - api_base 使用 sub2api OpenAI-compatible base URL
  - model 使用用户级 text model name
- 当用户不存在 sub2api credential 时，保持现有全局 provider 逻辑。

需要新增 helper：

- `get_user_sub2api_llm_config(db_session: Session, user: User, model_override: str | None) -> LLMConfig | None`

当前 `get_llm_for_persona` 没有 `db_session` 参数，因此采用最小侵入改法：

- 给 `get_llm_for_persona` 增加 `db_session: Session | None = None` 参数。
- 调用点传入已有 `db_session`。
- 未传入 `db_session` 时保持现有行为。

### `onyx/backend/onyx/chat/process_message.py`

调用 `get_llm_for_persona` 时传入当前 `db_session`。

错误处理：

- 用户 credential 缺失：走现有默认 provider。
- 用户 credential 存在但解密失败或配置非法：抛出明确错误，提示重新从 sub2api 进入 Onyx。

## Onyx Image Generation 用户级 API Key

### `onyx/backend/onyx/tools/tool_constructor.py`

修改 ImageGenerationTool 构造逻辑。

计划行为：

- 构造 ImageGenerationTool 时，优先读取当前用户 sub2api credential。
- 如果存在用户级 credential：
  - provider 使用 `openai`
  - model 使用 credential.image_model_name
  - api_key 使用 credential.api_key
  - api_base 使用 credential.api_base
  - api_version 为空
  - deployment_name 使用 image model name
- 如果不存在用户级 credential：
  - 保持现有全局默认 Image Generation config。

新增 helper：

- `_get_user_sub2api_image_generation_config(...)`

保留 `_get_image_generation_config(...)` 作为全局 fallback。

### `onyx/backend/onyx/tools/tool_implementations/images/image_generation_tool.py`

原则上不需要改动核心生图逻辑。当前工具已经会把 `self.model`、`api_key`、`api_base` 传给 provider；只要构造时换成用户级 config 即可。

## 安全策略

- 不把 sub2api API Key 放在浏览器 URL query、localStorage 新字段、iframe URL 或页面可见参数中。
- URL 只允许携带短期、单次使用 launch token。
- launch token 默认 60 秒有效。
- launch token 成功消费后立即失效。
- launch token 绑定 user id 与 api key id。
- 消费时重新校验 key 状态、过期时间、额度。
- 不能从 token 反推出 API Key。
- Onyx 使用加密字段保存 API Key。
- 不在日志中输出 API Key 明文。
- 不在普通 API response 中返回 API Key 明文。
- Onyx 调 sub2api exchange endpoint 时使用 shared secret 或签名 header。
- sub2api 校验调用方可信。
- exchange endpoint 不允许浏览器直接用 token 换 API Key。

## 测试计划

### sub2api 后端测试

新增测试：

- `sub2api/backend/internal/service/onyx_launch_service_test.go`
- `sub2api/backend/internal/handler/onyx_handler_test.go`

覆盖：

1. 用户有多条 key 时选择第一条符合条件的 key。
2. `status !== active` 的 key 被排除。
3. `expires_at` 早于当前时间的 key 被排除。
4. `quota_used >= quota` 的 key 被排除。
5. `quota = 0` 按本需求被排除。
6. 无可用 key 返回 409。
7. launch token 过期后不能 exchange。
8. launch token 成功 exchange 后不能重复使用。
9. exchange 时 key 状态变为 inactive，应失败。
10. public settings 不返回 secret。

### sub2api 前端测试

覆盖：

1. Sidebar 显示 Onyx 菜单。
2. 点击 Onyx 菜单会调用 `/api/v1/onyx/launch`。
3. 成功后跳转到返回的 `redirect_url`。
4. 409 时显示没有可用 API Key提示。
5. 未启用 Onyx 时不显示菜单。

### Onyx 后端测试

新增测试：

- `onyx/backend/tests/integration/tests/sub2api/test_sub2api_exchange.py`
- `onyx/backend/tests/unit/onyx/llm/test_sub2api_user_credentials.py`
- `onyx/backend/tests/unit/onyx/tools/test_sub2api_image_generation_config.py`

覆盖：

1. exchange token 成功时自动创建 Onyx 用户。
2. 已存在用户时直接登录，不重复创建。
3. inactive 用户不能登录。
4. exchange 成功后写入用户级 sub2api credential。
5. 同一用户再次 exchange 会覆盖旧 key。
6. 登录 response 设置 Onyx auth cookie。
7. 聊天 LLM 优先使用用户级 sub2api key。
8. ImageGenerationTool 优先使用用户级 sub2api key。
9. 无用户级 credential 时保持全局默认 provider fallback。
10. token 过期、无效、sub2api 返回 409 时返回友好错误。

## 本地联调步骤

1. 启动 sub2api。
2. 启动 Onyx。
3. 在 sub2api 管理后台配置 Onyx 地址、启用 Onyx 菜单、配置 exchange secret。
4. 在 Onyx 环境变量中配置相同 exchange secret 和 sub2api base URL。
5. 在 sub2api 创建至少一条符合条件的 API Key。
6. 确认该 key 状态为 active、未过期、`quota_used < quota`。
7. 点击 sub2api Sidebar 的 Onyx 菜单。
8. 浏览器跳转到 Onyx。
9. Onyx 自动进入已登录页面。
10. 在 Onyx 发起文本聊天，确认请求走当前用户 sub2api API Key。
11. 在 Onyx 发起 Image Generation，确认请求也走当前用户 sub2api API Key。
12. 换另一个 sub2api 用户重复验证，确认 Onyx 使用不同 API Key。

## 部署配置

### sub2api 系统设置

需要在 sub2api 管理后台提供配置位置：

- Onyx integration enabled。
- Onyx base URL，例如 `http://localhost:3000`。
- Onyx menu label，例如 `Onyx`。
- Onyx exchange secret。
- launch token TTL，默认 60 秒。

### Onyx 环境变量

需要配置：

- `SUB2API_INTEGRATION_ENABLED=true`
- `SUB2API_BASE_URL=http://localhost:<sub2api-port>`
- `SUB2API_EXCHANGE_SECRET=<same-secret>`
- `SUB2API_DEFAULT_TEXT_MODEL=gpt-5.5`
- `SUB2API_DEFAULT_IMAGE_MODEL=gpt-image-2`
- `SUB2API_ONYX_REDIRECT_PATH=/chat`

如果 Onyx 容器访问宿主机上的 sub2api，Windows Docker 环境通常使用 `host.docker.internal`。

## 实施清单

1. 修改 `sub2api/backend/internal/service/domain_constants.go`，新增 Onyx 集成 settings key。
2. 修改 `sub2api/backend/internal/handler/dto/settings.go`，新增 Onyx public/system settings DTO 字段。
3. 修改 `sub2api/backend/internal/service/setting_service.go`，加入 Onyx settings 默认值、读取、public 注入逻辑。
4. 修改 `sub2api/backend/internal/handler/admin/setting_handler.go`，加入 Onyx settings 更新与校验。
5. 新增 `sub2api/backend/internal/service/onyx_launch_service.go`，实现 launch token 创建、Redis 存储、单次消费、API Key 筛选。
6. 新增 `sub2api/backend/internal/service/onyx_launch_service_test.go`，覆盖 key 筛选、过期、单次消费、无可用 key 等场景。
7. 新增 `sub2api/backend/internal/handler/onyx_handler.go`，实现 `/api/v1/onyx/launch` 和 `/api/v1/onyx/exchange` handler。
8. 新增 `sub2api/backend/internal/handler/onyx_handler_test.go`，覆盖 launch 和 exchange HTTP 行为。
9. 修改 `sub2api/backend/internal/handler/handlers.go`，把 Onyx handler 加入 handlers 聚合结构。
10. 新增 `sub2api/backend/internal/server/routes/onyx_routes.go`，注册 Onyx routes。
11. 修改 `sub2api/backend/internal/server/router.go`，挂载 Onyx routes。
12. 修改 `sub2api/frontend/src/types/index.ts`，新增 Onyx public settings 类型字段。
13. 新增 `sub2api/frontend/src/api/onyx.ts`，封装 `launchOnyx()`。
14. 修改 `sub2api/frontend/src/components/layout/AppSidebar.vue`，新增 Sidebar Onyx 主菜单项和点击跳转逻辑。
15. 修改 sub2api locale 文件，新增 Onyx 菜单和错误提示文案。
16. 新增 sub2api 前端测试，覆盖菜单显示、点击跳转和错误提示。
17. 新增 Onyx Alembic migration，创建 `sub2api_user_credential` 表。
18. 修改 `onyx/backend/onyx/db/models.py`，新增 `Sub2APIUserCredential` ORM model。
19. 新增 `onyx/backend/onyx/db/sub2api_user_credentials.py`，实现用户级 credential get/upsert/delete helper。
20. 新增 `onyx/backend/onyx/server/sub2api/models.py`，定义 exchange 请求和响应模型。
21. 新增 `onyx/backend/onyx/server/sub2api/client.py`，封装 Onyx 调用 sub2api exchange 接口。
22. 新增 `onyx/backend/onyx/server/sub2api/api.py`，实现 Onyx `/api/sub2api/exchange` endpoint、自动创建用户、写 cookie、redirect。
23. 修改 `onyx/backend/onyx/main.py`，导入并挂载 sub2api router。
24. 修改 `onyx/backend/onyx/configs/app_configs.py`，新增 sub2api integration 环境配置。
25. 修改 `onyx/backend/onyx/llm/factory.py`，让聊天 LLM 在存在用户级 sub2api credential 时优先使用用户级 API Key。
26. 修改 `onyx/backend/onyx/chat/process_message.py`，调用 `get_llm_for_persona` 时传入 `db_session`。
27. 修改 `onyx/backend/onyx/tools/tool_constructor.py`，让 ImageGenerationTool 在存在用户级 sub2api credential 时优先使用用户级 API Key 和 image model。
28. 新增 Onyx exchange 集成测试，覆盖自动创建用户、登录 cookie、credential upsert。
29. 新增 Onyx LLM 构造测试，覆盖用户级 credential 优先与全局 fallback。
30. 新增 Onyx ImageGenerationTool 构造测试，覆盖用户级 image credential 优先与全局 fallback。
31. 运行 sub2api 后端测试，修复失败。
32. 运行 sub2api 前端测试和类型检查，修复失败。
33. 运行 Onyx 后端相关测试，修复失败。
34. 重新构建 sub2api 前端和后端。
35. 按 `onyx/docs/local-deployment-notes.md` 重新构建并部署 Onyx web/api 相关服务。
36. 使用浏览器或 `playwright-cli` 验证 sub2api Sidebar 出现 Onyx 菜单。
37. 点击 Onyx 菜单验证跳转、Onyx 自动登录、自动创建用户。
38. 在 Onyx 聊天页验证文本聊天使用当前用户 sub2api API Key。
39. 在 Onyx Image Generation 验证生图使用当前用户 sub2api API Key。
40. 使用另一个 sub2api 用户重复验证，确认两个用户在 Onyx 中使用各自独立 API Key。
