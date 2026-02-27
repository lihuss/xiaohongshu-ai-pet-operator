package xhs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	httpCli *http.Client
}

type route struct {
	Method   string
	Path     string
	QueryArg bool
}

var allowlist = map[string]route{
	"check_login_status": {Method: http.MethodGet, Path: "/api/v1/login/status", QueryArg: true},
	"my_profile":         {Method: http.MethodGet, Path: "/api/v1/user/me", QueryArg: true},
	"list_feeds":         {Method: http.MethodGet, Path: "/api/v1/feeds/list", QueryArg: true},
	"search_feeds":       {Method: http.MethodPost, Path: "/api/v1/feeds/search"},
	"feed_detail":        {Method: http.MethodPost, Path: "/api/v1/feeds/detail"},
	"user_profile":       {Method: http.MethodPost, Path: "/api/v1/user/profile"},
	"publish_content":    {Method: http.MethodPost, Path: "/api/v1/publish"},
	"publish_video":      {Method: http.MethodPost, Path: "/api/v1/publish_video"},
	"post_comment":       {Method: http.MethodPost, Path: "/api/v1/feeds/comment"},
	"reply_comment":      {Method: http.MethodPost, Path: "/api/v1/feeds/comment/reply"},
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpCli: &http.Client{Timeout: timeout},
	}
}

func (c *Client) Execute(ctx context.Context, command string, args map[string]any) (map[string]any, int, error) {
	rt, ok := allowlist[strings.ToLower(strings.TrimSpace(command))]
	if !ok {
		return nil, http.StatusBadRequest, fmt.Errorf("command not allowed: %s", command)
	}

	endpoint := c.baseURL + rt.Path
	var req *http.Request
	var err error

	if rt.Method == http.MethodGet && rt.QueryArg {
		u, parseErr := url.Parse(endpoint)
		if parseErr != nil {
			return nil, http.StatusInternalServerError, parseErr
		}
		q := u.Query()
		for k, v := range args {
			q.Set(k, fmt.Sprintf("%v", v))
		}
		u.RawQuery = q.Encode()
		req, err = http.NewRequestWithContext(ctx, rt.Method, u.String(), nil)
	} else {
		body, marshalErr := json.Marshal(args)
		if marshalErr != nil {
			return nil, http.StatusBadRequest, marshalErr
		}
		req, err = http.NewRequestWithContext(ctx, rt.Method, endpoint, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	resp, err := c.httpCli.Do(req)
	if err != nil {
		return nil, http.StatusBadGateway, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, http.StatusBadGateway, err
	}
	if len(raw) == 0 {
		return map[string]any{"status": "ok"}, resp.StatusCode, nil
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]any{"raw": string(raw)}, resp.StatusCode, nil
	}
	return out, resp.StatusCode, nil
}

