package postgres

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	apperrors "github.com/zakari/hopeitworks/backend/pkg/errors"
)

// currentStepJSON mirrors the JSON object produced by to_jsonb in the
// GetLatestRun* queries.
type currentStepJSON struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	ActionType  string    `json:"action_type"`
	Status      string    `json:"status"`
	Index       int       `json:"index"`
	ContainerID *string   `json:"container_id"`
}

// toDomainLatestRun maps the latest-run query columns into a domain LatestRun,
// decoding the optional current-step JSON blob (NULL when no step is in progress).
func toDomainLatestRun(runID uuid.UUID, runStatus string, currentStep []byte, totalSteps int32) (*model.LatestRun, error) {
	latest := &model.LatestRun{
		ID:     runID,
		Status: runStatus,
	}
	if len(currentStep) > 0 {
		var cs currentStepJSON
		if err := json.Unmarshal(currentStep, &cs); err != nil {
			return nil, apperrors.NewInternal("failed to unmarshal current step", err)
		}
		latest.CurrentStep = &model.LatestRunStep{
			ID:          cs.ID,
			Name:        cs.Name,
			ActionType:  cs.ActionType,
			Status:      cs.Status,
			Index:       cs.Index,
			Total:       int(totalSteps),
			ContainerID: cs.ContainerID,
		}
	}
	return latest, nil
}

// toDomainRun maps a sqlc-generated Run to a domain Run.
func toDomainRun(r Run) *model.Run {
	return toDomainRunWithStoryKey(r.ID, r.ProjectID, r.StoryID, r.Status,
		r.PipelineConfigSnapshot, r.StartedAt, r.CompletedAt, r.ErrorMessage,
		r.CreatedAt, r.UpdatedAt, r.PausedAt, r.Metadata, "")
}

// toDomainRunWithStoryKey maps run fields (including an optional story key from a JOIN) to a domain Run.
func toDomainRunWithStoryKey(
	id, projectID, storyID uuid.UUID,
	status string,
	pipelineConfigSnapshot []byte,
	startedAt, completedAt pgtype.Timestamptz,
	errorMessage pgtype.Text,
	createdAt, updatedAt time.Time,
	pausedAt pgtype.Timestamptz,
	metadata []byte,
	storyKey string,
) *model.Run {
	run := &model.Run{
		ID:                     id,
		ProjectID:              projectID,
		StoryID:                storyID,
		StoryKey:               storyKey,
		Status:                 model.RunStatus(status),
		PipelineConfigSnapshot: json.RawMessage(pipelineConfigSnapshot),
		CreatedAt:              createdAt,
		UpdatedAt:              updatedAt,
	}
	if len(metadata) > 0 {
		var meta map[string]interface{}
		if err := json.Unmarshal(metadata, &meta); err == nil {
			run.Metadata = meta
		}
	}
	if startedAt.Valid {
		run.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		run.CompletedAt = &completedAt.Time
	}
	if pausedAt.Valid {
		run.PausedAt = &pausedAt.Time
	}
	if errorMessage.Valid {
		run.ErrorMessage = &errorMessage.String
	}
	return run
}

// toDomainRunStep maps a sqlc-generated RunStep to a domain RunStep.
func toDomainRunStep(s RunStep) *model.RunStep {
	step := &model.RunStep{
		ID:         s.ID,
		RunID:      s.RunID,
		StepName:   s.StepName,
		StepOrder:  int(s.StepOrder),
		Action:     s.Action,
		Status:     model.StepStatus(s.Status),
		RetryCount: int(s.RetryCount),
		CreatedAt:  s.CreatedAt,
	}
	if s.StageID.Valid {
		step.StageID = s.StageID.String
	}
	if s.StageName.Valid {
		step.StageName = s.StageName.String
	}
	if s.StartedAt.Valid {
		step.StartedAt = &s.StartedAt.Time
	}
	if s.CompletedAt.Valid {
		step.CompletedAt = &s.CompletedAt.Time
	}
	if s.ErrorMessage.Valid {
		step.ErrorMessage = &s.ErrorMessage.String
	}
	if s.ContainerID.Valid {
		step.ContainerID = &s.ContainerID.String
	}
	if s.LogTail.Valid {
		step.LogTail = &s.LogTail.String
	}
	if s.RetryType.Valid {
		step.RetryType = &s.RetryType.String
	}
	if s.ParentStepID.Valid {
		uid := uuid.UUID(s.ParentStepID.Bytes)
		step.ParentStepID = &uid
	}
	return step
}
