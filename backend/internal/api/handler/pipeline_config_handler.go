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

	configYAML, err := stepsToYAML(req.Steps)
	if err != nil {
		writeErrorResponse(w, errors.NewValidation("steps", "invalid pipeline steps"))
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
	ID          string          `yaml:"id"`
	Name        string          `yaml:"name"`
	ActionType  string          `yaml:"action_type"`
	Model       string          `yaml:"model"`
	AutoApprove bool            `yaml:"auto_approve"`
	RetryPolicy retryPolicyYAML `yaml:"retry_policy"`
}

// retryPolicyYAML is the intermediate YAML representation for a retry policy.
type retryPolicyYAML struct {
	MaxRetries int    `yaml:"max_retries"`
	RetryType  string `yaml:"retry_type"`
}

// pipelineConfigYAML is the intermediate YAML representation for the full config.
type pipelineConfigYAML struct {
	Steps []pipelineStepYAML `yaml:"steps"`
}

// stepsToYAML serialises API pipeline steps to a YAML string for domain storage.
func stepsToYAML(steps []PipelineStep) (string, error) {
	cfg := pipelineConfigYAML{
		Steps: make([]pipelineStepYAML, len(steps)),
	}
	for i, s := range steps {
		cfg.Steps[i] = pipelineStepYAML{
			ID:          s.Id.String(),
			Name:        s.Name,
			ActionType:  string(s.ActionType),
			Model:       string(s.Model),
			AutoApprove: s.AutoApprove,
			RetryPolicy: retryPolicyYAML{
				MaxRetries: s.RetryPolicy.MaxRetries,
				RetryType:  string(s.RetryPolicy.RetryType),
			},
		}
	}
	out, err := yaml.Marshal(cfg)
	if err != nil {
		return "", err
	}
	return string(out), nil
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

	steps := make([]PipelineStep, len(cfg.Steps))
	for i, s := range cfg.Steps {
		stepID, _ := uuid.Parse(s.ID)
		steps[i] = PipelineStep{
			Id:          stepID,
			Name:        s.Name,
			ActionType:  PipelineStepActionType(s.ActionType),
			Model:       PipelineStepModel(s.Model),
			AutoApprove: s.AutoApprove,
			RetryPolicy: RetryPolicy{
				MaxRetries: s.RetryPolicy.MaxRetries,
				RetryType:  RetryPolicyRetryType(s.RetryPolicy.RetryType),
			},
		}
	}

	return PipelineConfig{
		ProjectId: c.ProjectID,
		Steps:     steps,
		UpdatedAt: c.UpdatedAt,
	}, nil
}
