package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/lihuss/xiaohongshu-ai-pet-operator/internal/config"
	"github.com/lihuss/xiaohongshu-ai-pet-operator/internal/model"
	"github.com/lihuss/xiaohongshu-ai-pet-operator/internal/owner"
	"github.com/lihuss/xiaohongshu-ai-pet-operator/internal/security"
	"github.com/lihuss/xiaohongshu-ai-pet-operator/internal/xhs"
)

type Handler struct {
	cfg   *config.Config
	guard *owner.Guard
	nonce *security.NonceStore
	xhs   *xhs.Client
}

func NewHandler(cfg *config.Config, guard *owner.Guard, nonce *security.NonceStore, x *xhs.Client) *Handler {
	return &Handler{cfg: cfg, guard: guard, nonce: nonce, xhs: x}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", h.health)
	mux.HandleFunc("/v1/command", h.command)
	return mux
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, model.CommandResponse{
		OK:      true,
		Message: "ok",
		Data: map[string]string{
			"owner_user_id": h.guard.OwnerUserID(),
		},
	})
}

func (h *Handler) command(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, model.CommandResponse{OK: false, Code: "method_not_allowed"})
		return
	}

	var req model.CommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, model.CommandResponse{OK: false, Code: "bad_request", Message: err.Error()})
		return
	}
	if req.Args == nil {
		req.Args = map[string]any{}
	}

	if h.guard.IsForbiddenCommand(req.Command) {
		writeJSON(w, http.StatusForbidden, model.CommandResponse{
			OK: false, Code: "forbidden_command", Message: "owner change commands are disabled",
		})
		return
	}

	if !h.guard.IsOwner(req.ActorUserID) {
		writeJSON(w, http.StatusForbidden, model.CommandResponse{
			OK: false,
			Code: "not_owner",
			Message: "only bound owner_user_id can control this AI pet",
		})
		return
	}

	now := time.Now().Unix()
	if req.Timestamp < now-300 || req.Timestamp > now+300 {
		writeJSON(w, http.StatusUnauthorized, model.CommandResponse{
			OK: false, Code: "expired", Message: "timestamp out of allowed window",
		})
		return
	}

	if err := security.VerifySignature(
		h.cfg.OwnerSharedSecret,
		req.ActorUserID,
		req.Command,
		req.Args,
		req.Timestamp,
		req.Nonce,
		req.Signature,
	); err != nil {
		status := http.StatusUnauthorized
		if errors.Is(err, security.ErrBadSignature) {
			status = http.StatusForbidden
		}
		writeJSON(w, status, model.CommandResponse{OK: false, Code: "signature_invalid", Message: err.Error()})
		return
	}

	nonceKey := strings.TrimSpace(req.ActorUserID) + ":" + strings.TrimSpace(req.Nonce)
	if err := h.nonce.CheckAndSet(nonceKey, time.Now()); err != nil {
		writeJSON(w, http.StatusUnauthorized, model.CommandResponse{OK: false, Code: "replay", Message: err.Error()})
		return
	}

	data, status, err := h.xhs.Execute(r.Context(), req.Command, req.Args)
	if err != nil {
		writeJSON(w, status, model.CommandResponse{OK: false, Code: "upstream_error", Message: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, model.CommandResponse{OK: true, Data: data})
}

func writeJSON(w http.ResponseWriter, status int, payload model.CommandResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

