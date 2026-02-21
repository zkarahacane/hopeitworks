package handler

import "net/http"

// Server implements the generated ServerInterface, delegating to specific handlers.
// It embeds Unimplemented to satisfy methods that are not yet implemented.
type Server struct {
	Unimplemented
	auth            *AuthHandler
	projects        *ProjectHandler
	users           *UserHandler
	profile         *ProfileHandler
	epics           *EpicHandler
	stories         *StoryHandler
	promptTemplates *PromptTemplateHandler
	runs            *RunHandler
	pipelineConfig  *PipelineConfigHandler
	hitl            *HITLHandler
	costs           *CostHandler
	notifications   *NotificationHandler
	epicRuns        *EpicRunHandler
}

// NewServer creates a new Server with the given handlers.
func NewServer(auth *AuthHandler, projects *ProjectHandler, users *UserHandler, profile *ProfileHandler, epics *EpicHandler, stories *StoryHandler, promptTemplates *PromptTemplateHandler, runs *RunHandler, pipelineConfig *PipelineConfigHandler, hitl *HITLHandler, costs *CostHandler, notifications *NotificationHandler, epicRuns *EpicRunHandler) *Server {
	return &Server{auth: auth, projects: projects, users: users, profile: profile, epics: epics, stories: stories, promptTemplates: promptTemplates, runs: runs, pipelineConfig: pipelineConfig, hitl: hitl, costs: costs, notifications: notifications, epicRuns: epicRuns}
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

// ForgotPassword delegates to AuthHandler.
func (s *Server) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	s.auth.ForgotPassword(w, r)
}

// ResetPassword delegates to AuthHandler.
func (s *Server) ResetPassword(w http.ResponseWriter, r *http.Request) {
	s.auth.ResetPassword(w, r)
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

// GetMyProfile delegates to ProfileHandler.
func (s *Server) GetMyProfile(w http.ResponseWriter, r *http.Request) {
	s.profile.GetMyProfile(w, r)
}

// UpdateMyProfile delegates to ProfileHandler.
func (s *Server) UpdateMyProfile(w http.ResponseWriter, r *http.Request) {
	s.profile.UpdateMyProfile(w, r)
}

// ChangeMyPassword delegates to ProfileHandler.
func (s *Server) ChangeMyPassword(w http.ResponseWriter, r *http.Request) {
	s.profile.ChangeMyPassword(w, r)
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

// GetEpicDAG delegates to EpicHandler.
func (s *Server) GetEpicDAG(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, epicID EpicIdPath) {
	s.epics.GetEpicDAG(w, r, projectID, epicID)
}

// LaunchEpicRun delegates to EpicRunHandler.
func (s *Server) LaunchEpicRun(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, epicID EpicIdPath) {
	s.epicRuns.LaunchEpicRun(w, r, projectID, epicID)
}

// GetEpicRun delegates to EpicRunHandler.
func (s *Server) GetEpicRun(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, epicRunID EpicRunIdPath) {
	s.epicRuns.GetEpicRun(w, r, projectID, epicRunID)
}

// ListStories delegates to StoryHandler.
func (s *Server) ListStories(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, params ListStoriesParams) {
	s.stories.ListStories(w, r, projectID, params)
}

// CreateStory delegates to StoryHandler.
func (s *Server) CreateStory(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath) {
	s.stories.CreateStory(w, r, projectID)
}

// GetStory delegates to StoryHandler.
func (s *Server) GetStory(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, storyID StoryIdPath) {
	s.stories.GetStory(w, r, projectID, storyID)
}

// UpdateStory delegates to StoryHandler.
func (s *Server) UpdateStory(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, storyID StoryIdPath) {
	s.stories.UpdateStory(w, r, projectID, storyID)
}

// DeleteStory delegates to StoryHandler.
func (s *Server) DeleteStory(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, storyID StoryIdPath) {
	s.stories.DeleteStory(w, r, projectID, storyID)
}

// ImportStories delegates to StoryHandler.
func (s *Server) ImportStories(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath) {
	s.stories.ImportStories(w, r, projectID)
}

// ListPromptTemplates delegates to PromptTemplateHandler.
func (s *Server) ListPromptTemplates(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, params ListPromptTemplatesParams) {
	s.promptTemplates.ListPromptTemplates(w, r, projectID, params)
}

// CreatePromptTemplate delegates to PromptTemplateHandler.
func (s *Server) CreatePromptTemplate(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath) {
	s.promptTemplates.CreatePromptTemplate(w, r, projectID)
}

// GetPromptTemplate delegates to PromptTemplateHandler.
func (s *Server) GetPromptTemplate(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, templateID TemplateIdPath) {
	s.promptTemplates.GetPromptTemplate(w, r, projectID, templateID)
}

// UpdatePromptTemplate delegates to PromptTemplateHandler.
func (s *Server) UpdatePromptTemplate(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, templateID TemplateIdPath) {
	s.promptTemplates.UpdatePromptTemplate(w, r, projectID, templateID)
}

// DeletePromptTemplate delegates to PromptTemplateHandler.
func (s *Server) DeletePromptTemplate(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, templateID TemplateIdPath) {
	s.promptTemplates.DeletePromptTemplate(w, r, projectID, templateID)
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

// LaunchRun delegates to RunHandler.
func (s *Server) LaunchRun(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, storyID StoryIdPath) {
	s.runs.LaunchRun(w, r, projectID, storyID)
}

// ListRunsByStory delegates to RunHandler.
func (s *Server) ListRunsByStory(w http.ResponseWriter, r *http.Request, storyID StoryIdPath, params ListRunsByStoryParams) {
	s.runs.ListRunsByStory(w, r, storyID, params)
}

// PauseRun delegates to RunHandler.
func (s *Server) PauseRun(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, runID RunIdPath) {
	s.runs.PauseRun(w, r, projectID, runID)
}

// ResumeRun delegates to RunHandler.
func (s *Server) ResumeRun(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, runID RunIdPath) {
	s.runs.ResumeRun(w, r, projectID, runID)
}

// PauseEpicRun delegates to RunHandler.
func (s *Server) PauseEpicRun(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, epicID EpicIdPath, runID RunIdPath) {
	s.runs.PauseEpicRun(w, r, projectID, epicID, runID)
}

// ResumeEpicRun delegates to RunHandler.
func (s *Server) ResumeEpicRun(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, epicID EpicIdPath, runID RunIdPath) {
	s.runs.ResumeEpicRun(w, r, projectID, epicID, runID)
}

// GetPipelineConfig delegates to PipelineConfigHandler.
func (s *Server) GetPipelineConfig(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath) {
	s.pipelineConfig.GetPipelineConfig(w, r, projectID)
}

// UpdatePipelineConfig delegates to PipelineConfigHandler.
func (s *Server) UpdatePipelineConfig(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath) {
	s.pipelineConfig.UpdatePipelineConfig(w, r, projectID)
}

// ListPendingHITLRequests delegates to HITLHandler.
func (s *Server) ListPendingHITLRequests(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath) {
	s.hitl.ListPendingHITLRequests(w, r, projectID)
}

// GetHITLRequest delegates to HITLHandler.
func (s *Server) GetHITLRequest(w http.ResponseWriter, r *http.Request, hitlRequestID HITLRequestIdPath) {
	s.hitl.GetHITLRequest(w, r, hitlRequestID)
}

// ApproveHITLRequest delegates to HITLHandler.
func (s *Server) ApproveHITLRequest(w http.ResponseWriter, r *http.Request, hitlRequestID HITLRequestIdPath) {
	s.hitl.ApproveHITLRequest(w, r, hitlRequestID)
}

// RejectHITLRequest delegates to HITLHandler.
func (s *Server) RejectHITLRequest(w http.ResponseWriter, r *http.Request, hitlRequestID HITLRequestIdPath) {
	s.hitl.RejectHITLRequest(w, r, hitlRequestID)
}

// GetProjectCosts delegates to CostHandler.
func (s *Server) GetProjectCosts(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, params GetProjectCostsParams) {
	s.costs.GetProjectCosts(w, r, projectID, params)
}

// GetProjectCostSummary delegates to CostHandler.
func (s *Server) GetProjectCostSummary(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, params GetProjectCostSummaryParams) {
	s.costs.GetProjectCostSummary(w, r, projectID, params)
}

// GetStoryCosts delegates to CostHandler.
func (s *Server) GetStoryCosts(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, storyID StoryIdPath) {
	s.costs.GetStoryCosts(w, r, projectID, storyID)
}

// GetRunCosts delegates to CostHandler.
func (s *Server) GetRunCosts(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, runID RunIdPath) {
	s.costs.GetRunCosts(w, r, projectID, runID)
}

// ListNotificationConfigs delegates to NotificationHandler.
func (s *Server) ListNotificationConfigs(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath) {
	s.notifications.ListNotificationConfigs(w, r, projectID)
}

// CreateNotificationConfig delegates to NotificationHandler.
func (s *Server) CreateNotificationConfig(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath) {
	s.notifications.CreateNotificationConfig(w, r, projectID)
}

// UpdateNotificationConfig delegates to NotificationHandler.
func (s *Server) UpdateNotificationConfig(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, notificationID NotificationIdPath) {
	s.notifications.UpdateNotificationConfig(w, r, projectID, notificationID)
}

// DeleteNotificationConfig delegates to NotificationHandler.
func (s *Server) DeleteNotificationConfig(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, notificationID NotificationIdPath) {
	s.notifications.DeleteNotificationConfig(w, r, projectID, notificationID)
}
