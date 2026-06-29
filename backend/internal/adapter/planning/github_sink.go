package planning

import (
	"context"
	"fmt"

	"github.com/shurcooL/githubv4"

	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

// StatusOptions resolves the single-select status field (its id + every option) of a
// board. It powers BOTH the connector status-options endpoint and the write-back
// (which needs the field id and the target option name for the audit). Result is
// memoized best-effort per (projectURL|field).
func (a *GitHubProjectsAdapter) StatusOptions(ctx context.Context, projectURL, statusField string) (port.PlanningStatusOptions, error) {
	if statusField == "" {
		statusField = "Status"
	}
	cacheKey := projectURL + "\x1f" + statusField
	a.mu.Lock()
	if cached, ok := a.fieldCache[cacheKey]; ok {
		a.mu.Unlock()
		return cached, nil
	}
	a.mu.Unlock()

	login, number, ownerIsOrg, err := parseProjectURL(projectURL)
	if err != nil {
		return port.PlanningStatusOptions{}, err
	}
	nodeID, err := a.resolveProjectID(ctx, ownerIsOrg, login, number, projectURL)
	if err != nil {
		return port.PlanningStatusOptions{}, err
	}

	var q struct {
		Node struct {
			ProjectV2 struct {
				Field struct {
					SingleSelect struct {
						ID      githubv4.String
						Name    githubv4.String
						Options []struct {
							ID   githubv4.String
							Name githubv4.String
						}
					} `graphql:"... on ProjectV2SingleSelectField"`
				} `graphql:"field(name: $fieldName)"`
			} `graphql:"... on ProjectV2"`
		} `graphql:"node(id: $projectId)"`
	}
	vars := map[string]interface{}{
		"projectId": githubv4.ID(nodeID),
		"fieldName": githubv4.String(statusField),
	}
	if err := a.client.Query(ctx, &q, vars); err != nil {
		return port.PlanningStatusOptions{}, fmt.Errorf("resolve status field %q: %w", statusField, err)
	}

	ss := q.Node.ProjectV2.Field.SingleSelect
	if string(ss.ID) == "" {
		return port.PlanningStatusOptions{}, fmt.Errorf(
			"status field %q not found on this board, or it is not a single-select field", statusField)
	}

	out := port.PlanningStatusOptions{
		FieldID:   string(ss.ID),
		FieldName: string(ss.Name),
		Options:   make([]port.PlanningStatusOption, 0, len(ss.Options)),
	}
	if out.FieldName == "" {
		out.FieldName = statusField
	}
	for _, o := range ss.Options {
		out.Options = append(out.Options, port.PlanningStatusOption{ID: string(o.ID), Name: string(o.Name)})
	}

	a.mu.Lock()
	a.fieldCache[cacheKey] = out
	a.mu.Unlock()
	return out, nil
}

// WriteBack pushes one status (and optional comment) to a single project item. It
// resolves the project node id + the status field id (from the request or via
// StatusOptions), sets the single-select value, then posts the comment on the item's
// content node. A DraftIssue (no content node) silently skips the comment.
func (a *GitHubProjectsAdapter) WriteBack(ctx context.Context, req port.WriteBackRequest) (port.WriteBackResult, error) {
	if req.ItemID == "" {
		return port.WriteBackResult{}, fmt.Errorf("write-back requires an item id")
	}
	if req.OptionID == "" {
		return port.WriteBackResult{}, fmt.Errorf("write-back requires a target option id")
	}

	login, number, ownerIsOrg, err := parseProjectURL(req.ProjectURL)
	if err != nil {
		return port.WriteBackResult{}, err
	}
	projectNodeID, err := a.resolveProjectID(ctx, ownerIsOrg, login, number, req.ProjectURL)
	if err != nil {
		return port.WriteBackResult{}, err
	}

	// Resolve the field id (+ option name for the audit). The cache makes this cheap.
	opts, err := a.StatusOptions(ctx, req.ProjectURL, req.StatusFieldName)
	if err != nil {
		return port.WriteBackResult{}, err
	}
	fieldID := req.StatusFieldID
	if fieldID == "" {
		fieldID = opts.FieldID
	}
	if fieldID == "" {
		return port.WriteBackResult{}, fmt.Errorf("could not resolve the status field id for write-back")
	}
	remoteStatus := req.OptionID
	for _, o := range opts.Options {
		if o.ID == req.OptionID {
			remoteStatus = o.Name
			break
		}
	}

	var m struct {
		UpdateProjectV2ItemFieldValue struct {
			ProjectV2Item struct {
				ID githubv4.String
			}
		} `graphql:"updateProjectV2ItemFieldValue(input: $input)"`
	}
	input := githubv4.UpdateProjectV2ItemFieldValueInput{
		ProjectID: githubv4.ID(projectNodeID),
		ItemID:    githubv4.ID(req.ItemID),
		FieldID:   githubv4.ID(fieldID),
		Value: githubv4.ProjectV2FieldValue{
			SingleSelectOptionID: githubv4.NewString(githubv4.String(req.OptionID)),
		},
	}
	if err := a.client.Mutate(ctx, &m, input, nil); err != nil {
		return port.WriteBackResult{}, fmt.Errorf("set status field value: %w", err)
	}

	res := port.WriteBackResult{RemoteStatus: remoteStatus}

	// Optional comment on the item's content node. A DraftIssue has no content node
	// (ContentNodeID == "") => skip cleanly, never error.
	if req.Comment != "" && req.ContentNodeID != "" {
		var cm struct {
			AddComment struct {
				ClientMutationID githubv4.String
			} `graphql:"addComment(input: $input)"`
		}
		ci := githubv4.AddCommentInput{
			SubjectID: githubv4.ID(req.ContentNodeID),
			Body:      githubv4.String(req.Comment),
		}
		if err := a.client.Mutate(ctx, &cm, ci, nil); err != nil {
			// The status push already succeeded; a comment failure must not fail the
			// whole write-back. Surface it in the log, keep the synced status.
			a.logger.Warn("write-back status applied but comment failed",
				"item_id", req.ItemID, "error", err)
		} else {
			res.CommentPosted = true
		}
	}

	return res, nil
}
