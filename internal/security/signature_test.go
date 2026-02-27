package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"
)

func TestVerifySignature(t *testing.T) {
	secret := "secret"
	args := map[string]any{"feed_id": "x1", "like": true}
	ts := int64(1700000000)
	nonce := "n-1"

	base, err := signatureBase("u123", "like_feed", args, ts, nonce)
	if err != nil {
		t.Fatal(err)
	}
	m := hmac.New(sha256.New, []byte(secret))
	_, _ = m.Write([]byte(base))
	sig := hex.EncodeToString(m.Sum(nil))

	if err := VerifySignature(secret, "u123", "like_feed", args, ts, nonce, sig); err != nil {
		t.Fatalf("expected signature to pass: %v", err)
	}
}

func TestNonceReplay(t *testing.T) {
	n := NewNonceStore(2 * time.Minute)
	now := time.Now()
	if err := n.CheckAndSet("u123:n-1", now); err != nil {
		t.Fatal(err)
	}
	if err := n.CheckAndSet("u123:n-1", now.Add(5*time.Second)); err == nil {
		t.Fatal("expected replay error")
	}
}

