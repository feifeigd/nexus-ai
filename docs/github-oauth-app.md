# GitHub OAuth App 申请教程

NexusAI 要支持「使用 GitHub 登录」（三方登录/OAuth），需要先在 GitHub 上注册一个
**OAuth App**，拿到 `Client ID` 和 `Client Secret`。本文是完整申请步骤，并说明这两个
密钥在项目里怎么用、回调地址怎么填。

## 1. 概述

- **OAuth App** 用于「用 GitHub 账号登录第三方应用」这种场景：用户在 GitHub 授权，
  你的后端拿到授权后换取 GitHub 用户信息（登录名、邮箱），再映射成本地账号并签发自己的 JWT。
- 注册后得到两个值：
  - **Client ID**：公开标识，可放在前端。
  - **Client Secret**：机密，只能放在后端，绝不能提交到 Git 或下发到浏览器。

> 注：GitHub 还有「GitHub App」（权限更细、可装到组织/仓库），但「用 GitHub 登录」
> 用 OAuth App 就够了，配置更简单。

## 2. 申请步骤

1. 登录 GitHub → 点右上角头像 → **Settings**。
2. 左侧最底部 **Developer settings** → **OAuth Apps** → **New OAuth App**。

   直达链接：<https://github.com/settings/developers>

3. 填写表单：

   | 字段 | 填写内容 |
   |---|---|
   | Application name | 应用名，例如 `NexusAI` |
   | Homepage URL | 主页地址；本地开发可填 `http://localhost:3000` |
   | Authorization callback URL | **回调地址**，见下方说明 |

4. 点 **Register application** 完成注册。

## 3. 回调地址怎么填（NexusAI 项目）

回调地址是 GitHub 授权完成后跳回你后端的那个端点，必须与代码里配置的 `redirect_uri`
**完全一致**（含协议、域名、端口、路径）。

NexusAI 本地开发各服务端口：

| 服务 | 端口 |
|---|---|
| go-zero 后端 API | `8080` |
| Web 用户端（Next.js） | `3000` |
| Admin 管理后台 | `8000` |

go-zero 是认证权威，OAuth 回调应由 go-zero 处理，所以回调地址填：

```
http://localhost:8080/auth/github/callback
```

- GitHub 允许开发期使用 `http://localhost`；生产环境要改成 `https://你的域名/auth/github/callback`
  并回 GitHub 后台同步更新。
- 如果之前填成了别的地址，去 **OAuth Apps → 你的应用** 里改（可随时修改，也可添加多个回调地址）。

## 4. 拿到 Client ID / Client Secret

- 注册成功后页面会直接显示 **Client ID**。
- 点 **Generate a new client secret** 生成 **Client Secret**。
- **Client Secret 只完整显示这一次**，请立刻保存；以后只能重新生成。

## 5. 配置到 NexusAI 项目

按项目约定，密钥只注入 go-zero 后端，不下发前端、不提交 Git。两种方式：

1. **环境变量**（推荐）：在 `services/api/.env` 里写（该文件已被 `.gitignore` 忽略）：

   ```
   NEXUS_GITHUB_CLIENT_ID=Ov23liIvffkcMPZLchVw
   NEXUS_GITHUB_CLIENT_SECRET=你的_Client_Secret
   NEXUS_GITHUB_REDIRECT_URI=http://localhost:8080/auth/github/callback
   ```

2. 或写进 `services/api/etc/nexus-api.yaml`（但别提交真实密钥，用环境变量覆盖更安全）。

## 6. OAuth 授权码流程（这两个 key 干什么用）

1. 前端跳转 GitHub 授权页：
   `https://github.com/login/oauth/authorize?client_id=CLIENT_ID&redirect_uri=回调地址&scope=user:email&state=随机值`
2. GitHub 重定向回 `回调地址?code=xxx&state=xxx`（带回一次性授权码）。
3. 后端用 `code` + `client_id` + `client_secret`，POST `https://github.com/login/oauth/access_token`
   换 `access_token`。
4. 用 `access_token` 请求 `https://api.github.com/user`（要邮箱再请求 `https://api.github.com/user/emails`），
   拿到 GitHub 登录名/邮箱。
5. 用 GitHub 身份（`github_id`）关联本地 `users` 表，签发项目自己的 JWT（复用 `auth.SignToken`）。

**scope 说明**：登录 + 邮箱用 `user:email` 即可；如果不需要邮箱，`read:user` 也可。

## 7. 安全注意事项

- `Client Secret` 绝不能进 Git、前端或日志。
- `state` 参数用于防 CSRF，实现时后端生成随机值并用短时效 Cookie 校验。
- 授权码 `code` 是一次性的，换 `access_token` 后立即失效。
- 生产环境回调必须用 HTTPS。
