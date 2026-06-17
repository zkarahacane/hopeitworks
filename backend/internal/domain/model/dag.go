package model

import "github.com/google/uuid"

// DAGResult holds the result of a topological sort on stories.
// Groups are execution layers: all stories in Groups[i] can run concurrently,
// and all must complete before any story in Groups[i+1] starts.
type DAGResult struct {
	Groups [][]Story
}

// DAGNodeRunInfo enriches a DAG node with data from the story's latest run:
// the run id and status, the most relevant container id of that run, and the
// total cost incurred. Returned keyed by story ID; stories with no run have no
// entry. ContainerID is nil when the latest run has no container attached.
type DAGNodeRunInfo struct {
	RunID       uuid.UUID
	RunStatus   string
	ContainerID *string
	CostUSD     float64
}
