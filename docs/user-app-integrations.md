# 普通用户应用接入

Sub2API 向已注册的应用签发当前用户的一次性票据。应用后端交换票据后，可以按注册权限列出该用户的脱敏 Key、查询每把 Key 的模型，或代用户发起聊天。Key 明文由 Sub2API 网关内部解析。

## 注册应用

`USER_APP_INTEGRATIONS_CONFIG_PATH` 指向 JSON 数组文件。配置文件权限至多为 `0640`。路径缺失或文件不存在时，注册表为空。应用配置包含 `app_id`、`enabled`、`launch_url`、`secret_env`、`scopes`。回调必须是固定 HTTPS URL。为每个启用的应用配置独立的高熵密钥，并通过 `secret_env` 指定环境变量名。可申请 `keys:read`、`models:read` 和 `chat:invoke` 中所需的权限。

设置 `CHAT_INTEGRATION_APP_ID` 可将现有聊天台入口指向一个已启用的应用。缺失时使用 LobeHub。应用注册及入口配置在 Sub2API 启动时载入。

## 请求约定

下表路径均以 `/api/v1/app-integrations/:app_id` 开头。

| 路径 | 调用方式与结果 |
| --- | --- |
| `POST /launch` | 使用 Sub2API 用户登录凭据，返回 `data.launch_url` 与 `data.ticket`。前端把 `ticket` 作为表单字段 POST 到已注册回调。票据有效期为 60 秒，仅可兑现一次。 |
| `POST /exchange` | 应用后端携带 `X-Sub2API-App-Secret`，提交 JSON 字段 `ticket`。成功响应包含 `user_id`、`email`、`username`、`grant_token`、`expires_in` 和 `scopes`。 |
| `GET /keys` | `keys:read`。使用 `page` 与 `page_size` 分页，默认每页 50 条，上限 100 条。返回 `items`、`page`、`page_size`、`total`。列表项提供 Key ID、名称、脱敏标识、所属分组、平台、状态及 `selectable`。 |
| `GET /keys/:key_id/models` | `models:read`。返回标准模型目录，`X-Sub2API-Key-Platform` 提供对应的分组平台。 |
| `POST /chat/completions` | `chat:invoke`。请求使用原生 Chat Completions 格式，并带 `X-Sub2API-Key-ID`。网关按所选 Key 鉴权、路由和计费。 |
| `POST /revoke` | 退出时撤回当前授权。重复撤回返回 204。 |

交换后的请求由后端同时携带应用密钥和 `Authorization: Bearer <grant_token>`。授权最长八小时，绑定应用和用户。授权无效返回 401，应用未启用、权限不足或指定的 Key 不可用返回 403。根据 Key 设置执行原有网关限制，来源 IP 受限的 Key 在代理链无法验证原始 IP 时不可选。

应用应校验回调来源和现有会话身份。浏览器只保存 Key ID 与模型 ID 等界面偏好，重新进入后根据实时目录验证。应用作用域授权、票据和独立密钥均使用服务端或短期传输，不写入 URL、日志与浏览器存储。
