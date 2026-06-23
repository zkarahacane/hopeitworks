package service

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

// BundleService assembles an agent's composed capabilities into a runtime-agnostic
// RuntimeBundle (skills + MCP servers + tool policy + resolved secrets), which the
// agent-runtime fetches at startup over an authenticated channel.
//
// It is deliberately fail-soft: a malformed capability spec, an unknown kind, or an
// unresolvable credential is warned-and-skipped — never fatal. This realises the
// platform-wide warn+skip invariant at the assembly boundary, and guarantees the
// back-compat property: an agent with no capabilities yields an empty bundle.
type BundleService struct {
	capRepo   port.CapabilityRepository
	credSvc   *CredentialService
	agentRepo port.AgentRepository
	logger    *slog.Logger
}

// NewBundleService creates a new BundleService.
func NewBundleService(
	capRepo port.CapabilityRepository,
	credSvc *CredentialService,
	agentRepo port.AgentRepository,
	logger *slog.Logger,
) *BundleService {
	return &BundleService{
		capRepo:   capRepo,
		credSvc:   credSvc,
		agentRepo: agentRepo,
		logger:    logger,
	}
}

// ComposeBundle builds the RuntimeBundle for an agent, filtered by role.
// A nil agent id (uuid.Nil) or an agent with no capabilities yields the zero-value
// bundle — the runtime then materialises nothing and behaves exactly as before.
// The returned error is reserved for hard failures the caller may choose to degrade
// on; capability-level problems are warn+skip and never surface here.
//
// Fail-safe: a role of "" (empty/unknown) only receives universal capabilities
// (Roles == nil/empty). Role-scoped capabilities (Roles non-empty) are never granted
// to an unidentified role, ensuring the narrowest possible permission surface.
func (s *BundleService) ComposeBundle(ctx context.Context, agentID uuid.UUID, role string) (model.RuntimeBundle, error) {
	var bundle model.RuntimeBundle
	if agentID == uuid.Nil {
		return bundle, nil
	}

	// The agent's project scopes credential resolution (project secrets shadow globals).
	var projectID *uuid.UUID
	if agent, err := s.agentRepo.GetAgent(ctx, agentID); err != nil {
		s.logger.Warn("bundle: agent lookup failed, composing without project scope",
			"agent_id", agentID, "error", err)
	} else {
		projectID = agent.ProjectID
	}

	caps, err := s.capRepo.ListForAgent(ctx, agentID)
	if err != nil {
		return bundle, err
	}

	for _, c := range caps {
		switch c.Kind {
		case model.CapabilityKindSkill:
			s.addSkill(c, &bundle)
		case model.CapabilityKindMCPServer:
			s.addMCPServer(ctx, c, projectID, role, &bundle)
		case model.CapabilityKindToolPolicy:
			s.addToolPolicy(c, role, &bundle)
		default:
			s.logger.Warn("bundle: unknown capability kind, skipping",
				"capability_id", c.ID, "kind", c.Kind)
		}
	}

	return bundle, nil
}

func (s *BundleService) addSkill(c *model.Capability, bundle *model.RuntimeBundle) {
	var skill model.SkillSpec
	if err := json.Unmarshal(c.Spec, &skill); err != nil {
		s.logger.Warn("bundle: malformed skill spec, skipping",
			"capability_id", c.ID, "name", c.Name, "error", err)
		return
	}
	if skill.Name == "" {
		skill.Name = c.Name
	}
	if len(skill.Files) == 0 {
		s.logger.Warn("bundle: skill has no files, skipping",
			"capability_id", c.ID, "name", c.Name)
		return
	}
	bundle.Skills = append(bundle.Skills, skill)
}

func (s *BundleService) addMCPServer(ctx context.Context, c *model.Capability, projectID *uuid.UUID, role string, bundle *model.RuntimeBundle) {
	var spec model.MCPServerSpec
	if err := json.Unmarshal(c.Spec, &spec); err != nil {
		s.logger.Warn("bundle: malformed mcp_server spec, skipping",
			"capability_id", c.ID, "name", c.Name, "error", err)
		return
	}

	// Fail-safe: an empty/unknown role does not match role-scoped capabilities.
	if !model.RoleMatches(spec.Roles, role) {
		s.logger.Debug("bundle: mcp_server filtered by role",
			"capability_id", c.ID, "name", c.Name, "spec_roles", spec.Roles, "role", role)
		return
	}

	name := spec.Name
	if name == "" {
		name = c.Name
	}

	entry := model.MCPServerEntry{
		URL:     spec.URL,
		Headers: spec.Headers,
		Command: spec.Command,
	}

	// Resolve the referenced secret (if any) into the bundle's credentials map. The
	// header keeps its ${ref} placeholder; the runtime expands it from credentials.
	if spec.CredentialRef != "" {
		value, err := s.credSvc.Resolve(ctx, spec.CredentialRef, projectID)
		if err != nil {
			s.logger.Warn("bundle: credential resolution failed, mcp server kept without secret",
				"capability_id", c.ID, "credential_ref", spec.CredentialRef, "error", err)
		} else {
			if bundle.Credentials == nil {
				bundle.Credentials = make(map[string]string)
			}
			bundle.Credentials[spec.CredentialRef] = value
		}
	}

	if bundle.MCP.MCPServers == nil {
		bundle.MCP.MCPServers = make(map[string]model.MCPServerEntry)
	}
	bundle.MCP.MCPServers[name] = entry
}

func (s *BundleService) addToolPolicy(c *model.Capability, role string, bundle *model.RuntimeBundle) {
	var policy model.ToolPolicySpec
	if err := json.Unmarshal(c.Spec, &policy); err != nil {
		s.logger.Warn("bundle: malformed tool_policy spec, skipping",
			"capability_id", c.ID, "name", c.Name, "error", err)
		return
	}

	// Fail-safe: an empty/unknown role does not match role-scoped policies.
	if !model.RoleMatches(policy.Roles, role) {
		s.logger.Debug("bundle: tool_policy filtered by role",
			"capability_id", c.ID, "name", c.Name, "spec_roles", policy.Roles, "role", role)
		return
	}

	if bundle.ToolPolicy == nil {
		bundle.ToolPolicy = &model.ToolPolicySpec{}
	}
	bundle.ToolPolicy.Allow = append(bundle.ToolPolicy.Allow, policy.Allow...)
	bundle.ToolPolicy.Deny = append(bundle.ToolPolicy.Deny, policy.Deny...)
}
