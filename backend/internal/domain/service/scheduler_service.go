package service

import (
	"sort"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// SchedulerService provides DAG computation for story execution ordering.
type SchedulerService struct{}

// NewSchedulerService creates a new SchedulerService.
func NewSchedulerService() *SchedulerService {
	return &SchedulerService{}
}

// BuildDAG computes topological execution layers for the given stories.
// Returns DAGResult with Groups where each group can run in parallel.
// Returns DAG_CYCLE_DETECTED DomainError if a cycle is found.
func (s *SchedulerService) BuildDAG(stories []model.Story) (model.DAGResult, error) {
	if len(stories) == 0 {
		return model.DAGResult{Groups: [][]model.Story{}}, nil
	}

	// Index stories by key
	byKey := make(map[string]*model.Story, len(stories))
	keys := make([]string, 0, len(stories))
	for i := range stories {
		byKey[stories[i].Key] = &stories[i]
		keys = append(keys, stories[i].Key)
	}

	// Build adjacency list and in-degree map from explicit DependsOn edges
	adj := make(map[string][]string)    // dep → []dependents
	inDegree := make(map[string]int)    // story key → in-degree
	edgeSet := make(map[[2]string]bool) // track existing edges to avoid duplicates

	for _, key := range keys {
		inDegree[key] = 0
	}

	for _, story := range stories {
		for _, dep := range story.DependsOn {
			if _, ok := byKey[dep]; !ok {
				continue // skip unknown keys (AC6)
			}
			edge := [2]string{dep, story.Key}
			if edgeSet[edge] {
				continue
			}
			edgeSet[edge] = true
			adj[dep] = append(adj[dep], story.Key)
			inDegree[story.Key]++
		}
	}

	// Add implicit file-conflict edges
	addFileConflictEdges(stories, byKey, adj, inDegree, edgeSet)

	// Kahn's algorithm: process zero-in-degree nodes layer by layer
	var groups [][]model.Story
	processed := 0

	for processed < len(stories) {
		// Collect all nodes with zero in-degree
		var zeroKeys []string
		for _, key := range keys {
			if inDegree[key] == 0 {
				zeroKeys = append(zeroKeys, key)
			}
		}

		if len(zeroKeys) == 0 {
			return model.DAGResult{}, errors.NewInvalidState(
				"DAG_CYCLE_DETECTED",
				"cycle detected in story dependencies",
			)
		}

		// Sort for deterministic output within a layer
		sort.Strings(zeroKeys)

		group := make([]model.Story, 0, len(zeroKeys))
		for _, key := range zeroKeys {
			group = append(group, *byKey[key])
			// Remove this node: set in-degree to -1 so it's skipped in future iterations
			inDegree[key] = -1
			for _, dependent := range adj[key] {
				inDegree[dependent]--
			}
		}

		groups = append(groups, group)
		processed += len(group)
	}

	return model.DAGResult{Groups: groups}, nil
}

// addFileConflictEdges adds implicit directed edges between stories that share target files.
// For each file with multiple stories, edges go from lexicographically smaller keys to larger keys.
func addFileConflictEdges(
	stories []model.Story,
	byKey map[string]*model.Story,
	adj map[string][]string,
	inDegree map[string]int,
	edgeSet map[[2]string]bool,
) {
	// Build file → story keys index
	fileIndex := make(map[string][]string)
	for _, story := range stories {
		for _, f := range story.TargetFiles {
			fileIndex[f] = append(fileIndex[f], story.Key)
		}
	}

	// For each file with >1 story, create chain edges in sorted key order
	for _, storyKeys := range fileIndex {
		if len(storyKeys) < 2 {
			continue
		}
		sort.Strings(storyKeys)
		// Deduplicate keys for the same file
		unique := dedup(storyKeys)
		for i := 0; i < len(unique)-1; i++ {
			src := unique[i]
			dst := unique[i+1]
			// Only check stories that are in our input
			if _, ok := byKey[src]; !ok {
				continue
			}
			if _, ok := byKey[dst]; !ok {
				continue
			}
			edge := [2]string{src, dst}
			if edgeSet[edge] {
				continue // skip if explicit edge already exists
			}
			edgeSet[edge] = true
			adj[src] = append(adj[src], dst)
			inDegree[dst]++
		}
	}
}

// dedup returns a new slice with duplicates removed, preserving order.
func dedup(sorted []string) []string {
	if len(sorted) == 0 {
		return sorted
	}
	result := []string{sorted[0]}
	for i := 1; i < len(sorted); i++ {
		if sorted[i] != sorted[i-1] {
			result = append(result, sorted[i])
		}
	}
	return result
}
