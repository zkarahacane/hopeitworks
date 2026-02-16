package handler

import "net/http"

// Server implements the generated ServerInterface, delegating to specific handlers.
// It embeds Unimplemented to satisfy methods that are not yet implemented.
type Server struct {
	Unimplemented
	projects *ProjectHandler
	users    *UserHandler
}

// NewServer creates a new Server with the given handlers.
func NewServer(projects *ProjectHandler, users *UserHandler) *Server {
	return &Server{projects: projects, users: users}
}

// ListProjects delegates to ProjectHandler.
func (s *Server) ListProjects(w http.ResponseWriter, r *http.Request, params ListProjectsParams) {
	s.projects.ListProjects(w, r, params)
}

// CreateProject delegates to ProjectHandler.
func (s *Server) CreateProject(w http.ResponseWriter, r *http.Request) {
	s.projects.CreateProject(w, r)
}

// GetProject delegates to ProjectHandler.
func (s *Server) GetProject(w http.ResponseWriter, r *http.Request, id IdPath) {
	s.projects.GetProject(w, r, id)
}

// UpdateProject delegates to ProjectHandler.
func (s *Server) UpdateProject(w http.ResponseWriter, r *http.Request, id IdPath) {
	s.projects.UpdateProject(w, r, id)
}

// DeleteProject delegates to ProjectHandler.
func (s *Server) DeleteProject(w http.ResponseWriter, r *http.Request, id IdPath) {
	s.projects.DeleteProject(w, r, id)
}

// ListUsers delegates to UserHandler.
func (s *Server) ListUsers(w http.ResponseWriter, r *http.Request, params ListUsersParams) {
	s.users.ListUsers(w, r, params)
}

// GetUser delegates to UserHandler.
func (s *Server) GetUser(w http.ResponseWriter, r *http.Request, id IdPath) {
	s.users.GetUser(w, r, id)
}

// UpdateUser delegates to UserHandler.
func (s *Server) UpdateUser(w http.ResponseWriter, r *http.Request, id IdPath) {
	s.users.UpdateUser(w, r, id)
}

// DeleteUser delegates to UserHandler.
func (s *Server) DeleteUser(w http.ResponseWriter, r *http.Request, id IdPath) {
	s.users.DeleteUser(w, r, id)
}
