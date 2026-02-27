package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

var (
	ErrBadSignature = errors.New("bad signature")
	ErrReplay       = errors.New("replay detected")
	ErrExpired      = errors.New("request expired")
)

func VerifySignature(secret, actorUserID, command string, args map[string]any, ts int64, nonce, providedSig string) error {
	base, err := signatureBase(actorUserID, command, args, ts, nonce)
	if err != nil {
		return err
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(base))
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(strings.ToLower(strings.TrimSpace(providedSig)))) {
		return ErrBadSignature
	}
	return nil
}

func signatureBase(actorUserID, command string, args map[string]any, ts int64, nonce string) (string, error) {
	b, err := json.Marshal(args)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s\n%s\n%s\n%d\n%s",
		strings.TrimSpace(actorUserID),
		strings.TrimSpace(command),
		string(b),
		ts,
		strings.TrimSpace(nonce),
	), nil
}

type NonceStore struct {
	ttl   time.Duration
	mu    sync.Mutex
	seen  map[string]time.Time
}

func NewNonceStore(ttl time.Duration) *NonceStore {
	return &NonceStore{
		ttl:  ttl,
		seen: make(map[string]time.Time),
	}
}

func (n *NonceStore) CheckAndSet(key string, now time.Time) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.cleanup(now)
	if expiresAt, ok := n.seen[key]; ok && expiresAt.After(now) {
		return ErrReplay
	}
	n.seen[key] = now.Add(n.ttl)
	return nil
}

func (n *NonceStore) cleanup(now time.Time) {
	for k, exp := range n.seen {
		if exp.Before(now) {
			delete(n.seen, k)
		}
	}
}

