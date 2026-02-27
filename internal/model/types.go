package model

import "encoding/json"

type CommandRequest struct {
	ActorUserID   string                 `json:"actor_user_id"`
	ActorNickname string                 `json:"actor_nickname,omitempty"`
	Command       string                 `json:"command"`
	Args          map[string]any         `json:"args,omitempty"`
	Timestamp     int64                  `json:"timestamp"`
	Nonce         string                 `json:"nonce"`
	Signature     string                 `json:"signature"`
	RawPayload    map[string]json.RawMessage `json:"-"`
}

type CommandResponse struct {
	OK      bool   `json:"ok"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

