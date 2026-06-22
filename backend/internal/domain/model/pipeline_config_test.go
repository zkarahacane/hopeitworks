package model

import (
	"testing"
)

func TestParsePipelineConfigYAML_GroupsFormat(t *testing.T) {
	yaml := []byte(`groups:
  - id: setup
    name: Setup
    steps:
      - name: create-branch
        action_type: git_branch
  - id: dev
    name: Development
    steps:
      - name: dev-agent
        action_type: agent_run
      - name: review-agent
        action_type: agent_run
`)

	cfg, err := ParsePipelineConfigYAML(yaml)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(cfg.Groups))
	}

	if cfg.Groups[0].Name != "Setup" {
		t.Errorf("expected group[0].Name = %q, got %q", "Setup", cfg.Groups[0].Name)
	}
	if cfg.Groups[0].ID != "setup" {
		t.Errorf("expected group[0].ID = %q, got %q", "setup", cfg.Groups[0].ID)
	}
	if len(cfg.Groups[0].Steps) != 1 {
		t.Fatalf("expected 1 step in group[0], got %d", len(cfg.Groups[0].Steps))
	}
	if cfg.Groups[0].Steps[0].ActionType != "git_branch" {
		t.Errorf("expected group[0].steps[0].action_type = %q, got %q", "git_branch", cfg.Groups[0].Steps[0].ActionType)
	}

	if cfg.Groups[1].Name != "Development" {
		t.Errorf("expected group[1].Name = %q, got %q", "Development", cfg.Groups[1].Name)
	}
	if len(cfg.Groups[1].Steps) != 2 {
		t.Fatalf("expected 2 steps in group[1], got %d", len(cfg.Groups[1].Steps))
	}
}

func TestParsePipelineConfigYAML_LegacyFlatSteps(t *testing.T) {
	yaml := []byte(`steps:
  - name: implement
    action_type: implement
  - name: review
    action_type: review
  - name: merge
    action_type: merge
`)

	cfg, err := ParsePipelineConfigYAML(yaml)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Legacy steps should be wrapped in a single "Default" group
	if len(cfg.Groups) != 1 {
		t.Fatalf("expected 1 group (auto-wrapped), got %d", len(cfg.Groups))
	}

	group := cfg.Groups[0]
	if group.ID != "default" {
		t.Errorf("expected auto-wrapped group ID %q, got %q", "default", group.ID)
	}
	if group.Name != "Default" {
		t.Errorf("expected auto-wrapped group Name %q, got %q", "Default", group.Name)
	}
	if len(group.Steps) != 3 {
		t.Fatalf("expected 3 steps in auto-wrapped group, got %d", len(group.Steps))
	}
	if group.Steps[0].Name != "implement" {
		t.Errorf("expected step[0].Name = %q, got %q", "implement", group.Steps[0].Name)
	}
	if group.Steps[1].Name != "review" {
		t.Errorf("expected step[1].Name = %q, got %q", "review", group.Steps[1].Name)
	}
	if group.Steps[2].Name != "merge" {
		t.Errorf("expected step[2].Name = %q, got %q", "merge", group.Steps[2].Name)
	}
}

func TestParsePipelineConfigYAML_EmptyInput(t *testing.T) {
	cfg, err := ParsePipelineConfigYAML([]byte(""))
	if err != nil {
		t.Fatalf("unexpected error for empty input: %v", err)
	}
	if len(cfg.Groups) != 0 {
		t.Errorf("expected 0 groups for empty input, got %d", len(cfg.Groups))
	}
}

func TestParsePipelineConfigYAML_InvalidYAML(t *testing.T) {
	_, err := ParsePipelineConfigYAML([]byte("{{not valid yaml"))
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

func TestParsePipelineConfigYAML_GroupsPreferredOverSteps(t *testing.T) {
	// If both groups and steps are present, groups should take precedence
	yaml := []byte(`groups:
  - id: g1
    name: G1
    steps:
      - name: s1
        action_type: agent_run
steps:
  - name: legacy
    action_type: implement
`)

	cfg, err := ParsePipelineConfigYAML(yaml)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Groups) != 1 {
		t.Fatalf("expected 1 group (groups take precedence), got %d", len(cfg.Groups))
	}
	if cfg.Groups[0].Name != "G1" {
		t.Errorf("expected group from groups field, got %q", cfg.Groups[0].Name)
	}
}

func TestParsePipelineConfigYAML_StepConfig(t *testing.T) {
	yaml := []byte(`groups:
  - id: setup
    name: Setup
    steps:
      - name: create-branch
        action_type: git_branch
        config:
          base_branch: main
          target_branch: develop
`)

	cfg, err := ParsePipelineConfigYAML(yaml)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	step := cfg.Groups[0].Steps[0]
	if step.Config == nil {
		t.Fatal("expected non-nil Config map")
	}
	if step.Config["base_branch"] != "main" {
		t.Errorf("expected config[base_branch] = %q, got %q", "main", step.Config["base_branch"])
	}
	if step.Config["target_branch"] != "develop" {
		t.Errorf("expected config[target_branch] = %q, got %q", "develop", step.Config["target_branch"])
	}
}

func TestParsePipelineConfigYAML_StepDescription(t *testing.T) {
	yaml := []byte(`groups:
  - id: setup
    name: Setup
    steps:
      - name: create-branch
        action_type: git_branch
        description: Create feature branch from base
`)

	cfg, err := ParsePipelineConfigYAML(yaml)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	step := cfg.Groups[0].Steps[0]
	if step.Description != "Create feature branch from base" {
		t.Errorf("expected description %q, got %q", "Create feature branch from base", step.Description)
	}
}

func TestFlatSteps_MultipleGroups(t *testing.T) {
	cfg := &PipelineConfigYAML{
		Groups: []PipelineGroup{
			{
				ID:   "g1",
				Name: "G1",
				Steps: []PipelineStep{
					{Name: "s1", ActionType: "git_branch"},
					{Name: "s2", ActionType: "agent_run"},
				},
			},
			{
				ID:   "g2",
				Name: "G2",
				Steps: []PipelineStep{
					{Name: "s3", ActionType: "git_pr"},
				},
			},
			{
				ID:   "g3",
				Name: "G3",
				Steps: []PipelineStep{
					{Name: "s4", ActionType: "ci_poll"},
					{Name: "s5", ActionType: "notification"},
				},
			},
		},
	}

	flat := cfg.FlatSteps()
	if len(flat) != 5 {
		t.Fatalf("expected 5 flat steps, got %d", len(flat))
	}

	expectedNames := []string{"s1", "s2", "s3", "s4", "s5"}
	for i, expected := range expectedNames {
		if flat[i].Name != expected {
			t.Errorf("flat step[%d]: expected name %q, got %q", i, expected, flat[i].Name)
		}
	}
}

func TestFlatSteps_EmptyGroups(t *testing.T) {
	cfg := &PipelineConfigYAML{
		Groups: []PipelineGroup{},
	}

	flat := cfg.FlatSteps()
	if len(flat) != 0 {
		t.Errorf("expected 0 flat steps for empty groups, got %d", len(flat))
	}
}

func TestFlatStepsWithStage_PreservesGroupIdentity(t *testing.T) {
	cfg := &PipelineConfigYAML{
		Groups: []PipelineGroup{
			{ID: "dev", Name: "Dev", Steps: []PipelineStep{
				{Name: "branch", ActionType: "git_branch"},
				{Name: "code", ActionType: "agent_run"},
			}},
			{ID: "review", Name: "Review", Steps: []PipelineStep{
				{Name: "pr", ActionType: "git_pr"},
			}},
		},
	}

	withStage := cfg.FlatStepsWithStage()
	if len(withStage) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(withStage))
	}
	// Order matches FlatSteps; each step carries its originating group identity.
	wantStage := []struct{ name, gid, gname string }{
		{"branch", "dev", "Dev"},
		{"code", "dev", "Dev"},
		{"pr", "review", "Review"},
	}
	for i, w := range wantStage {
		if withStage[i].Step.Name != w.name {
			t.Errorf("step[%d]: expected name %q, got %q", i, w.name, withStage[i].Step.Name)
		}
		if withStage[i].GroupID != w.gid {
			t.Errorf("step[%d]: expected group id %q, got %q", i, w.gid, withStage[i].GroupID)
		}
		if withStage[i].GroupName != w.gname {
			t.Errorf("step[%d]: expected group name %q, got %q", i, w.gname, withStage[i].GroupName)
		}
	}
}

func TestParsePipelineConfigYAML_TransitionDefaultsToAuto(t *testing.T) {
	yaml := []byte(`
groups:
  - id: dev
    name: Dev
    steps:
      - id: s1
        name: code
        action_type: agent_run
  - id: review
    name: Review
    transition: gate
    steps:
      - id: s2
        name: pr
        action_type: git_pr
`)
	cfg, err := ParsePipelineConfigYAML(yaml)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Groups[0].Transition != TransitionAuto {
		t.Errorf("group with no transition: expected %q, got %q", TransitionAuto, cfg.Groups[0].Transition)
	}
	if cfg.Groups[1].Transition != TransitionGate {
		t.Errorf("group with explicit gate: expected %q, got %q", TransitionGate, cfg.Groups[1].Transition)
	}
}

func TestValidActionTypes_NewTypes(t *testing.T) {
	newTypes := []string{"agent_run", "git_branch", "git_pr", "notification", "human", "ci_poll", "hitl_gate"}
	for _, at := range newTypes {
		if !ValidActionTypes[at] {
			t.Errorf("expected action type %q to be valid", at)
		}
	}
}

func TestValidActionTypes_LegacyTypes(t *testing.T) {
	legacyTypes := []string{"implement", "review", "merge", "test", "custom"}
	for _, at := range legacyTypes {
		if !ValidActionTypes[at] {
			t.Errorf("expected legacy action type %q to be valid", at)
		}
	}
}

func TestValidActionTypes_InvalidType(t *testing.T) {
	if ValidActionTypes["unknown_action"] {
		t.Error("expected unknown_action to be invalid")
	}
}
