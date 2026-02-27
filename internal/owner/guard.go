package owner

import "strings"

type Guard struct {
	ownerUserID string
}

var forbiddenOwnerCommands = map[string]struct{}{
	"change_owner":   {},
	"transfer_owner": {},
	"reset_owner":    {},
	"bind_owner":     {},
}

func NewGuard(ownerUserID string) *Guard {
	return &Guard{ownerUserID: strings.TrimSpace(ownerUserID)}
}

func (g *Guard) OwnerUserID() string {
	return g.ownerUserID
}

func (g *Guard) IsOwner(actorUserID string) bool {
	return strings.TrimSpace(actorUserID) == g.ownerUserID
}

func (g *Guard) IsForbiddenCommand(command string) bool {
	_, ok := forbiddenOwnerCommands[strings.ToLower(strings.TrimSpace(command))]
	return ok
}

