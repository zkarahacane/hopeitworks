package postgres

import (
	"context"
	"time"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	apperrors "github.com/zakari/hopeitworks/backend/pkg/errors"
)

// Ensure WatchdogRepo implements port.WatchdogRepository at compile time.
var _ port.WatchdogRepository = (*WatchdogRepo)(nil)

// WatchdogRepo implements port.WatchdogRepository using sqlc-generated queries.
type WatchdogRepo struct {
	queries *Queries
}

// NewWatchdogRepo creates a new WatchdogRepo.
func NewWatchdogRepo(queries *Queries) *WatchdogRepo {
	return &WatchdogRepo{queries: queries}
}

// ListRunningSteps returns every running step of a running run across all
// projects, with its last log-event timestamp and start time.
func (r *WatchdogRepo) ListRunningSteps(ctx context.Context) ([]*model.RunningStep, error) {
	rows, err := r.queries.ListRunningStepsForWatchdog(ctx)
	if err != nil {
		return nil, apperrors.NewInternal("failed to list running steps for watchdog", err)
	}
	result := make([]*model.RunningStep, len(rows))
	for i, row := range rows {
		rs := &model.RunningStep{
			StepID:    row.StepID,
			RunID:     row.RunID,
			StepName:  row.StepName,
			ProjectID: row.ProjectID,
			StoryID:   row.StoryID,
		}
		if row.StageID.Valid {
			rs.StageID = row.StageID.String
		}
		if row.StageName.Valid {
			rs.StageName = row.StageName.String
		}
		if row.StartedAt.Valid {
			t := row.StartedAt.Time
			rs.StartedAt = &t
		}
		// last_log_at comes back as interface{} because sqlc cannot infer the type
		// of max() through the LATERAL join: nil when the step has no log yet, a
		// time.Time when it has emitted at least one log.emitted event.
		if t, ok := row.LastLogAt.(time.Time); ok {
			rs.LastLogAt = &t
		}
		result[i] = rs
	}
	return result, nil
}
