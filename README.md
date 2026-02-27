# 小红书 AI 宠物 (xiaohongshu-ai-pet)

本项目把小红书账号接入 MCP，让 Gemini / Claude 成为“宠物大脑”，通过对话自主刷小红书。

## 1. 双账号模型（重要）

- 主人账号（Owner）：只负责发命令与身份识别，必须填写 `owner.user_id`。
- 宠物账号（Pet）：实际登录并执行搜索、发帖、评论等动作。
- 结论：`owner.user_id` 不能为空，且必须是**主人账号**，不是宠物账号。

## 2. Skill 思路（大脑自主，不是机械脚本）

- 自主性来自 Gemini/Claude：它们决定看什么、怎么互动、何时收尾。
- “刷 5 分钟”是**软时长预算**，不是定时器强杀。
- 到时间或收到停止指令时，执行**缓刹**：
  - 不再开启新互动
  - 完成当前评论/回复动作
  - 输出简短总结后停下
- 避免急刹导致“回复写到一半被终止”。

## 3. 配置文件

编辑 [config/user.config.json](config/user.config.json)：

```json
{
  "owner": {
    "user_id": "主人账号 user_id（必填，不是宠物账号）"
  },
  "mcp": {
    "base_url": "http://127.0.0.1:18060"
  }
}
```

字段说明：
- `owner.user_id`：宠物识别主人消息和命令的唯一标识。
- `mcp.base_url`：底层服务地址，默认值即可。

## 4. 编译 MCP 插件

```powershell
go build -o bin/xhs-pet.exe ./cmd/mcp/main.go
```

## 5. 接入 Gemini / Claude

将编译产物注册为 StdIO MCP Server。

Claude Desktop 示例（`%APPDATA%\Claude\claude_desktop_config.json`）：

```json
{
  "mcpServers": {
    "xhs-pet": {
      "command": "C:\\你的项目路径\\bin\\xhs-pet.exe",
      "args": []
    }
  }
}
```

## 6. 对话内登录（无需单独跑命令）

- 插件启动后会自动拉起底层引擎，并打开可见浏览器（`headless=false`）。
- 在 Gemini / Claude 对话里先调用 `ensure_pet_login`：
  - 若已登录：直接返回可用状态
  - 若未登录：触发登录流程，用户在弹窗浏览器扫码登录宠物账号

## 7. 建议的 Skill 启动流程

1) 调用 `pet_skill_profile` 获取人格与行为准则
2) 调用 `ensure_pet_login` 确认宠物账号登录
3) 调用 `pet_autonomy_begin`（可带 `duration_minutes`）
4) 期间循环调用 `pet_autonomy_status`，临近时长主动收尾
5) 用户中途想停时，调用 `pet_autonomy_stop`，按缓刹策略结束

## 8. 运行特性

- 无需手动先启动脚本：Gemini/Claude 调用 MCP 时自动拉起底层引擎。
- 会话结束自动释放：MCP 进程退出时会清理子进程。
- 动态端口分配：避免固定端口冲突。

