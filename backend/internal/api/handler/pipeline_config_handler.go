package handler

import (
	"net/http"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/service"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// PipelineConfigHandler implements pipeline config HTTP handlers.
type PipelineConfigHandler struct {
	service *service.PipelineConfigService
}

// NewPipelineConfigHandler creates a new PipelineConfigHandler.
func NewPipelineConfigHandler(svc *service.PipelineConfigService) *PipelineConfigHandler {
	return &PipelineConfigHandler{service: svc}
}

// GetPipelineConfig handles GET /projects/{projectId}/pipeline.
func (h *PipelineConfigHandler) GetPipelineConfig(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath) {
	config, err := h.service.GetByProjectID(r.Context(), projectID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	resp, err := toAPIPipelineConfig(config)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// UpdatePipelineConfig handles PUT /projects/{projectId}/pipeline.
// Only admin users can update pipeline configs.
func (h *PipelineConfigHandler) UpdatePipelineConfig(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath) {
	if !requireAdmin(w, r) {
		return
	}

	var req UpdatePipelineConfigRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}

	configYAML, err := groupsToYAML(req.Groups)
	if err != nil {
		writeErrorResponse(w, errors.NewValidation("groups", "invalid pipeline groups"))
		return
	}

	config, err := h.service.Upsert(r.Context(), projectID, configYAML)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	resp, err := toAPIPipelineConfig(config)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// pipelineStepYAML is the intermediate YAML representation for a pipeline step.
type pipelineStepYAML struct {
	ID          string            `yaml:"id"`
	Name        string            `yaml:"name"`
	ActionType  string            `yaml:"action_type"`
	AgentID     string            `yaml:"agent_id,omitempty"`
	Model       string            `yaml:"model,omitempty"`
	AutoApprove bool              `yaml:"auto_approve"`
	RetryPolicy retryPolicyYAML   `yaml:"retry_policy"`
	Config      map[string]string `yaml:"config,omitempty"`
}

// retryPolicyYAML is the intermediate YAML representation for a retry policy.
type retryPolicyYAML struct {
	MaxRetries int    `yaml:"max_retries"`
	RetryType  string `yaml:"retry_type"`
}

// pipelineGroupYAML is the intermediate YAML representation for a pipeline group.
type pipelineGroupYAML struct {
	ID    string             `yaml:"id"`
	Name  string             `yaml:"name"`
	Steps []pipelineStepYAML `yaml:"steps"`
}

// pipelineConfigYAML is the intermediate YAML representation for the full config.
type pipelineConfigYAML struct {
	Groups []pipelineGroupYAML `yaml:"groups"`
}

// groupsToYAML serialises API pipeline groups to a YAML string for domain storage.
func groupsToYAML(groups []PipelineGroup) (string, error) {
	cfg := pipelineConfigYAML{
		Groups: make([]pipelineGroupYAML, len(groups)),
	}
	for i, g := range groups {
		steps := make([]pipelineStepYAML, len(g.Steps))
		for j, s := range g.Steps {
			steps[j] = stepToYAML(s)
		}
		cfg.Groups[i] = pipelineGroupYAML{
			ID:    g.Id,
			Name:  g.Name,
			Steps: steps,
		}
	}
	out, err := yaml.Marshal(cfg)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// stepToYAML converts a single API PipelineStep to its YAML representation.
func stepToYAML(s PipelineStep) pipelineStepYAML {
	y := pipelineStepYAML{
		ID:          s.Id.String(),
		Name:        s.Name,
		ActionType:  string(s.ActionType),
		AutoApprove: s.AutoApprove,
		RetryPolicy: retryPolicyYAML{
			MaxRetries: s.RetryPolicy.MaxRetries,
			RetryType:  string(s.RetryPolicy.RetryType),
		},
	}
	if s.AgentId != nil {
		y.AgentID = s.AgentId.String()
	}
	if s.Model != nil {
		y.Model = *s.Model
	}
	if s.Config != nil {
		y.Config = *s.Config
	}
	return y
}

// toAPIPipelineConfig converts a domain PipelineConfig to the API PipelineConfig type.
func toAPIPipelineConfig(c *model.PipelineConfig) (PipelineConfig, error) {
	var cfg pipelineConfigYAML
	if err := yaml.Unmarshal([]byte(c.ConfigYAML), &cfg); err != nil {
		return PipelineConfig{}, &errors.DomainError{
			Category: errors.CategoryInternal,
			Code:     "INVALID_PIPELINE_CONFIG",
			Message:  "failed to parse stored pipeline config",
		}
	}

	groups := make([]PipelineGroup, len(cfg.Groups))
	for i, g := range cfg.Groups {
		steps := make([]PipelineStep, len(g.Steps))
		for j, s := range g.Steps {
			stepID, _ := uuid.Parse(s.ID)
			step := PipelineStep{
				Id:          stepID,
				Name:        s.Name,
				ActionType:  PipelineStepActionType(s.ActionType),
				AutoApprove: s.AutoApprove,
				RetryPolicy: RetryPolicy{
					MaxRetries: s.RetryPolicy.MaxRetries,
					RetryType:  RetryPolicyRetryType(s.RetryPolicy.RetryType),
				},
			}
			if s.AgentID != "" {
				agentUUID, err := uuid.Parse(s.AgentID)
				if err == nil {
					step.AgentId = &agentUUID
				}
			}
			if s.Model != "" {
				model := s.Model
				step.Model = &model
			}
			if len(s.Config) > 0 {
				step.Config = &s.Config
			}
			steps[j] = step
		}
		groups[i] = PipelineGroup{
			Id:    g.ID,
			Name:  g.Name,
			Steps: steps,
		}
	}

	return PipelineConfig{
		ProjectId: c.ProjectID,
		Groups:    groups,
		UpdatedAt: c.UpdatedAt,
	}, nil
}
