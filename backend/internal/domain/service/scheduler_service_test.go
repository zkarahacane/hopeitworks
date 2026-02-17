package service

import (
	"testing"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

func newStory(key string, dependsOn []string, targetFiles []string) model.Story {
	return model.Story{
		ID:          uuid.New(),
		ProjectID:   uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		Key:         key,
		Title:       "Story " + key,
		DependsOn:   dependsOn,
		TargetFiles: targetFiles,
		Status:      model.StoryStatusBacklog,
	}
}

func extractKeys(groups [][]model.Story) [][]string {
	result := make([][]string, len(groups))
	for i, group := range groups {
		keys := make([]string, len(group))
		for j, s := range group {
			keys[j] = s.Key
		}
		result[i] = keys
	}
	return result
}

func TestBuildDAG(t *testing.T) {
	svc := NewSchedulerService()

	tests := []struct {
		name       string
		stories    []model.Story
		wantGroups [][]string
		wantErr    string
	}{
		{
			name:       "empty input",
			stories:    []model.Story{},
			wantGroups: [][]string{},
		},
		{
			name:       "single story no deps",
			stories:    []model.Story{newStory("S-01", nil, nil)},
			wantGroups: [][]string{{"S-01"}},
		},
		{
			name: "two independent stories",
			stories: []model.Story{
				newStory("S-01", nil, nil),
				newStory("S-02", nil, nil),
			},
			wantGroups: [][]string{{"S-01", "S-02"}},
		},
		{
			name: "linear chain A→B→C",
			stories: []model.Story{
				newStory("S-01", nil, nil),
				newStory("S-02", []string{"S-01"}, nil),
				newStory("S-03", []string{"S-02"}, nil),
			},
			wantGroups: [][]string{{"S-01"}, {"S-02"}, {"S-03"}},
		},
		{
			name: "diamond A→B, A→C, B→D, C→D",
			stories: []model.Story{
				newStory("S-01", nil, nil),
				newStory("S-02", []string{"S-01"}, nil),
				newStory("S-03", []string{"S-01"}, nil),
				newStory("S-04", []string{"S-02", "S-03"}, nil),
			},
			wantGroups: [][]string{{"S-01"}, {"S-02", "S-03"}, {"S-04"}},
		},
		{
			name: "cycle of two A↔B",
			stories: []model.Story{
				newStory("S-01", []string{"S-02"}, nil),
				newStory("S-02", []string{"S-01"}, nil),
			},
			wantErr: "DAG_CYCLE_DETECTED",
		},
		{
			name: "cycle of three A→B→C→A",
			stories: []model.Story{
				newStory("S-01", []string{"S-03"}, nil),
				newStory("S-02", []string{"S-01"}, nil),
				newStory("S-03", []string{"S-02"}, nil),
			},
			wantErr: "DAG_CYCLE_DETECTED",
		},
		{
			name: "unknown dep key ignored",
			stories: []model.Story{
				newStory("S-01", nil, nil),
				newStory("S-02", []string{"GHOST-99"}, nil),
			},
			wantGroups: [][]string{{"S-01", "S-02"}},
		},
		{
			name: "file conflict implicit edge",
			stories: []model.Story{
				newStory("S-02", nil, []string{"shared.go"}),
				newStory("S-01", nil, []string{"shared.go"}),
			},
			wantGroups: [][]string{{"S-01"}, {"S-02"}},
		},
		{
			name: "combined explicit and file conflict",
			stories: []model.Story{
				newStory("S-01", nil, []string{"main.go"}),
				newStory("S-02", []string{"S-01"}, []string{"main.go"}),
				newStory("S-03", nil, nil),
			},
			wantGroups: [][]string{{"S-01", "S-03"}, {"S-02"}},
		},
		{
			name: "file conflict with three stories on same file",
			stories: []model.Story{
				newStory("S-03", nil, []string{"config.yaml"}),
				newStory("S-01", nil, []string{"config.yaml"}),
				newStory("S-02", nil, []string{"config.yaml"}),
			},
			wantGroups: [][]string{{"S-01"}, {"S-02"}, {"S-03"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.BuildDAG(tt.stories)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error with code %s, got nil", tt.wantErr)
				}
				domainErr, ok := err.(*errors.DomainError)
				if !ok {
					t.Fatalf("expected *errors.DomainError, got %T", err)
				}
				if domainErr.Code != tt.wantErr {
					t.Errorf("expected error code %s, got %s", tt.wantErr, domainErr.Code)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			gotGroups := extractKeys(result.Groups)
			if len(gotGroups) != len(tt.wantGroups) {
				t.Fatalf("expected %d groups, got %d: %v", len(tt.wantGroups), len(gotGroups), gotGroups)
			}

			for i, wantGroup := range tt.wantGroups {
				if len(gotGroups[i]) != len(wantGroup) {
					t.Errorf("group %d: expected %d stories, got %d: %v", i, len(wantGroup), len(gotGroups[i]), gotGroups[i])
					continue
				}
				for j, wantKey := range wantGroup {
					if gotGroups[i][j] != wantKey {
						t.Errorf("group %d, position %d: expected %s, got %s", i, j, wantKey, gotGroups[i][j])
					}
				}
			}
		})
	}
}
