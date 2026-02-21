package model

// DAGResult holds the result of a topological sort on stories.
// Groups are execution layers: all stories in Groups[i] can run concurrently,
// and all must complete before any story in Groups[i+1] starts.
type DAGResult struct {
	Groups [][]Story
}
