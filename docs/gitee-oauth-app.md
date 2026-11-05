# Gitee（码云）OAuth 接入文档

NexusAI 支持「使用码云（Gitee）登录」，通过 Gitee OAuth App 实现授权码登录。
本文说明如何在 Gitee 申请 OAuth 应用、如何配置到项目，以及代码里的接入方式。

> Gitee 与已接入的 GitHub 同构，个人开发者即可申请、免费（区别于微信需企业认证）。

## 1. 申请 Gitee OAuth App

1. 登录 [gitee.com](https://gitee.com) → 右上角头像 → **设置** → 左侧 **第三方应用**。
   直达：<https://gitee.com/oauth/applications>

2. 点 **创建应用**，填表：

   | 字段 | 填写内容 |
   |---|---|
   | 应用名称 | `NexusAI` |
   | 应用主页 | `http://localhost:3000` |
   | 应用回调地址 | **`http://localhost:8080/auth/gitee/callback`** |
   | 应用描述 / Logo | 可选 |

3. **权限（scope）**：勾选 `user_info`（用户信息）和 `emails`（读邮箱）即可，其余不用。

4. 提交后拿到：
   - **Client ID**：公开，可放前端。
   - **Client Secret**：机密，只显示一次，务必保存。

## 2. 回调地址

回调地址必须与代码里 `NEXUS_GITEE_REDIRECT_URI` **完全一致**。NexusAI 本地端口：

| 服务 | 端口 |
|---|---|
| go-zero 后端 API | `8080` |
| Web 用户端（Next.js） | `3000` |
| Admin 管理后台 | `8000` |

go-zero 是认证权威，OAuth 回调由 go-zero 处理，所以填：

```
http://localhost:8080/auth/gitee/callback
```

生产环境改为 `https://你的域名/auth/gitee/callback` 并回 Gitee 后台同步更新。

## 3. 配置到项目

密钥只注入 go-zero 后端（`services/api/.env`，该文件已被 `.gitignore` 忽略）：

```
NEXUS_GITEE_CLIENT_ID=你的_Client_ID
NEXUS_GITEE_CLIENT_SECRET=你的_Client_Secret
NEXUS_GITEE_REDIRECT_URI=http://localhost:8080/auth/gitee/callback
```

环境变量在 `services/api/nexus.go` 里注入到 `config.Config.Gitee`。

## 4. OAuth 授权码流程

1. Web 端点「使用 Gitee 登录」→ `/api/auth/gitee/login` → 后端 302 到：
   `https://gitee.com/oauth/authorize?client_id=…&redirect_uri=…&response_type=code&scope=user_info+emails&state=随机值`
   （同时下发一次性 `oauth_state` Cookie，用于防 CSRF。）
2. Gitee 用户授权后，302 回 `回调地址?code=xxx&state=xxx`。
3. 后端 POST `https://gitee.com/oauth/token`（`grant_type=authorization_code` + code/client_id/client_secret/redirect_uri）换 `access_token`。
4. 后端 GET `https://gitee.com/api/v5/user?access_token=…` 拉用户 `id`/`login`/`email`。
5. 按 `oauth_provider='gitee'` + `oauth_subject=<id>` 建号/复用本地 `users`，签 JWT，跳回 `http://localhost:3000/auth/callback?token=…`。

## 5. Gitee 特有的坑（代码已处理）

- **换 token 必须带 `User-Agent`**，否则 Gitee 返回 403。
- **用户接口用 `access_token` 查询参数鉴权**（不是 `Authorization: Bearer`）。
- **`/user` 的 `email` 可能是字面量「未公开邮箱」**（用户隐藏邮箱时），代码会回落到
  `GET /api/v5/emails`，取 `state=confirmed` 且 `scope` 含 `primary` 的项。
- **`id` 可能是数字也可能是字符串**，代码用 `json.RawMessage` 按字符串兼容。

## 6. 安全注意事项

- `Client Secret` 不进 Git、前端或日志。
- `state` 防 CSRF：随机生成、回调校验、一次性失效。
- 授权码 `code` 一次性，换 `access_token` 后失效。
- 生产环境回调必须用 HTTPS。

## 7. 代码位置

| 文件 | 职责 |
|---|---|
| `services/api/internal/auth/oauth.go` | `OAuthProvider` 接口 + `OAuthUser` |
| `services/api/internal/auth/gitee.go` | `GiteeOAuth` 实现（授权码流程 + Gitee 特有处理） |
| `services/api/internal/auth/github.go` | `GitHubOAuth` 实现（同接口） |
| `services/api/internal/logic/oauth_callback.go` | 通用回调流程（换 token → 拉用户 → 建号/复用 → 签 JWT） |
| `services/api/internal/logic/giteeloginlogic.go` / `giteecallbacklogic.go` | Gitee 薄包装 |
| `services/api/internal/handler/giteeloginhandler.go` / `giteecallbackhandler.go` | state Cookie + 302 跳转 |

身份存储：`users.oauth_provider` + `users.oauth_subject`（唯一约束 `(provider, subject)`），
加新的三方登录不用再为每平台加一列。

## 参考

- Gitee 官方 OAuth 文档：<https://gitee.com/api/v5/oauth_doc>
- 获取授权用户资料 API：<https://docs.gitcode.com/docs/apis/get-api-v-5-user>
