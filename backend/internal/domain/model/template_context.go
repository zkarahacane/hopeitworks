package model

// TemplateContext provides variables available to Handlebars prompt templates.
// All fields are exported and JSON-tagged for serialization and template rendering.
type TemplateContext struct {
	// StoryKey is the story identifier (e.g., "S-42").
	StoryKey string `json:"story_key"`
	// StoryTitle is the story summary/title.
	StoryTitle string `json:"story_title"`
	// StoryObjective is the story description/objective.
	StoryObjective string `json:"story_objective"`
	// TargetFiles lists files to modify (for implement templates).
	TargetFiles []string `json:"target_files"`
	// AcceptanceCriteria contains the acceptance criteria in BDD format.
	AcceptanceCriteria string `json:"acceptance_criteria"`
	// ErrorContext holds error details (for retry templates).
	ErrorContext string `json:"error_context"`
	// DiffContent holds git diff or changes (for review/merge templates).
	DiffContent string `json:"diff_content"`
	// BranchName is the git branch name.
	BranchName string `json:"branch_name"`
	// RepoURL is the repository URL.
	RepoURL string `json:"repo_url"`
}
