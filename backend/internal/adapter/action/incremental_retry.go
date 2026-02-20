package action

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/internal/domain/service"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// AgentRunExecutor is an interface to allow mocking AgentRunAction in tests.
type AgentRunExecutor interface {
	Execute(ctx context.Context, runCtx *model.RunContext) error
}

// IncrementalRetryAction coordinates retry logic for failed agent steps.
// It creates a new RunStep record and delegates execution to an AgentRunExecutor.
// After max_incremental incremental retries, it falls back to a full retry
// using the original implement template without error context injection.
type IncrementalRetryAction struct {
	runRepo  port.RunRepository
	agentRun AgentRunExecutor
	logger   *slog.Logger
}

// NewIncrementalRetryAction creates a new IncrementalRetryAction.
func NewIncrementalRetryAction(
	runRepo port.RunRepository,
	agentRun AgentRunExecutor,
	logger *slog.Logger,
) *IncrementalRetryAction {
	return &IncrementalRetryAction{
		runRepo:  runRepo,
		agentRun: agentRun,
		logger:   logger,
	}
}

// Name returns the action identifier.
func (a *IncrementalRetryAction) Name() string {
	return "incremental_retry"
}

// Execute fetches the parent step, evaluates the retry policy, creates a new
// retry RunStep, and delegates execution to the agent run executor.
func (a *IncrementalRetryAction) Execute(ctx context.Context, runCtx *model.RunContext) error {
	// 1. Extract parent step ID from metadata
	parentStepIDStr, ok := runCtx.Metadata["parent_step_id"].(string)
	if !ok || parentStepIDStr == "" {
		return &errors.DomainError{
			Category: errors.CategoryValidation,
			Code:     "RETRY_MISSING_PARENT",
			Message:  "missing required metadata key parent_step_id",
		}
	}
	parentStepID, err := uuid.Parse(parentStepIDStr)
	if err != nil {
		return &errors.DomainError{
			Category: errors.CategoryValidation,
			Code:     "RETRY_MISSING_PARENT",
			Message:  fmt.Sprintf("invalid UUID format for parent_step_id: %s", parentStepIDStr),
		}
	}

	// 2. Fetch parent step
	parent, err := a.runRepo.GetRunStep(ctx, parentStepID)
	if err != nil {
		return fmt.Errorf("fetch parent step: %w", err)
	}

	// 3. Resolve retry policy from metadata
	maxRetries := intFromMetadata(runCtx.Metadata, "retry_policy.max_retries", 3)
	maxIncremental := intFromMetadata(runCtx.Metadata, "retry_policy.max_incremental", 2)

	// 4. Check max retries
	if parent.RetryCount >= maxRetries {
		return &errors.DomainError{
			Category: errors.CategoryValidation,
			Code:     "RETRY_MAX_EXCEEDED",
			Message:  fmt.Sprintf("max %d retries reached for step %s", maxRetries, parent.ID),
		}
	}

	// 5. Determine retry type and template
	retryType := "incremental"
	templateName := service.TemplateNameImplementRetry
	if parent.RetryCount >= maxIncremental {
		retryType = "full"
		templateName = service.TemplateNameImplement
	}

	a.logger.Info("creating retry step",
		"parent_step_id", parent.ID,
		"retry_count", parent.RetryCount+1,
		"retry_type", retryType,
		"template", templateName,
	)

	// 6. Create new RunStep
	newStep := &model.RunStep{
		ID:           uuid.New(),
		RunID:        parent.RunID,
		StepName:     parent.StepName,
		StepOrder:    parent.StepOrder,
		Action:       parent.Action,
		Status:       model.StepStatusPending,
		RetryCount:   parent.RetryCount + 1,
		RetryType:    &retryType,
		ParentStepID: &parent.ID,
	}
	created, err := a.runRepo.CreateRetryRunStep(ctx, newStep)
	if err != nil {
		return fmt.Errorf("create retry step: %w", err)
	}

	// 7. Build new RunContext with retry metadata
	errorContext := ""
	if parent.ErrorMessage != nil {
		errorContext = *parent.ErrorMessage
	}
	logTail := ""
	if parent.LogTail != nil {
		logTail = *parent.LogTail
	}

	newMetadata := make(map[string]any, len(runCtx.Metadata))
	for k, v := range runCtx.Metadata {
		newMetadata[k] = v
	}
	newMetadata["template_name"] = templateName
	newMetadata["error_context"] = errorContext
	newMetadata["log_tail"] = logTail

	newRunCtx := &model.RunContext{
		Run:       runCtx.Run,
		RunStep:   created,
		StoryID:   runCtx.StoryID,
		ProjectID: runCtx.ProjectID,
		Metadata:  newMetadata,
	}

	// 8. Delegate to agent run executor
	return a.agentRun.Execute(ctx, newRunCtx)
}

// intFromMetadata reads an integer value from metadata with a default fallback.
func intFromMetadata(metadata map[string]any, key string, defaultVal int) int {
	v, ok := metadata[key]
	if !ok {
		return defaultVal
	}
	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	default:
		return defaultVal
	}
}
