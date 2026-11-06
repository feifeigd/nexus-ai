# SSH 远程命令执行设计

> 本文档是 Admin 机器列表「SSH 远程命令执行」功能的**设计源头**。任何改动（字段、接口、白名单、鉴权、审计等）都必须同步更新本文档。

## 目标

在 Admin 的机器列表上，对单台机器执行一条 SSH 命令，回显 stdout/stderr，并写入审计日志。当前为「单次命令执行」（Option A），完整 Web 交互终端（xterm.js + WebSocket）留待以后（Option B）。

## 设计决策

| 决策 | 结论 | 说明 |
| --- | --- | --- |
| 实现层次 | 单次命令执行（A） | 交互终端（B）以后再做 |
| SSH 用户名 | 固定 `root` | 与 `root_public_key` 字段语义一致 |
| 私钥存放 | `machines` 表新增 `ssh_private_key` 字段 | 用户决策；见下方安全约束 |
| 权限 | 仅 `admin` 可执行 | `AdminOnlyMiddleware` 强制 |
| 命令范围 | 白名单 + 逐步开放 | 命中前缀且拒绝 shell 元字符 |
| 私钥读取 | **只写不读** | API 响应与前端永不返回/显示私钥 |

## 架构与数据流

```
浏览器(Admin) ──POST /machines/:id/exec──▶ go-zero ──SSH(root@privateAddress)──▶ 目标机器
     ▲                                        │
     └──────────── stdout/stderr ──────────────┘
```

- 浏览器**不直接连接目标机器**；SSH 全部由 go-zero 完成，符合「go-zero 为鉴权/校验/审计权威」的原则。
- 执行链路：按 id 取机器 → 校验命令白名单 → 检查私钥是否配置 → 调 `sshx.Runner` → 写审计 → 返回输出。

## 数据模型

`machines` 表新增：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `ssh_private_key` | TEXT（可空） | root 登录私钥（PEM 多行）。只写不读，见安全约束 |

更新语义：`ssh_private_key` 在 `Update` 中**留空表示保持不变**（不是清空），因为私钥只写不读、前端无法回填旧值；其它可选字段仍保持「留空即清空」。要更换/清除私钥，传新值即可（彻底清除需删除机器后重建，作为后续增量可加「清除私钥」入口）。

迁移：`015_machines_ssh.sql`（PostgreSQL 加列）；SQLite 走 Ent AutoMigrate。

## 命令白名单

- 配置项 `SSH.AllowedCommands`（nexus-api.yaml 列表，环境变量可覆盖）。
- 首版命令集（只读诊断，逐步开放）：

```
uptime
df
free
hostname
uname
systemctl status
docker ps
kubectl get
tail
```

- 校验规则：命令必须命中某个前缀；且**拒绝 shell 元字符**（`;` `&&` `||` `|` `>` `<` `` ` `` `$(` 等），防止拼接绕过。

## 接口

- `POST /machines/:id/exec`（`jwt: Auth` + `AdminOnlyMiddleware`）
- 入参：`{ "command": "..." }`
- 出参：`{ "stdout": "...", "stderr": "...", "exitCode": 0 }`
- 错误：机器不存在 / 私钥未配置 / 命令不允许 / SSH 连接失败；不外泄内部错误。

## SSH 执行服务

- 新包 `internal/sshx`，接口 `Runner`：

```go
type Result struct {
    Stdout   string
    Stderr   string
    ExitCode int // 命令退出码；连接失败时无意义
}

type Runner interface {
    // Exec 在 host 上以 privateKey 登录执行 command。
    // err 非 nil 表示未能执行（解析私钥 / 连接 / 会话失败），Result 无意义；
    // 命令退出码非零时 err 仍为 nil，由 Result.ExitCode 表达。
    Exec(ctx context.Context, host string, privateKey []byte, command string) (Result, error)
}
```

- 实现用 `golang.org/x/crypto/ssh`：root 登录、密钥认证、连接 + 命令双超时（默认 30s）。
- 测试用内存内 `crypto/ssh` 测试服务器（测试密钥对）跑通真链路。

## 安全约束（必须遵守）

1. 私钥**只写不读**：`types.Machine` 响应不含私钥；`machineResponse` 不映射私钥；前端表格/详情不显示。
2. 私钥**不落日志**：审计只记「机器 id + 命令」，不记私钥内容。
3. 仅 `admin` 可执行；执行动作写审计 `run_ssh_command`。
4. 命令走白名单校验，不自由执行任意命令。
5. 私钥字段当前**明文落库**；后续可选加密（用 `NEXUS_AUTH_SECRET` AES-GCM）作为独立增量，改动时同步本文档。

## 配置

```yaml
SSH:
  Timeout: 30s
  AllowedCommands:
    - uptime
    - df
    - free
    - hostname
    - uname
    - "systemctl status"
    - "docker ps"
    - "kubectl get"
    - tail
```

## 演进（待定，落地时更新本文档）

- Option B：完整 Web 交互终端（xterm.js + WebSocket）。
- 私钥加密落库（AES-GCM）。
- 命令白名单扩到写操作（重启服务等）时，考虑接入人工审批流程。
