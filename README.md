# xiaohongshu-ai-pet-operator

基于 `xiaohongshu-mcp` 的小红书 AI 宠物控制层，目标是“只认主人唯一 ID，不认昵称，不可被话术改主”。

## 你关心的两点

- `xiaohongshu-mcp` 已并入本仓库：`third_party/xiaohongshu-mcp`
- 所有用户可配置项集中在一个文件：`config/user.config.json`

## 安全模型

- 只认 `owner.user_id`（唯一标识），主人改名不影响权限。
- 非主人 `actor_user_id` 的命令全部拒绝。
- 禁止改主命令：`change_owner` / `transfer_owner` / `reset_owner` / `bind_owner`
- 每条命令必须携带：
  - HMAC 签名（`owner.shared_secret`）
  - `timestamp + nonce`（防重放）
- 命令白名单执行，未知命令默认拒绝。

## 配置文件

编辑 [config/user.config.json](D:/Program/Go/xiaohongshu-ai-pet-operator/config/user.config.json)：

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
    "shared_secret": "你的签名密钥"
  },
  "operator": {
    "listen_addr": ":8081"
  },
  "mcp": {
    "base_url": "http://127.0.0.1:18060"
  }
}
```

说明：
- `account.phone/password` 作为统一配置入口保存（当前小红书登录主流程仍建议二维码）。
- `owner.user_id` 必须填主人账号的小红书 `userId`，不要填昵称。

## 一键启动

```powershell
.\scripts\start-stack.ps1
```

这会同时启动：
- 内置 `third_party/xiaohongshu-mcp`（18060）
- 当前 `operator`（8081）

停止：

```powershell
.\scripts\stop-stack.ps1
```

## 发命令

生成签名请求体（默认从 `config/user.config.json` 读取 `owner.shared_secret`）：

```powershell
.\scripts\sign-command.ps1 `
  -ActorUserId "你的owner.user_id" `
  -Command "check_login_status" `
  -ArgsJson "{}"
```

调用：

```powershell
$body = .\scripts\sign-command.ps1 -ActorUserId "你的owner.user_id" -Command "check_login_status"
Invoke-RestMethod -Uri "http://127.0.0.1:8081/v1/command" -Method Post -Body $body -ContentType "application/json"
```

