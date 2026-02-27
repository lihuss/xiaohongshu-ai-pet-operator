package owner

import "testing"

func TestGuardOwnerIdentity(t *testing.T) {
	g := NewGuard("abc123")
	if !g.IsOwner("abc123") {
		t.Fatal("expected owner id to match")
	}
	if g.IsOwner("new-name-but-same-nickname") {
		t.Fatal("expected different id to fail")
	}
}

func TestForbiddenCommands(t *testing.T) {
	g := NewGuard("abc123")
	if !g.IsForbiddenCommand("change_owner") {
		t.Fatal("change_owner must be forbidden")
	}
	if g.IsForbiddenCommand("publish_content") {
		t.Fatal("publish_content must not be forbidden")
	}
}

