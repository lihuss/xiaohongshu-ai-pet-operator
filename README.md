# xiaohongshu-ai-pet-operator

一个面向小红书 AI 宠物的安全控制层。

目标：
- 只认主人唯一标识 `owner.user_id`（不认昵称，改名不影响）
- 只执行主人命令
- 拒绝“改主人”类指令与重放攻击

本仓库已内置可运行的 `xiaohongshu-mcp`，`clone` 后无需再单独下载。

## 1. 目录结构

- `third_party/xiaohongshu-mcp`: 内置上游 MCP（已含 Windows 兼容补丁）
- `config/user.config.json`: 用户唯一配置文件
- `scripts/start-stack.ps1`: 一键启动 MCP + Operator
- `scripts/stop-stack.ps1`: 一键停止
- `scripts/sign-command.ps1`: 生成签名命令请求

## 2. 前置要求

- Windows + PowerShell
- Go（推荐 1.22+，已验证 1.26.0）

## 3. 配置

编辑 `config/user.config.json`：

```json
{
  "account": {
    "phone": "",
    "password": "",
    "country_code": "+86",
    "login_method": "qrcode"
  },
  "owner": {
    "user_id": "你的主人user_id",
    "shared_secret": "你的长随机密钥"
  },
  "operator": {
    "listen_addr": ":8081"
  },
  "mcp": {
    "base_url": "http://127.0.0.1:18060"
  }
}
```

字段说明：
- `owner.user_id`: 主人唯一 ID（必须是小红书 userId，不是昵称）
- `owner.shared_secret`: 命令签名密钥（建议 32 位以上）
- `account.*`: 登录信息集中入口（当前实际登录建议二维码流程）

## 4. 启动

在仓库根目录执行：

```powershell
.\scripts\start-stack.ps1
```

健康检查：

```powershell
Invoke-RestMethod http://127.0.0.1:18060/health
Invoke-RestMethod http://127.0.0.1:8081/healthz
```

停止：

```powershell
.\scripts\stop-stack.ps1
```

## 5. 首次登录小红书账号

先看登录状态（通过 Operator 转发到 MCP）：

```powershell
$body = .\scripts\sign-command.ps1 -ActorUserId "你的owner.user_id" -Command "check_login_status"
Invoke-RestMethod -Uri "http://127.0.0.1:8081/v1/command" -Method Post -Body $body -ContentType "application/json"
```

如果返回 `is_logged_in=false`，按 `third_party/xiaohongshu-mcp` 的登录流程扫码登录（其原始 README 有完整步骤）。

## 6. 发送命令

示例：查询登录状态

```powershell
$body = .\scripts\sign-command.ps1 `
  -ActorUserId "你的owner.user_id" `
  -Command "check_login_status" `
  -ArgsJson "{}"

Invoke-RestMethod -Uri "http://127.0.0.1:8081/v1/command" -Method Post -Body $body -ContentType "application/json"
```

示例：搜索内容

```powershell
$args = '{"keyword":"宠物","filters":{}}'
$body = .\scripts\sign-command.ps1 -ActorUserId "你的owner.user_id" -Command "search_feeds" -ArgsJson $args
Invoke-RestMethod -Uri "http://127.0.0.1:8081/v1/command" -Method Post -Body $body -ContentType "application/json"
```

## 7. 安全机制

- 仅当 `actor_user_id == owner.user_id` 才放行
- 签名校验：HMAC-SHA256
- 防重放：`timestamp + nonce`（超时与重复请求拒绝）
- 禁止指令：`change_owner` / `transfer_owner` / `reset_owner` / `bind_owner`
- 命令白名单执行，未知命令直接拒绝

## 8. 常见问题

- `leakless.exe` 被拦截  
  本仓库内置 MCP 已关闭 leakless 启动方式，默认可用。

- 端口占用  
  先运行 `.\scripts\stop-stack.ps1` 再重启。

- 主人改名后是否失效  
  不会。系统只认 `owner.user_id`。

