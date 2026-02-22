package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/service"
)

const codeNotImplemented = "NOT_IMPLEMENTED"

func TestGetProjectCostChart_ReturnsNotImplemented(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	costSvc := service.NewCostService(nil, nil, nil, nil, logger)
	h := NewCostHandler(costSvc)

	projectID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/costs/chart", nil)
	rec := httptest.NewRecorder()
	h.GetProjectCostChart(rec, req, ProjectIdPath(projectID), GetProjectCostChartParams{})

	if rec.Code != http.StatusNotImplemented {
		t.Errorf("expected 501, got %d", rec.Code)
	}

	var resp errorEnvelope
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Error.Code != codeNotImplemented {
		t.Errorf("expected error code NOT_IMPLEMENTED, got %s", resp.Error.Code)
	}
}

func TestGetProjectCostRuns_ReturnsNotImplemented(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	costSvc := service.NewCostService(nil, nil, nil, nil, logger)
	h := NewCostHandler(costSvc)

	projectID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/costs/runs", nil)
	rec := httptest.NewRecorder()
	h.GetProjectCostRuns(rec, req, ProjectIdPath(projectID), GetProjectCostRunsParams{})

	if rec.Code != http.StatusNotImplemented {
		t.Errorf("expected 501, got %d", rec.Code)
	}

	var resp errorEnvelope
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Error.Code != codeNotImplemented {
		t.Errorf("expected error code NOT_IMPLEMENTED, got %s", resp.Error.Code)
	}
}

func TestTestNotificationConfig_ReturnsNotImplemented(t *testing.T) {
	notifSvc := service.NewNotificationConfigService(nil)
	h := NewNotificationHandler(notifSvc)

	projectID := uuid.New()
	notificationID := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/notifications/"+notificationID.String()+"/test", nil)
	rec := httptest.NewRecorder()
	h.TestNotificationConfig(rec, req, ProjectIdPath(projectID), NotificationIdPath(notificationID))

	if rec.Code != http.StatusNotImplemented {
		t.Errorf("expected 501, got %d", rec.Code)
	}

	var resp errorEnvelope
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Error.Code != codeNotImplemented {
		t.Errorf("expected error code NOT_IMPLEMENTED, got %s", resp.Error.Code)
	}
}
