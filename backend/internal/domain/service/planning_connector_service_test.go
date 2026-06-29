package service

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	apperrors "github.com/zakari/hopeitworks/backend/pkg/errors"
)

func mappingDone() model.PlanningStatusMapping {
	return model.PlanningStatusMapping{Done: strptrT("OPT_DONE")}
}

// assertInvalidStateCode asserts err is a 422 DomainError with the expected code.
func assertInvalidStateCode(t *testing.T, err error, code string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error with code %q, got nil", code)
	}
	de, ok := err.(*apperrors.DomainError)
	if !ok {
		t.Fatalf("expected *DomainError, got %T: %v", err, err)
	}
	if de.Code != code {
		t.Fatalf("error code = %q, want %q", de.Code, code)
	}
}

func TestSetConnector_WritebackEnabled_NoToken_422(t *testing.T) {
	conn := &wbConnectorRepo{}
	svc := NewPlanningConnectorService(conn, nil, &wbResolver{token: ""}, &wbSinkFactory{})

	_, err := svc.SetConnector(context.Background(), uuid.New(), SetConnectorInput{
		Source:           string(port.SourceGitHub),
		WritebackEnabled: true,
		StatusMapping:    mappingDone(),
	})
	assertInvalidStateCode(t, err, CodePlanningConnectorNoGitConnection)
	if conn.upsertCnt != 0 {
		t.Fatalf("must not persist a rejected connector")
	}
}

func TestSetConnector_WritebackEnabled_EmptyMapping_422(t *testing.T) {
	conn := &wbConnectorRepo{}
	svc := NewPlanningConnectorService(conn, nil, &wbResolver{token: "ghp_token"}, &wbSinkFactory{})

	_, err := svc.SetConnector(context.Background(), uuid.New(), SetConnectorInput{
		Source:           string(port.SourceGitHub),
		WritebackEnabled: true,
		StatusMapping:    model.PlanningStatusMapping{}, // no target
	})
	assertInvalidStateCode(t, err, CodePlanningConnectorInvalidMapping)
	if conn.upsertCnt != 0 {
		t.Fatalf("must not persist a rejected connector")
	}
}

func TestSetConnector_WritebackEnabled_OK(t *testing.T) {
	conn := &wbConnectorRepo{}
	svc := NewPlanningConnectorService(conn, nil, &wbResolver{token: "ghp_token"}, &wbSinkFactory{})

	out, err := svc.SetConnector(context.Background(), uuid.New(), SetConnectorInput{
		Source:           string(port.SourceGitHub),
		WritebackEnabled: true,
		StatusMapping:    mappingDone(),
	})
	if err != nil {
		t.Fatalf("SetConnector: %v", err)
	}
	if conn.upsertCnt != 1 || conn.upserted == nil {
		t.Fatalf("expected one upsert")
	}
	// defaults applied
	if conn.upserted.StatusField != "Status" || conn.upserted.EpicIssueType != "Epic" {
		t.Fatalf("defaults not applied: %+v", conn.upserted)
	}
	if out == nil || !out.WritebackEnabled {
		t.Fatalf("unexpected result: %+v", out)
	}
}

func TestSetConnector_Disabled_SkipsValidation(t *testing.T) {
	conn := &wbConnectorRepo{}
	// no token, empty mapping — both are fine when write-back is OFF.
	svc := NewPlanningConnectorService(conn, nil, &wbResolver{token: ""}, &wbSinkFactory{})

	_, err := svc.SetConnector(context.Background(), uuid.New(), SetConnectorInput{
		Source:           string(port.SourceGitHub),
		WritebackEnabled: false,
		StatusMapping:    model.PlanningStatusMapping{},
	})
	if err != nil {
		t.Fatalf("disabled write-back must skip validation, got %v", err)
	}
	if conn.upsertCnt != 1 {
		t.Fatalf("expected the connector to persist")
	}
}

func TestStatusOptions_FromOverride_OK(t *testing.T) {
	conn := &wbConnectorRepo{getErr: notFound("planning_connector")} // no persisted connector
	sink := &wbSink{statusOpts: port.PlanningStatusOptions{
		FieldID:   "FIELD_1",
		FieldName: "Status",
		Options:   []port.PlanningStatusOption{{ID: "o1", Name: "Todo"}, {ID: "o2", Name: "Done"}},
	}}
	svc := NewPlanningConnectorService(conn, nil, &wbResolver{token: "ghp_token"}, &wbSinkFactory{sink: sink})

	url := "https://github.com/orgs/acme/projects/7"
	opts, err := svc.StatusOptions(context.Background(), uuid.New(), &url, nil)
	if err != nil {
		t.Fatalf("StatusOptions: %v", err)
	}
	if opts.FieldID != "FIELD_1" || len(opts.Options) != 2 {
		t.Fatalf("unexpected options: %+v", opts)
	}
}

func TestStatusOptions_FieldNotFound_422(t *testing.T) {
	conn := &wbConnectorRepo{getErr: notFound("planning_connector")}
	sink := &wbSink{statusErr: apperrors.NewInternal("boom", nil)}
	svc := NewPlanningConnectorService(conn, nil, &wbResolver{token: "ghp_token"}, &wbSinkFactory{sink: sink})

	url := "https://github.com/orgs/acme/projects/7"
	_, err := svc.StatusOptions(context.Background(), uuid.New(), &url, nil)
	assertInvalidStateCode(t, err, CodePlanningStatusFieldNotFound)
}
