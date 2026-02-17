package handler

import "net/http"

// Server implements the generated ServerInterface, delegating to specific handlers.
// It embeds Unimplemented to satisfy methods that are not yet implemented.
type Server struct {
	Unimplemented
	auth     *AuthHandler
	projects *ProjectHandler
	users    *UserHandler
	epics    *EpicHandler
	runs     *RunHandler
}

// NewServer creates a new Server with the given handlers.
func NewServer(auth *AuthHandler, projects *ProjectHandler, users *UserHandler, epics *EpicHandler, runs *RunHandler) *Server {
	return &Server{auth: auth, projects: projects, users: users, epics: epics, runs: runs}
}

// RegisterUser delegates to AuthHandler.
func (s *Server) RegisterUser(w http.ResponseWriter, r *http.Request) {
	s.auth.Register(w, r)
}

// LoginUser delegates to AuthHandler.
func (s *Server) LoginUser(w http.ResponseWriter, r *http.Request) {
	s.auth.Login(w, r)
}

// LogoutUser delegates to AuthHandler.
func (s *Server) LogoutUser(w http.ResponseWriter, r *http.Request) {
	s.auth.Logout(w, r)
}

// GetCurrentUser delegates to AuthHandler.
func (s *Server) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	s.auth.Me(w, r)
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

// ListEpics delegates to EpicHandler.
func (s *Server) ListEpics(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, params ListEpicsParams) {
	s.epics.ListEpics(w, r, projectID, params)
}

// CreateEpic delegates to EpicHandler.
func (s *Server) CreateEpic(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath) {
	s.epics.CreateEpic(w, r, projectID)
}

// GetEpic delegates to EpicHandler.
func (s *Server) GetEpic(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, epicID EpicIdPath) {
	s.epics.GetEpic(w, r, projectID, epicID)
}

// UpdateEpic delegates to EpicHandler.
func (s *Server) UpdateEpic(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, epicID EpicIdPath) {
	s.epics.UpdateEpic(w, r, projectID, epicID)
}

// DeleteEpic delegates to EpicHandler.
func (s *Server) DeleteEpic(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, epicID EpicIdPath) {
	s.epics.DeleteEpic(w, r, projectID, epicID)
}

// ListRunsByProject delegates to RunHandler.
func (s *Server) ListRunsByProject(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, params ListRunsByProjectParams) {
	s.runs.ListRunsByProject(w, r, projectID, params)
}

// CreateRun delegates to RunHandler.
func (s *Server) CreateRun(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath) {
	s.runs.CreateRun(w, r, projectID)
}

// GetRun delegates to RunHandler.
func (s *Server) GetRun(w http.ResponseWriter, r *http.Request, runID RunIdPath) {
	s.runs.GetRun(w, r, runID)
}

// ListRunsByStory delegates to RunHandler.
func (s *Server) ListRunsByStory(w http.ResponseWriter, r *http.Request, storyID StoryIdPath, params ListRunsByStoryParams) {
	s.runs.ListRunsByStory(w, r, storyID, params)
}
