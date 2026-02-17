package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// mockPipelineConfigRepo is a mock implementation of port.PipelineConfigRepository for testing.
type mockPipelineConfigRepo struct {
	configs map[uuid.UUID]*model.PipelineConfig
}

func newMockPipelineConfigRepo() *mockPipelineConfigRepo {
	return &mockPipelineConfigRepo{
		configs: make(map[uuid.UUID]*model.PipelineConfig),
	}
}

func (m *mockPipelineConfigRepo) GetByProjectID(_ context.Context, projectID uuid.UUID) (*model.PipelineConfig, error) {
	c, ok := m.configs[projectID]
	if !ok {
		return nil, errors.NewNotFound("pipeline_config", projectID)
	}
	return c, nil
}

func (m *mockPipelineConfigRepo) Upsert(_ context.Context, config *model.PipelineConfig) (*model.PipelineConfig, error) {
	existing, ok := m.configs[config.ProjectID]
	if ok {
		existing.ConfigYAML = config.ConfigYAML
		existing.Version++
		return existing, nil
	}
	config.ID = uuid.New()
	config.Version = 1
	m.configs[config.ProjectID] = config
	return config, nil
}

// validConfigYAML returns a minimal valid pipeline config in the new format.
func validConfigYAML() string {
	return `steps:
  - id: 880e8400-e29b-41d4-a716-446655440001
    name: implement
    action_type: implement
    model: claude-opus-4-6
    auto_approve: false
    retry_policy:
      max_retries: 2
      retry_type: on-failure
`
}

func TestPipelineConfigService_Upsert_ValidConfig(t *testing.T) {
	repo := newMockPipelineConfigRepo()
	svc := NewPipelineConfigService(repo)

	projectID := uuid.New()
	configYAML := validConfigYAML()

	result, err := svc.Upsert(context.Background(), projectID, configYAML)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ProjectID != projectID {
		t.Errorf("expected project_id %v, got %v", projectID, result.ProjectID)
	}
	if result.Version != 1 {
		t.Errorf("expected version 1, got %d", result.Version)
	}
	if result.ConfigYAML != configYAML {
		t.Errorf("config_yaml mismatch")
	}
}

func TestPipelineConfigService_Upsert_VersionIncrement(t *testing.T) {
	repo := newMockPipelineConfigRepo()
	svc := NewPipelineConfigService(repo)

	projectID := uuid.New()
	configYAML := validConfigYAML()

	// First upsert
	result, err := svc.Upsert(context.Background(), projectID, configYAML)
	if err != nil {
		t.Fatalf("unexpected error on first upsert: %v", err)
	}
	if result.Version != 1 {
		t.Errorf("expected version 1, got %d", result.Version)
	}

	// Second upsert should increment version
	result, err = svc.Upsert(context.Background(), projectID, configYAML)
	if err != nil {
		t.Fatalf("unexpected error on second upsert: %v", err)
	}
	if result.Version != 2 {
		t.Errorf("expected version 2, got %d", result.Version)
	}
}

func TestPipelineConfigService_Upsert_InvalidYAML(t *testing.T) {
	repo := newMockPipelineConfigRepo()
	svc := NewPipelineConfigService(repo)

	projectID := uuid.New()

	tests := []struct {
		name    string
		yaml    string
		errCode string
	}{
		{
			name:    "empty yaml",
			yaml:    "",
			errCode: "VALIDATION_ERROR",
		},
		{
			name:    "invalid yaml syntax",
			yaml:    "{{not valid yaml",
			errCode: "INVALID_PIPELINE_CONFIG",
		},
		{
			name:    "no steps",
			yaml:    "steps: []\n",
			errCode: "INVALID_PIPELINE_CONFIG",
		},
		{
			name: "step without name",
			yaml: `steps:
  - action_type: implement
`,
			errCode: "INVALID_PIPELINE_CONFIG",
		},
		{
			name: "step without action_type",
			yaml: `steps:
  - name: my_step
`,
			errCode: "INVALID_PIPELINE_CONFIG",
		},
		{
			name: "invalid action_type value",
			yaml: `steps:
  - name: my_step
    action_type: invalid_action
`,
			errCode: "INVALID_PIPELINE_CONFIG",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Upsert(context.Background(), projectID, tt.yaml)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			domainErr, ok := err.(*errors.DomainError)
			if !ok {
				t.Fatalf("expected DomainError, got %T", err)
			}
			if domainErr.Code != tt.errCode {
				t.Errorf("expected error code %q, got %q", tt.errCode, domainErr.Code)
			}
		})
	}
}

func TestPipelineConfigService_Upsert_AllValidActionTypes(t *testing.T) {
	repo := newMockPipelineConfigRepo()
	svc := NewPipelineConfigService(repo)

	projectID := uuid.New()
	configYAML := `steps:
  - name: step1
    action_type: implement
  - name: step2
    action_type: review
  - name: step3
    action_type: merge
  - name: step4
    action_type: test
  - name: step5
    action_type: custom
`

	result, err := svc.Upsert(context.Background(), projectID, configYAML)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestPipelineConfigService_GetByProjectID(t *testing.T) {
	repo := newMockPipelineConfigRepo()
	svc := NewPipelineConfigService(repo)

	projectID := uuid.New()

	// Get non-existent config
	_, err := svc.GetByProjectID(context.Background(), projectID)
	if err == nil {
		t.Fatal("expected error for non-existent config, got nil")
	}
	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected DomainError, got %T", err)
	}
	if domainErr.Category != errors.CategoryNotFound {
		t.Errorf("expected not_found category, got %q", domainErr.Category)
	}

	// Seed and then get
	_, err = svc.SeedDefault(context.Background(), projectID)
	if err != nil {
		t.Fatalf("unexpected error seeding: %v", err)
	}

	result, err := svc.GetByProjectID(context.Background(), projectID)
	if err != nil {
		t.Fatalf("unexpected error getting: %v", err)
	}
	if result.ProjectID != projectID {
		t.Errorf("expected project_id %v, got %v", projectID, result.ProjectID)
	}
}

func TestPipelineConfigService_SeedDefault(t *testing.T) {
	repo := newMockPipelineConfigRepo()
	svc := NewPipelineConfigService(repo)

	projectID := uuid.New()

	result, err := svc.SeedDefault(context.Background(), projectID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ProjectID != projectID {
		t.Errorf("expected project_id %v, got %v", projectID, result.ProjectID)
	}
	if result.ConfigYAML != DefaultPipelineConfigYAML {
		t.Error("expected default config yaml")
	}
	if result.Version != 1 {
		t.Errorf("expected version 1, got %d", result.Version)
	}
}
