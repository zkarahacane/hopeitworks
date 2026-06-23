package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	apperrors "github.com/zakari/hopeitworks/backend/pkg/errors"
)

// --- hand-written mocks ---

type mockCapabilityRepo struct {
	forAgent map[uuid.UUID][]*model.Capability
}

func (m *mockCapabilityRepo) Create(context.Context, *model.Capability) (*model.Capability, error) {
	return nil, nil
}
func (m *mockCapabilityRepo) GetByID(context.Context, uuid.UUID) (*model.Capability, error) {
	return nil, apperrors.NewNotFound("capability", "x")
}
func (m *mockCapabilityRepo) ListByScope(context.Context, *uuid.UUID) ([]*model.Capability, error) {
	return nil, nil
}
func (m *mockCapabilityRepo) Delete(context.Context, uuid.UUID) error { return nil }
func (m *mockCapabilityRepo) AttachToAgent(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}
func (m *mockCapabilityRepo) DetachFromAgent(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}
func (m *mockCapabilityRepo) ListForAgent(_ context.Context, agentID uuid.UUID) ([]*model.Capability, error) {
	return m.forAgent[agentID], nil
}

type mockCredentialRepo struct {
	creds []*model.Credential
}

func (m *mockCredentialRepo) Create(_ context.Context, c *model.Credential) (*model.Credential, error) {
	m.creds = append(m.creds, c)
	return c, nil
}
func (m *mockCredentialRepo) GetByID(context.Context, uuid.UUID) (*model.Credential, error) {
	return nil, apperrors.NewNotFound("credential", "x")
}
func (m *mockCredentialRepo) GetGlobalByName(_ context.Context, name string) (*model.Credential, error) {
	for _, c := range m.creds {
		if c.Scope == model.CapabilityScopeGlobal && c.Name == name {
			return c, nil
		}
	}
	return nil, apperrors.NewNotFound("credential", name)
}
func (m *mockCredentialRepo) GetProjectByName(_ context.Context, name string, projectID uuid.UUID) (*model.Credential, error) {
	for _, c := range m.creds {
		if c.Scope == model.CapabilityScopeProject && c.Name == name && c.ProjectID != nil && *c.ProjectID == projectID {
			return c, nil
		}
	}
	return nil, apperrors.NewNotFound("credential", name)
}
func (m *mockCredentialRepo) ListByScope(context.Context, *uuid.UUID) ([]*model.Credential, error) {
	return m.creds, nil
}
func (m *mockCredentialRepo) Delete(context.Context, uuid.UUID) error { return nil }

// mockAgentRepo + testLogger are defined in the package's other _test.go files.

// --- helpers ---

func mustSpec(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}

func newBundleService(caps *mockCapabilityRepo, credRepo *mockCredentialRepo, agents *mockAgentRepo) *BundleService {
	credSvc := NewCredentialService(credRepo, "test-master-key")
	return NewBundleService(caps, credSvc, agents, testLogger())
}

// --- tests ---

// The core back-compat proof: a nil agent and a capability-less agent both yield an empty
// bundle, so the runtime materialises nothing and behaves exactly as before.
func TestComposeBundle_BackCompat_Empty(t *testing.T) {
	caps := &mockCapabilityRepo{forAgent: map[uuid.UUID][]*model.Capability{}}
	agentID := uuid.New()
	agents := &mockAgentRepo{agents: map[uuid.UUID]*model.Agent{
		agentID: {ID: agentID},
	}}
	svc := newBundleService(caps, &mockCredentialRepo{}, agents)

	t.Run("nil agent id", func(t *testing.T) {
		b, err := svc.ComposeBundle(context.Background(), uuid.Nil, "")
		require.NoError(t, err)
		assert.True(t, b.IsEmpty())
	})

	t.Run("agent with no capabilities", func(t *testing.T) {
		b, err := svc.ComposeBundle(context.Background(), agentID, "")
		require.NoError(t, err)
		assert.True(t, b.IsEmpty())
	})
}

func TestComposeBundle_Skill(t *testing.T) {
	agentID := uuid.New()
	caps := &mockCapabilityRepo{forAgent: map[uuid.UUID][]*model.Capability{
		agentID: {
			{
				ID:   uuid.New(),
				Kind: model.CapabilityKindSkill,
				Name: "code-review",
				Spec: mustSpec(t, model.SkillSpec{
					Files: map[string]string{"SKILL.md": "# Review skill"},
				}),
			},
		},
	}}
	agents := &mockAgentRepo{agents: map[uuid.UUID]*model.Agent{agentID: {ID: agentID}}}
	svc := newBundleService(caps, &mockCredentialRepo{}, agents)

	b, err := svc.ComposeBundle(context.Background(), agentID, "")
	require.NoError(t, err)
	require.Len(t, b.Skills, 1)
	assert.Equal(t, "code-review", b.Skills[0].Name) // falls back to capability name
	assert.Equal(t, "# Review skill", b.Skills[0].Files["SKILL.md"])
	assert.False(t, b.IsEmpty())
}

func TestComposeBundle_MCPServerWithCredential(t *testing.T) {
	agentID := uuid.New()
	credRepo := &mockCredentialRepo{}
	svc := newBundleService(&mockCapabilityRepo{}, credRepo, &mockAgentRepo{
		agents: map[uuid.UUID]*model.Agent{agentID: {ID: agentID}},
	})

	// Seed a global credential through the real CredentialService (encrypts at rest).
	_, err := svc.credSvc.CreateGlobal(context.Background(), "KANBAN_TOKEN", "s3cret")
	require.NoError(t, err)

	caps := &mockCapabilityRepo{forAgent: map[uuid.UUID][]*model.Capability{
		agentID: {
			{
				ID:   uuid.New(),
				Kind: model.CapabilityKindMCPServer,
				Name: "kanban",
				Spec: mustSpec(t, model.MCPServerSpec{
					Transport:     "http",
					URL:           "http://mcp-kanban.internal/sse",
					Headers:       map[string]string{"Authorization": "Bearer ${KANBAN_TOKEN}"},
					CredentialRef: "KANBAN_TOKEN",
				}),
			},
		},
	}}
	svc.capRepo = caps

	b, err := svc.ComposeBundle(context.Background(), agentID, "")
	require.NoError(t, err)
	require.Contains(t, b.MCP.MCPServers, "kanban")
	assert.Equal(t, "http://mcp-kanban.internal/sse", b.MCP.MCPServers["kanban"].URL)
	// The secret is resolved (decrypted) into the credentials map; the header keeps its ref.
	assert.Equal(t, "s3cret", b.Credentials["KANBAN_TOKEN"])
	assert.Equal(t, "Bearer ${KANBAN_TOKEN}", b.MCP.MCPServers["kanban"].Headers["Authorization"])
}

// A missing credential is warn+skipped: the server is still added, no secret, no error.
func TestComposeBundle_MissingCredential_WarnSkip(t *testing.T) {
	agentID := uuid.New()
	caps := &mockCapabilityRepo{forAgent: map[uuid.UUID][]*model.Capability{
		agentID: {
			{
				ID:   uuid.New(),
				Kind: model.CapabilityKindMCPServer,
				Name: "kanban",
				Spec: mustSpec(t, model.MCPServerSpec{
					URL:           "http://mcp/sse",
					CredentialRef: "ABSENT",
				}),
			},
		},
	}}
	svc := newBundleService(caps, &mockCredentialRepo{}, &mockAgentRepo{
		agents: map[uuid.UUID]*model.Agent{agentID: {ID: agentID}},
	})

	b, err := svc.ComposeBundle(context.Background(), agentID, "")
	require.NoError(t, err)
	assert.Contains(t, b.MCP.MCPServers, "kanban")
	assert.Empty(t, b.Credentials)
}

func TestComposeBundle_ToolPolicyAndMalformedSkipped(t *testing.T) {
	agentID := uuid.New()
	caps := &mockCapabilityRepo{forAgent: map[uuid.UUID][]*model.Capability{
		agentID: {
			{
				ID:   uuid.New(),
				Kind: model.CapabilityKindToolPolicy,
				Name: "reviewer-policy",
				Spec: mustSpec(t, model.ToolPolicySpec{
					Allow: []string{"Read", "Grep"},
					Deny:  []string{"Bash(rm:*)"},
				}),
			},
			{
				ID:   uuid.New(),
				Kind: model.CapabilityKindSkill,
				Name: "broken",
				Spec: []byte(`{not json`), // malformed -> warn+skip
			},
		},
	}}
	svc := newBundleService(caps, &mockCredentialRepo{}, &mockAgentRepo{
		agents: map[uuid.UUID]*model.Agent{agentID: {ID: agentID}},
	})

	b, err := svc.ComposeBundle(context.Background(), agentID, "")
	require.NoError(t, err)
	require.NotNil(t, b.ToolPolicy)
	assert.Equal(t, []string{"Read", "Grep"}, b.ToolPolicy.Allow)
	assert.Equal(t, []string{"Bash(rm:*)"}, b.ToolPolicy.Deny)
	assert.Empty(t, b.Skills) // the malformed skill was skipped
}

func TestComposeBundle_RoleFiltering(t *testing.T) {
	agentBase := &mockAgentRepo{agents: map[uuid.UUID]*model.Agent{}}

	tests := []struct {
		name    string
		caps    func(agentID uuid.UUID) []*model.Capability
		role    string
		checkFn func(t *testing.T, b model.RuntimeBundle)
	}{
		{
			name: "review role — mcp_server scopé dev filtré",
			caps: func(_ uuid.UUID) []*model.Capability {
				return []*model.Capability{
					{
						ID:   uuid.New(),
						Kind: model.CapabilityKindMCPServer,
						Name: "write-mcp",
						Spec: mustSpec(t, model.MCPServerSpec{
							Name:  "write-mcp",
							URL:   "http://write-mcp.internal/sse",
							Roles: []string{model.RoleDev},
						}),
					},
				}
			},
			role: model.RoleReview,
			checkFn: func(t *testing.T, b model.RuntimeBundle) {
				assert.NotContains(t, b.MCP.MCPServers, "write-mcp")
			},
		},
		{
			name: "review role — mcp_server universel inclus",
			caps: func(_ uuid.UUID) []*model.Capability {
				return []*model.Capability{
					{
						ID:   uuid.New(),
						Kind: model.CapabilityKindMCPServer,
						Name: "read-mcp",
						Spec: mustSpec(t, model.MCPServerSpec{
							Name:  "read-mcp",
							URL:   "http://read-mcp.internal/sse",
							Roles: []string{},
						}),
					},
				}
			},
			role: model.RoleReview,
			checkFn: func(t *testing.T, b model.RuntimeBundle) {
				assert.Contains(t, b.MCP.MCPServers, "read-mcp")
			},
		},
		{
			name: "dev role — mcp_server scopé dev inclus",
			caps: func(_ uuid.UUID) []*model.Capability {
				return []*model.Capability{
					{
						ID:   uuid.New(),
						Kind: model.CapabilityKindMCPServer,
						Name: "write-mcp",
						Spec: mustSpec(t, model.MCPServerSpec{
							Name:  "write-mcp",
							URL:   "http://write-mcp.internal/sse",
							Roles: []string{model.RoleDev},
						}),
					},
				}
			},
			role: model.RoleDev,
			checkFn: func(t *testing.T, b model.RuntimeBundle) {
				assert.Contains(t, b.MCP.MCPServers, "write-mcp")
			},
		},
		{
			name: "rôle vide — seulement capacités universelles",
			caps: func(_ uuid.UUID) []*model.Capability {
				return []*model.Capability{
					{
						ID:   uuid.New(),
						Kind: model.CapabilityKindMCPServer,
						Name: "write-mcp",
						Spec: mustSpec(t, model.MCPServerSpec{
							Name:  "write-mcp",
							URL:   "http://write-mcp.internal/sse",
							Roles: []string{model.RoleDev},
						}),
					},
					{
						ID:   uuid.New(),
						Kind: model.CapabilityKindMCPServer,
						Name: "read-mcp",
						Spec: mustSpec(t, model.MCPServerSpec{
							Name:  "read-mcp",
							URL:   "http://read-mcp.internal/sse",
							Roles: []string{},
						}),
					},
				}
			},
			role: "",
			checkFn: func(t *testing.T, b model.RuntimeBundle) {
				assert.NotContains(t, b.MCP.MCPServers, "write-mcp")
				assert.Contains(t, b.MCP.MCPServers, "read-mcp")
			},
		},
		{
			name: "tool_policy scopée review appliquée à review",
			caps: func(_ uuid.UUID) []*model.Capability {
				return []*model.Capability{
					{
						ID:   uuid.New(),
						Kind: model.CapabilityKindToolPolicy,
						Name: "review-policy",
						Spec: mustSpec(t, model.ToolPolicySpec{
							Allow: []string{"Read", "Grep"},
							Deny:  []string{"Bash", "Edit", "Write"},
							Roles: []string{model.RoleReview},
						}),
					},
				}
			},
			role: model.RoleReview,
			checkFn: func(t *testing.T, b model.RuntimeBundle) {
				require.NotNil(t, b.ToolPolicy)
				assert.Contains(t, b.ToolPolicy.Deny, "Bash")
			},
		},
		{
			name: "tool_policy scopée review non appliquée à dev",
			caps: func(_ uuid.UUID) []*model.Capability {
				return []*model.Capability{
					{
						ID:   uuid.New(),
						Kind: model.CapabilityKindToolPolicy,
						Name: "review-policy",
						Spec: mustSpec(t, model.ToolPolicySpec{
							Allow: []string{"Read", "Grep"},
							Deny:  []string{"Bash", "Edit", "Write"},
							Roles: []string{model.RoleReview},
						}),
					},
				}
			},
			role: model.RoleDev,
			checkFn: func(t *testing.T, b model.RuntimeBundle) {
				assert.Nil(t, b.ToolPolicy)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			agentID := uuid.New()
			agentBase.agents[agentID] = &model.Agent{ID: agentID}
			caps := &mockCapabilityRepo{forAgent: map[uuid.UUID][]*model.Capability{
				agentID: tc.caps(agentID),
			}}
			svc := newBundleService(caps, &mockCredentialRepo{}, agentBase)
			b, err := svc.ComposeBundle(context.Background(), agentID, tc.role)
			require.NoError(t, err)
			tc.checkFn(t, b)
		})
	}
}
