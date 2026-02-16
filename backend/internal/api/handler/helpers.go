package handler

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// writeErrorResponse maps a domain error to an HTTP error response.
func writeErrorResponse(w http.ResponseWriter, err error) {
	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		domainErr = errors.NewInternal("unexpected error", err)
	}

	status := mapCategoryToStatus(domainErr.Category)
	resp := Error{
		Error: struct {
			Code    string                  `json:"code"`
			Details *map[string]interface{} `json:"details,omitempty"`
			Message string                  `json:"message"`
		}{
			Code:    domainErr.Code,
			Message: domainErr.Message,
		},
	}

	writeJSON(w, status, resp)
}

func mapCategoryToStatus(cat errors.ErrorCategory) int {
	switch cat {
	case errors.CategoryNotFound:
		return http.StatusNotFound
	case errors.CategoryValidation:
		return http.StatusBadRequest
	case errors.CategoryConflict:
		return http.StatusConflict
	case errors.CategoryUnauthorized:
		return http.StatusUnauthorized
	case errors.CategoryForbidden:
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}

// toAPIProject converts a domain Project to the API Project type.
func toAPIProject(p *model.Project) Project {
	proj := Project{
		Id:        p.ID,
		Name:      p.Name,
		OwnerId:   uuid.Nil,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
	if p.Description != nil {
		proj.Description = p.Description
	}
	if p.OwnerID != nil {
		proj.OwnerId = *p.OwnerID
	}
	return proj
}
