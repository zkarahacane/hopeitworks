package action_test

import (
	"reflect"
	"testing"

	"github.com/zakari/hopeitworks/backend/internal/adapter/action"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

// TestA4_AgentRunNeverDependsOnGitCredentialResolver enforces the A4 security
// invariant: a resolved git PAT is consumed ONLY by server-side git adapters and is
// NEVER injected into an agent container. The structural guarantee is that the
// agent_run action does not (transitively, as a direct field) hold a
// port.GitCredentialResolver — so it cannot route the decrypted token into the
// container env/prompt/transcript. The git token the container still receives comes
// exclusively from os.Getenv (the legacy operator-level env), not from the resolver.
func TestA4_AgentRunNeverDependsOnGitCredentialResolver(t *testing.T) {
	resolverIface := reflect.TypeOf((*port.GitCredentialResolver)(nil)).Elem()

	typ := reflect.TypeOf(action.AgentRunAction{})
	for i := 0; i < typ.NumField(); i++ {
		ft := typ.Field(i).Type
		if ft.Implements(resolverIface) {
			t.Fatalf("A4 violation: AgentRunAction field %q (%s) implements GitCredentialResolver; "+
				"the resolved PAT must never reach the agent container", typ.Field(i).Name, ft.String())
		}
	}
}
