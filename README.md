# xiaohongshu-ai-pet-operator

通过 MCP 协议将小红书账号接入 Gemini / Claude，让 AI 模型以自主宠物的角色操作小红书，完成浏览、互动与发布。

## 功能概述

- **自主浏览**：AI 主动获取推荐流或按主题搜索内容。
- **自主互动**：AI 对笔记发表评论、回复评论，风格由人格设定驱动。
- **内容发布**：AI 可代表宠物账号发布笔记。
- **时长控制**：支持软时长预算，临近到点时 AI 自动完成当前动作后收尾，避免强制中断。
- **缓停机制**：收到停止指令后优先完成进行中的动作，再输出总结。
- **双账号隔离**：主人账号仅用于身份识别与指令来源验证，宠物账号执行全部操作。

## 架构说明

`text
[Gemini / Claude]
      │  对话 + MCP 工具调用
      ▼
[xhs-pet MCP 插件]  ← 本项目编译产物
      │  HTTP
      ▼
[xiaohongshu-mcp 底层服务]  ← third_party/xiaohongshu-mcp
      │  Chromium DevTools Protocol
      ▼
[可见浏览器 (headless=false)]
      │
      ▼
[小红书 宠物账号]
`

- MCP 插件以 StdIO 方式被 Claude Desktop / Gemini 客户端调用。
- 底层服务在插件首次被调用时自动启动，进程退出时自动清理。
- 端口动态分配，避免冲突。

## 前置条件

- Go 1.21+
- Google Chrome 或 Chromium（底层服务驱动浏览器）
- 支持 MCP 的 AI 客户端，如 [Claude Desktop](https://claude.ai/download)
- 两个小红书账号：一个作为主人账号，一个作为宠物账号

## 快速开始

### 1. 克隆项目

`bash
git clone https://github.com/your-org/xiaohongshu-ai-pet-operator.git
cd xiaohongshu-ai-pet-operator
`

### 2. 配置主人账号

编辑 `config/user.config.json`：

`json
{
  "owner": {
    "user_id": "<主人账号的小红书 user_id>"
  },
  "mcp": {
    "base_url": "http://127.0.0.1:18060"
  }
}
`

- `owner.user_id`：填写**主人账号**的 user_id，用于宠物识别指令来源，不能填宠物账号。
- `mcp.base_url`：底层服务监听地址，保持默认即可。

> 获取 user_id：登录小红书网页版，进入个人主页，URL 中 `/user/profile/` 后的字符串即为 user_id。

### 3. 编译 MCP 插件


# Windows
```bash
go build -o bin/xhs-pet.exe ./cmd/mcp/main.go
```
# macOS / Linux
```bash
go build -o bin/xhs-pet ./cmd/mcp/main.go
```

### 4. 注册到 AI 客户端

**Claude Desktop**（`%APPDATA%\Claude\claude_desktop_config.json`）：

`json
{
  "mcpServers": {
    "xhs-pet": {
      "command": "C:\\path\\to\\bin\\xhs-pet.exe",
      "args": []
    }
  }
}
`

修改后重启 Claude Desktop。

### 5. 加载 Skill 提示词

打开 `~/.gemini/skills` ，创建一个新文件夹并命名为 `xiaohongshu_ai_pet` ，将 SKILL.md 放进其中。

或将 `SKILL.md` 的内容作为系统提示词（System Prompt）或对话开头提示词输入给 AI 模型，使其具备宠物人格与行为规则。

### 6. 登录宠物账号

在 AI 对话中发送任意启动指令，AI 会自动调用 `ensure_pet_login`。若宠物账号尚未登录，底层服务会打开浏览器窗口，扫码登录宠物账号即可。登录状态持久化保存，后续会话无需重复扫码。

### 7. 开始使用

直接在对话中向 AI 下达指令，例如：

`
给你刷五分钟小红书，走可爱萌宠路线。
`

`
帮我搜一下最近流行的猫咪视频，看到有趣的就评论一下。
`

`
先别刷了，停一下。
`

AI 会根据 `SKILL.md` 中的行为规则自主执行，并周期性汇报进展。

## 项目结构

`
cmd/mcp/          MCP 插件入口
cmd/server/       独立 HTTP 服务入口（可选）
config/           用户配置文件
internal/         核心逻辑（配置、模型、安全、XHS 客户端）
third_party/xiaohongshu-mcp/  底层浏览器自动化服务
SKILL.md          AI 宠物技能提示词
`

## 注意事项

- 本项目仅供学习与个人研究使用，请勿用于商业目的或违反小红书服务条款的行为。
- 宠物账号的所有操作均以真实账号身份执行，请对发布内容负责。
- 建议使用专用宠物账号，不要使用主要个人账号。