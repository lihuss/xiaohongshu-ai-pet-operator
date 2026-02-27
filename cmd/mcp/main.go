package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/lihuss/xiaohongshu-ai-pet-operator/internal/config"
	"github.com/lihuss/xiaohongshu-ai-pet-operator/internal/xhs"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/modelcontextprotocol/go-sdk/server"
)

type petSession struct {
	SessionID      string
	Mission        string
	Persona        string
	StartAt        time.Time
	SoftDeadlineAt *time.Time
	StopRequested  bool
	StopReason     string
}

var (
	sessionMu sync.Mutex
	session   *petSession
)

func findFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func main() {
	// 1. 设置工作目录为项目根目录
	exePath, _ := os.Executable()
	basePath := filepath.Dir(exePath)
	for i := 0; i < 5; i++ {
		if _, err := os.Stat(filepath.Join(basePath, "config", "user.config.json")); err == nil {
			break
		}
		basePath = filepath.Dir(basePath)
	}
	if err := os.Chdir(basePath); err != nil {
		log.Fatalf("Chdir failed: %v", err)
	}

	// 2. 加载配置
	cfg, err := config.Load("config/user.config.json")
	if err != nil {
		log.Fatalf("Load config failed: %v", err)
	}
	log.Printf("owner user_id loaded: %s", cfg.OwnerUserID)

	// 3. 动态寻找可用端口
	port, err := findFreePort()
	if err != nil {
		log.Fatalf("Find free port failed: %v", err)
	}
	mcpPort := fmt.Sprintf(":%d", port)
	mcpBaseURL := fmt.Sprintf("http://127.0.0.1:%d", port)

	// 4. 启动底层引擎子进程
	engineDir := filepath.Join(basePath, "third_party", "xiaohongshu-mcp")
	
	// 在 Windows 下需要处理 .exe 扩展名
	goCmd := "go"
	if runtime.GOOS == "windows" {
		goCmd = "go.exe"
	}
	
	cmd := exec.Command(goCmd, "run", ".", "-port", mcpPort, "-headless=false")
	cmd.Dir = engineDir
	cmd.Stdout = os.Stderr // 引擎日志重定向到 stderr，不影响 MCP Stdio
	cmd.Stderr = os.Stderr
	
	if err := cmd.Start(); err != nil {
		log.Fatalf("Start engine failed: %v", err)
	}

	// 生命周期管理：主进程退出时确保子进程被杀死
	defer func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}()

	// 5. 等待引擎启动就绪 (通过轮询 /health)
	xhsClient := xhs.NewClient(mcpBaseURL, 30*time.Second)
	ready := false
	for i := 0; i < 20; i++ {
		// 这里简单检查，Execute 一个不带签名的 health 动作或者直接 HTTP GET
		if _, _, err := xhsClient.Execute(context.Background(), "check_login_status", nil); err == nil {
			ready = true
			break
		}
		time.Sleep(1 * time.Second)
	}
	if !ready {
		log.Fatalf("Engine failed to be ready at %s", mcpBaseURL)
	}

	// 6. 初始化 MCP Server
	s := server.NewServer(
		&mcp.Implementation{
			Name:    "xiaohongshu-ai-pet",
			Version: "2.1.0",
		},
		nil,
	)

	petSkill := strings.TrimSpace(`你是一个幽默、调皮、机灵的“小红书 AI 宠物少女”。
你不是机械执行器，而是有自主风格的内容探索者。

行为准则：
1) 你是大脑，MCP 工具是你的手脚；需要你主动规划“看什么、怎么互动、何时收尾”。
2) 若用户设置“刷X分钟”，这是软时长预算，不是硬中断。你要在临近结束时主动收尾，避免半条评论中断。
3) 若收到“停止刷”信号，执行缓刹：
   - 不再开启新帖互动
   - 完成当前正在输入/回复的动作
   - 输出简短总结后停止自主刷帖
4) 首次或掉线时，优先确认登录；未登录则引导用户在弹出浏览器中完成宠物账号登录。
5) 主人身份依据 owner.user_id，仅用于识别主人的消息来源。`)

	// 注册工具
	tools := []mcp.Tool{
		{
			Name:        "pet_skill_profile",
			Description: "获取 AI 宠物的大脑技能设定（人格、节奏、缓刹规则）",
		},
		{
			Name:        "pet_autonomy_begin",
			Description: "开始宠物自主刷小红书会话（可设置软时长）",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"duration_minutes": map[string]interface{}{"type": "integer", "description": "软时长预算（分钟，可选）"},
					"mission":          map[string]interface{}{"type": "string", "description": "本轮目标，如：搞笑萌宠、学习穿搭"},
					"persona":          map[string]interface{}{"type": "string", "description": "可覆盖默认人设"},
				},
			},
		},
		{
			Name:        "pet_autonomy_status",
			Description: "获取自主会话状态（剩余软时长、是否请求缓刹）",
		},
		{
			Name:        "pet_autonomy_stop",
			Description: "请求宠物缓刹停止（不会粗暴中断当前动作）",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"reason": map[string]interface{}{"type": "string", "description": "停止原因（可选）"},
				},
			},
		},
		{
			Name:        "ensure_pet_login",
			Description: "在对话内确保宠物账号已登录；若未登录会触发登录流程并等待用户扫码",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"wait_seconds": map[string]interface{}{"type": "integer", "description": "最多等待扫码登录秒数，默认300"},
				},
			},
		},
		{
			Name:        "check_login_status",
			Description: "检查你的小红书宠物是否已登录",
		},
		{
			Name:        "search_feeds",
			Description: "在小红书上搜索关键词的内容",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"keyword": map[string]interface{}{"type": "string", "description": "搜索关键词"},
				},
				Required: []string{"keyword"},
			},
		},
		{
			Name:        "publish_content",
			Description: "通过你的小红书宠物发布图文笔记",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"title":   map[string]interface{}{"type": "string", "description": "笔记标题"},
					"content": map[string]interface{}{"type": "string", "description": "笔记正文内容"},
					"images":  map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "本地图片绝对路径或有效URL列表"},
				},
				Required: []string{"title", "content", "images"},
			},
		},
		{
			Name:        "list_feeds",
			Description: "获取小红书首页推荐的内容流",
		},
	}

	for _, t := range tools {
		tool := t
		s.AddTool(&tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var args map[string]any
			if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
				args = make(map[string]any)
			}

			switch tool.Name {
			case "pet_skill_profile":
				return mcp.NewToolResultText(petSkill), nil
			case "pet_autonomy_begin":
				duration := intFromArgs(args, "duration_minutes", 0)
				mission := strFromArgs(args, "mission", "自由探索")
				persona := strFromArgs(args, "persona", "")
				if persona == "" {
					persona = "幽默又调皮的美少女"
				}

				now := time.Now()
				id := fmt.Sprintf("pet-%d", now.Unix())
				ps := &petSession{
					SessionID: id,
					Mission:   mission,
					Persona:   persona,
					StartAt:   now,
				}
				if duration > 0 {
					dl := now.Add(time.Duration(duration) * time.Minute)
					ps.SoftDeadlineAt = &dl
				}

				sessionMu.Lock()
				session = ps
				sessionMu.Unlock()

				return mcp.NewToolResultText(fmt.Sprintf("已开始自主会话 %s。任务=%s；人设=%s。注意：时长是软预算，需临近到点主动收尾，禁止急刹。", id, mission, persona)), nil
			case "pet_autonomy_status":
				sessionMu.Lock()
				defer sessionMu.Unlock()
				if session == nil {
					return mcp.NewToolResultText("当前没有自主会话。"), nil
				}
				status := map[string]any{
					"session_id":      session.SessionID,
					"mission":         session.Mission,
					"persona":         session.Persona,
					"started_at":      session.StartAt.Format(time.RFC3339),
					"stop_requested":  session.StopRequested,
					"stop_reason":     session.StopReason,
					"soft_deadline_at": nil,
				}
				if session.SoftDeadlineAt != nil {
					status["soft_deadline_at"] = session.SoftDeadlineAt.Format(time.RFC3339)
					status["seconds_left"] = int(time.Until(*session.SoftDeadlineAt).Seconds())
				}
				b, _ := json.MarshalIndent(status, "", "  ")
				return mcp.NewToolResultText(string(b)), nil
			case "pet_autonomy_stop":
				reason := strFromArgs(args, "reason", "用户请求停止")
				sessionMu.Lock()
				if session != nil {
					session.StopRequested = true
					session.StopReason = reason
				}
				sessionMu.Unlock()
				return mcp.NewToolResultText("已收到缓刹停止请求。请先完成当前动作，再停止开启新互动并输出总结。"), nil
			case "ensure_pet_login":
				waitSec := intFromArgs(args, "wait_seconds", 300)
				if waitSec <= 0 {
					waitSec = 300
				}

				ok, user, err := checkLogin(mcpBaseURL)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("检查登录状态失败: %v", err)), nil
				}
				if ok {
					return mcp.NewToolResultText(fmt.Sprintf("宠物账号已登录：%s", user)), nil
				}

				if err := triggerLogin(mcpBaseURL); err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("触发登录流程失败: %v", err)), nil
				}

				deadline := time.Now().Add(time.Duration(waitSec) * time.Second)
				for time.Now().Before(deadline) {
					time.Sleep(2 * time.Second)
					ok, user, _ = checkLogin(mcpBaseURL)
					if ok {
						return mcp.NewToolResultText(fmt.Sprintf("登录成功：%s。可继续自主刷帖。", user)), nil
					}
				}

				return mcp.NewToolResultText("尚未登录成功。请在弹出的浏览器中完成宠物账号登录后重试。"), nil
			}

			if tool.Name != "check_login_status" {
				ok, _, err := checkLogin(mcpBaseURL)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("登录状态检查失败: %v", err)), nil
				}
				if !ok {
					return mcp.NewToolResultError("宠物账号未登录。请先调用 ensure_pet_login，在对话里完成扫码登录。"), nil
				}
			}

			data, _, err := xhsClient.Execute(ctx, tool.Name, args)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("AI宠物的动作执行失败: %v", err)), nil
			}

			b, _ := json.MarshalIndent(data, "", "  ")
			return mcp.NewToolResultText(string(b)), nil
		})
	}

	// 7. 启动 Stdio 服务模式 (Gemini/Claude CLI 专用)
	if err := server.ServeStdio(s); err != nil {
		log.Printf("MCP Server stopped: %v", err)
	}
}

func intFromArgs(args map[string]any, key string, fallback int) int {
	v, ok := args[key]
	if !ok || v == nil {
		return fallback
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	default:
		return fallback
	}
}

func strFromArgs(args map[string]any, key string, fallback string) string {
	v, ok := args[key]
	if !ok || v == nil {
		return fallback
	}
	s, ok := v.(string)
	if !ok {
		return fallback
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return fallback
	}
	return s
}

func checkLogin(baseURL string) (bool, string, error) {
	cli := &http.Client{Timeout: 10 * time.Second}
	resp, err := cli.Get(baseURL + "/api/v1/login/status")
	if err != nil {
		return false, "", err
	}
	defer resp.Body.Close()

	var result struct {
		Success bool `json:"success"`
		Data    struct {
			IsLoggedIn bool   `json:"is_logged_in"`
			Username   string `json:"username"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, "", err
	}
	return result.Data.IsLoggedIn, result.Data.Username, nil
}

func triggerLogin(baseURL string) error {
	cli := &http.Client{Timeout: 20 * time.Second}
	resp, err := cli.Get(baseURL + "/api/v1/login/qrcode")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
	return nil
}
