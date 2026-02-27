# xiaohongshu-ai-pet-operator

一个放在 `xiaohongshu-mcp` 前面的安全控制层，目标是让 AI 宠物只听“主人账号”的命令。

## 核心安全设计

- 主人身份只看 `owner_user_id`（唯一 ID），不看昵称。
- 主人改名不影响控制权，只要 `actor_user_id` 不变就仍然是主人。
- 默认禁止任何“改主人”指令：`change_owner / transfer_owner / reset_owner / bind_owner`。
- 每个命令都要携带 HMAC 签名（`OWNER_SHARED_SECRET`），避免被伪造。
- 每个命令都要携带 `timestamp + nonce`，防重放攻击。
- 命令白名单执行，未知命令全部拒绝。

## 目录说明

- `cmd/server`: 启动入口
- `internal/server`: HTTP API
- `internal/security`: 签名与防重放
- `internal/owner`: 主人识别与高危命令封禁
- `internal/xhs`: 调用 `xiaohongshu-mcp` 的白名单路由

## 环境变量

复制 `.env.example` 自行填值：

- `OWNER_USER_ID`: 主人唯一 ID（小红书 userId）
- `OWNER_SHARED_SECRET`: 签名密钥（建议 32+ 随机字符）
- `MCP_BASE_URL`: `xiaohongshu-mcp` 地址，默认 `http://127.0.0.1:18060`
- `LISTEN_ADDR`: 本服务监听地址，默认 `:8081`

## 启动

```bash
go run ./cmd/server
```

## 发命令（PowerShell）

先生成签名请求体：

```powershell
.\scripts\sign-command.ps1 `
  -ActorUserId "你的owner_user_id" `
  -Command "check_login_status" `
  -Secret "你的OWNER_SHARED_SECRET" `
  -ArgsJson "{}"
```

再调用服务：

```powershell
$body = .\scripts\sign-command.ps1 -ActorUserId "你的owner_user_id" -Command "check_login_status" -Secret "你的OWNER_SHARED_SECRET"
Invoke-RestMethod -Uri "http://127.0.0.1:8081/v1/command" -Method Post -Body $body -ContentType "application/json"
```

## 已开放命令

- `check_login_status`
- `my_profile`
- `list_feeds`
- `search_feeds`
- `feed_detail`
- `user_profile`
- `publish_content`
- `publish_video`
- `post_comment`
- `reply_comment`

要新增命令，请修改 `internal/xhs/client.go` 的 `allowlist`。

