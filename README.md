# 小红书 AI 宠物安全控制层 (xiaohongshu-ai-pet-operator)

一个面向小红书 AI 宠物的安全控制层。

### 核心目标
- **身份唯一识别**：只认主人的 `userId`（不认昵称，改名不影响）。
- **指令执行权限**：确保只有主人下达的指令能被执行。
- **安全加固**：通过 HMAC 签名验证（Shared Secret）防御拦截篡改，防止重放攻击。
- **管理隔离**：禁止通过 AI 指令修改主人绑定信息，防止“宠物被拐”。

本仓库已内置可运行的 `xiaohongshu-mcp`，`clone` 后无需再单独下载。

---

## 1. 快速开始流程

### 第一步：前置要求
- **操作系统**：Windows (已配置 PowerShell)
- **环境驱动**：已安装 **Go 1.22+** (推荐 1.26+)
- **小红书账号**：一个用于作为“宠物”运行的小红书账号。

### 第二步：配置 `user.config.json`
编辑 [config/user.config.json](config/user.config.json)：

```json
{
  "owner": {
    "user_id": "你的小红书数字ID",
    "shared_secret": "自定义的长随机字符串"
  },
  "operator": {
    "listen_addr": ":8081"
  },
  "mcp": {
    "base_url": "http://127.0.0.1:18060"
  }
}
```

#### 字段详细说明：

| 字段 | 作用 | 如何获取 |
| :--- | :--- | :--- |
| `owner.user_id` | **主人的唯一身份标识**。只有该 ID 发出的指令会被执行。 | 打开手机小红书 -> 我 -> 这里的 ID 或者是主页分享链接中的数字部分。 |
| `owner.shared_secret` | **通信密钥**。用于对发给 Operator 的指令进行加密签名。 | **自行设定**。建议在 [随机字符串生成器](https://1password.com/zh-cn/password-generator/) 生成一个 32 位以上的字符串。 |
| `operator.listen_addr` | 安全层监听地址。 | 默认 `:8081` 即可。 |
| `mcp.base_url` | 内部 MCP 服务地址。 | 默认 `http://127.0.0.1:18060` 即可。 |

---

## 2. 登录流程 (重要)

由于小红书的 API 限制，必须先通过模拟浏览器进行登录获取 Cookie。

### 1. 启动登录工具
在项目根目录打开 PowerShell，运行：
```powershell
cd third_party\xiaohongshu-mcp
go run cmd/login/main.go
```

### 2. 扫码登录
- 程序会弹出一个 **Chrome 浏览器窗口**（如果是第一次运行，会先自动下载浏览器内核，请耐心等待）。
- 在弹出的窗口中，使用**作为宠物的那个小红书账号**扫码登录。
- 登录成功后，浏览器窗口会自动关闭，登录信息会保存到 `third_party/xiaohongshu-mcp/data/` 目录下。

### 3. 验证登录状态
登录完成后，回到根目录启动整个服务堆栈：
```powershell
.\scripts\start-stack.ps1
```
然后运行以下程序检查状态（需要主人身份）：
```powershell
$body = .\scripts\sign-command.ps1 -ActorUserId "你的主人user_id" -Command "check_login_status"
Invoke-RestMethod -Uri "http://127.0.0.1:8081/v1/command" -Method Post -Body $body -ContentType "application/json"
```

---

## 3. 运行与停止

### 启动服务
一键启动 MCP 服务 + 安全控制层：
```powershell
.\scripts\start-stack.ps1
```
*日志将保存在 `logs/` 目录下（`mcp.out.log` 和 `operator.out.log`）。*

### 停止服务
```powershell
.\scripts\stop-stack.ps1
```

---

## 4. 如何发送安全指令

为了防止指令被伪造，所有的 POST 请求必须带有 HMAC-SHA256 签名。

### 推荐方式：使用内置脚本
我们提供了 [scripts/sign-command.ps1](scripts/sign-command.ps1) 辅助生成带签名的 JSON：

```powershell
# 1. 生成带签名的 JSON 体
$payload = .\scripts\sign-command.ps1 `
  -ActorUserId "主人ID" `
  -Command "search" `
  -ArgsJson '{"keyword": "萨摩耶", "sort": "general"}'

# 2. 发送请求给 Operator
Invoke-RestMethod -Uri "http://127.0.0.1:8081/v1/command" -Method Post -Body $payload -ContentType "application/json"
```

### 签名算法 (供二次开发参考)
签名 Base 串格式：
`actor_user_id + "\n" + command + "\n" + args_json_string + "\n" + timestamp + "\n" + nonce`
使用 `shared_secret` 对该字符串进行 HMAC-SHA256 计算。

---

## 5. 目录结构说明

- `third_party/xiaohongshu-mcp`: 上游 MCP 服务，负责具体的网页自动化操作。
- `internal/security`: 核心签名验证逻辑。
- `internal/owner`: 主人权限控制逻辑（拦截白名单以外的敏感操作）。
- `scripts/`: 运维与调试工具集。

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

