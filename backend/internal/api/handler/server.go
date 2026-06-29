package handler

import "net/http"

// Server implements the generated ServerInterface, delegating to specific handlers.
// It embeds Unimplemented to satisfy methods that are not yet implemented.
type Server struct {
	Unimplemented
	auth              *AuthHandler
	projects          *ProjectHandler
	users             *UserHandler
	profile           *ProfileHandler
	epics             *EpicHandler
	stories           *StoryHandler
	agents            *AgentHandler
	stacks            *StackHandler
	runs              *RunHandler
	pipelineConfig    *PipelineConfigHandler
	hitl              *HITLHandler
	costs             *CostHandler
	notifications     *NotificationHandler
	epicRuns          *EpicRunHandler
	environment       *EnvironmentHandler
	apiKeys           *APIKeyHandler
	planning          *PlanningHandler
	planningConnector *PlanningConnectorHandler
	gitConnection     *GitConnectionHandler
}

// NewServer creates a new Server with the given handlers.
func NewServer(auth *AuthHandler, projects *ProjectHandler, users *UserHandler, profile *ProfileHandler, epics *EpicHandler, stories *StoryHandler, agents *AgentHandler, stacks *StackHandler, runs *RunHandler, pipelineConfig *PipelineConfigHandler, hitl *HITLHandler, costs *CostHandler, notifications *NotificationHandler, epicRuns *EpicRunHandler, environment *EnvironmentHandler, apiKeys *APIKeyHandler, planning *PlanningHandler, planningConnector *PlanningConnectorHandler, gitConnection *GitConnectionHandler) *Server {
	return &Server{auth: auth, projects: projects, users: users, profile: profile, epics: epics, stories: stories, agents: agents, stacks: stacks, runs: runs, pipelineConfig: pipelineConfig, hitl: hitl, costs: costs, notifications: notifications, epicRuns: epicRuns, environment: environment, apiKeys: apiKeys, planning: planning, planningConnector: planningConnector, gitConnection: gitConnection}
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

// ImportPlanning delegates to PlanningHandler.
func (s *Server) ImportPlanning(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath) {
	s.planning.ImportPlanning(w, r, projectID)
}

// GetPlanningConnector delegates to PlanningConnectorHandler.
func (s *Server) GetPlanningConnector(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath) {
	s.planningConnector.GetPlanningConnector(w, r, projectID)
}

// SetPlanningConnector delegates to PlanningConnectorHandler.
func (s *Server) SetPlanningConnector(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath) {
	s.planningConnector.SetPlanningConnector(w, r, projectID)
}

// GetPlanningStatusOptions delegates to PlanningConnectorHandler.
func (s *Server) GetPlanningStatusOptions(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, params GetPlanningStatusOptionsParams) {
	s.planningConnector.GetPlanningStatusOptions(w, r, projectID, params)
}

// ListGlobalAgents delegates to AgentHandler.
func (s *Server) ListGlobalAgents(w http.ResponseWriter, r *http.Request, params ListGlobalAgentsParams) {
	s.agents.ListGlobalAgents(w, r, params)
}

// ListProjectAgents delegates to AgentHandler.
func (s *Server) ListProjectAgents(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, params ListProjectAgentsParams) {
	s.agents.ListProjectAgents(w, r, projectID, params)
}

// CreateAgent delegates to AgentHandler.
func (s *Server) CreateAgent(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath) {
	s.agents.CreateAgent(w, r, projectID)
}

// GetAgent delegates to AgentHandler.
func (s *Server) GetAgent(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, agentID AgentIdPath) {
	s.agents.GetAgent(w, r, projectID, agentID)
}

// UpdateAgent delegates to AgentHandler.
func (s *Server) UpdateAgent(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, agentID AgentIdPath) {
	s.agents.UpdateAgent(w, r, projectID, agentID)
}

// DeleteAgent delegates to AgentHandler.
func (s *Server) DeleteAgent(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, agentID AgentIdPath) {
	s.agents.DeleteAgent(w, r, projectID, agentID)
}

// ListStacks delegates to StackHandler.
func (s *Server) ListStacks(w http.ResponseWriter, r *http.Request) {
	s.stacks.ListStacks(w, r)
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

// StartStage delegates to RunHandler.
func (s *Server) StartStage(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, storyID StoryIdPath) {
	s.runs.StartStage(w, r, projectID, storyID)
}

// ListRunsByStory delegates to RunHandler.
func (s *Server) ListRunsByStory(w http.ResponseWriter, r *http.Request, storyID StoryIdPath, params ListRunsByStoryParams) {
	s.runs.ListRunsByStory(w, r, storyID, params)
}

// RetryFailedStep delegates to RunHandler.
func (s *Server) RetryFailedStep(w http.ResponseWriter, r *http.Request, runID RunIdPath, stepID StepIdPath) {
	s.runs.RetryStep(w, r, runID, stepID)
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

// CancelRun delegates to RunHandler.
func (s *Server) CancelRun(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, runID RunIdPath) {
	s.runs.CancelRun(w, r, projectID, runID)
}

// CancelEpicRun delegates to RunHandler.
func (s *Server) CancelEpicRun(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, epicID EpicIdPath, runID RunIdPath) {
	s.runs.CancelEpicRun(w, r, projectID, epicID, runID)
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

// ResolveHITLRequest delegates to HITLHandler.
func (s *Server) ResolveHITLRequest(w http.ResponseWriter, r *http.Request, hitlRequestID HITLRequestIdPath) {
	s.hitl.ResolveHITLRequest(w, r, hitlRequestID)
}

// ListProbeHalts delegates to HITLHandler.
func (s *Server) ListProbeHalts(w http.ResponseWriter, r *http.Request, params ListProbeHaltsParams) {
	s.hitl.ListProbeHalts(w, r, params)
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

// GetRunCostsByRole delegates to CostHandler.
func (s *Server) GetRunCostsByRole(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, runID RunIdPath) {
	s.costs.GetRunCostsByRole(w, r, projectID, runID)
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

// ResetCircuitBreaker delegates to ProjectHandler.
func (s *Server) ResetCircuitBreaker(w http.ResponseWriter, r *http.Request, id IdPath) {
	s.projects.ResetCircuitBreaker(w, r, id)
}

// ListHITLRequests delegates to HITLHandler.
func (s *Server) ListHITLRequests(w http.ResponseWriter, r *http.Request, params ListHITLRequestsParams) {
	s.hitl.ListHITLRequests(w, r, params)
}

// GetHITLRequestByStep delegates to HITLHandler.
func (s *Server) GetHITLRequestByStep(w http.ResponseWriter, r *http.Request, stepID StepIdPath) {
	s.hitl.GetHITLRequestByStep(w, r, stepID)
}

// GetProjectCostsByAgent delegates to CostHandler.
func (s *Server) GetProjectCostsByAgent(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath) {
	s.costs.GetProjectCostsByAgent(w, r, projectID)
}

// GetProjectCostsByRole delegates to CostHandler.
func (s *Server) GetProjectCostsByRole(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath) {
	s.costs.GetProjectCostsByRole(w, r, projectID)
}

// GetProjectCostChart delegates to CostHandler.
func (s *Server) GetProjectCostChart(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, params GetProjectCostChartParams) {
	s.costs.GetProjectCostChart(w, r, projectID, params)
}

// GetProjectCostRuns delegates to CostHandler.
func (s *Server) GetProjectCostRuns(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, params GetProjectCostRunsParams) {
	s.costs.GetProjectCostRuns(w, r, projectID, params)
}

// TestNotificationConfig delegates to NotificationHandler.
func (s *Server) TestNotificationConfig(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, notificationID NotificationIdPath) {
	s.notifications.TestNotificationConfig(w, r, projectID, notificationID)
}

// ListPromptTemplates delegates to AgentHandler for backward compatibility.
// Deprecated: use ListProjectAgents instead.
func (s *Server) ListPromptTemplates(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, params ListPromptTemplatesParams) {
	s.agents.ListProjectAgents(w, r, projectID, ListProjectAgentsParams(params))
}

// CreatePromptTemplate delegates to AgentHandler for backward compatibility.
// Deprecated: use CreateAgent instead.
func (s *Server) CreatePromptTemplate(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath) {
	s.agents.CreateAgent(w, r, projectID)
}

// GetPromptTemplate delegates to AgentHandler for backward compatibility.
// Deprecated: use GetAgent instead.
func (s *Server) GetPromptTemplate(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, templateID TemplateIdPath) {
	s.agents.GetAgent(w, r, projectID, templateID)
}

// UpdatePromptTemplate delegates to AgentHandler for backward compatibility.
// Deprecated: use UpdateAgent instead.
func (s *Server) UpdatePromptTemplate(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, templateID TemplateIdPath) {
	s.agents.UpdateAgent(w, r, projectID, templateID)
}

// DeletePromptTemplate delegates to AgentHandler for backward compatibility.
// Deprecated: use DeleteAgent instead.
func (s *Server) DeletePromptTemplate(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, templateID TemplateIdPath) {
	s.agents.DeleteAgent(w, r, projectID, templateID)
}

// GetProjectEnvironment delegates to EnvironmentHandler.
func (s *Server) GetProjectEnvironment(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath) {
	s.environment.GetProjectEnvironment(w, r, projectID)
}

// PutProjectEnvironment delegates to EnvironmentHandler.
func (s *Server) PutProjectEnvironment(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath) {
	s.environment.PutProjectEnvironment(w, r, projectID)
}

// DeleteProjectEnvironment delegates to EnvironmentHandler.
func (s *Server) DeleteProjectEnvironment(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath) {
	s.environment.DeleteProjectEnvironment(w, r, projectID)
}

// ListMyAPIKeys delegates to APIKeyHandler.
func (s *Server) ListMyAPIKeys(w http.ResponseWriter, r *http.Request) {
	s.apiKeys.ListMyAPIKeys(w, r)
}

// CreateMyAPIKey delegates to APIKeyHandler.
func (s *Server) CreateMyAPIKey(w http.ResponseWriter, r *http.Request) {
	s.apiKeys.CreateMyAPIKey(w, r)
}

// DeleteMyAPIKey delegates to APIKeyHandler.
func (s *Server) DeleteMyAPIKey(w http.ResponseWriter, r *http.Request, keyID KeyIdPath) {
	s.apiKeys.DeleteMyAPIKey(w, r, keyID)
}

// GetProjectGitConnection delegates to GitConnectionHandler.
func (s *Server) GetProjectGitConnection(w http.ResponseWriter, r *http.Request, id IdPath) {
	s.gitConnection.GetProjectGitConnection(w, r, id)
}

// SetProjectGitConnection delegates to GitConnectionHandler.
func (s *Server) SetProjectGitConnection(w http.ResponseWriter, r *http.Request, id IdPath) {
	s.gitConnection.SetProjectGitConnection(w, r, id)
}

// ClearProjectGitConnection delegates to GitConnectionHandler.
func (s *Server) ClearProjectGitConnection(w http.ResponseWriter, r *http.Request, id IdPath) {
	s.gitConnection.ClearProjectGitConnection(w, r, id)
}

// TestProjectGitConnection delegates to GitConnectionHandler.
func (s *Server) TestProjectGitConnection(w http.ResponseWriter, r *http.Request, id IdPath) {
	s.gitConnection.TestProjectGitConnection(w, r, id)
}
