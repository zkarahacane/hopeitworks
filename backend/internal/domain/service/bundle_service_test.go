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
		b, err := svc.ComposeBundle(context.Background(), uuid.Nil)
		require.NoError(t, err)
		assert.True(t, b.IsEmpty())
	})

	t.Run("agent with no capabilities", func(t *testing.T) {
		b, err := svc.ComposeBundle(context.Background(), agentID)
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

	b, err := svc.ComposeBundle(context.Background(), agentID)
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

	b, err := svc.ComposeBundle(context.Background(), agentID)
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

	b, err := svc.ComposeBundle(context.Background(), agentID)
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

	b, err := svc.ComposeBundle(context.Background(), agentID)
	require.NoError(t, err)
	require.NotNil(t, b.ToolPolicy)
	assert.Equal(t, []string{"Read", "Grep"}, b.ToolPolicy.Allow)
	assert.Equal(t, []string{"Bash(rm:*)"}, b.ToolPolicy.Deny)
	assert.Empty(t, b.Skills) // the malformed skill was skipped
}
