package planning

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

const sinkOrgURL = "https://github.com/orgs/acme/projects/7"

// sinkHandler is a gqlHandler that serves the resolution + field + mutation phases of
// a write-back, recording the GraphQL operations seen for assertions.
func sinkHandler(seen *[]string) gqlHandler {
	return func(query string, _ map[string]any) string {
		*seen = append(*seen, query)
		switch {
		case strings.Contains(query, "organization(login:"):
			return `{"data":{"organization":{"projectV2":{"id":"PROJ_NODE"}}}}`
		case strings.Contains(query, "ProjectV2SingleSelectField"):
			return `{"data":{"node":{"field":{"id":"FIELD_1","name":"Status",` +
				`"options":[{"id":"o1","name":"Todo"},{"id":"o2","name":"Done"}]}}}}`
		case strings.Contains(query, "updateProjectV2ItemFieldValue"):
			return `{"data":{"updateProjectV2ItemFieldValue":{"projectV2Item":{"id":"ITEM_ID"}}}}`
		case strings.Contains(query, "addComment"):
			return `{"data":{"addComment":{"clientMutationId":"x"}}}`
		default:
			return `{"data":{}}`
		}
	}
}

func anySeen(seen []string, substr string) bool {
	for _, q := range seen {
		if strings.Contains(q, substr) {
			return true
		}
	}
	return false
}

func TestSink_StatusOptions_ResolvesField(t *testing.T) {
	var seen []string
	a := newTestAdapter(t, sinkHandler(&seen))

	opts, err := a.StatusOptions(context.Background(), sinkOrgURL, "Status")
	require.NoError(t, err)
	assert.Equal(t, "FIELD_1", opts.FieldID)
	assert.Equal(t, "Status", opts.FieldName)
	require.Len(t, opts.Options, 2)
	assert.Equal(t, "o2", opts.Options[1].ID)
	assert.Equal(t, "Done", opts.Options[1].Name)
}

func TestSink_StatusOptions_FieldNotFound(t *testing.T) {
	a := newTestAdapter(t, func(query string, _ map[string]any) string {
		if strings.Contains(query, "organization(login:") {
			return `{"data":{"organization":{"projectV2":{"id":"PROJ_NODE"}}}}`
		}
		return `{"data":{"node":{"field":null}}}` // no single-select field by that name
	})

	_, err := a.StatusOptions(context.Background(), sinkOrgURL, "Nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestSink_WriteBack_SetsStatusAndComments(t *testing.T) {
	var seen []string
	a := newTestAdapter(t, sinkHandler(&seen))

	res, err := a.WriteBack(context.Background(), port.WriteBackRequest{
		ProjectURL:      sinkOrgURL,
		ItemID:          "ITEM_ID",
		ContentNodeID:   "CONTENT_NODE",
		StatusFieldName: "Status",
		OptionID:        "o2",
		Comment:         "hopeitworks: done",
	})
	require.NoError(t, err)
	assert.Equal(t, "Done", res.RemoteStatus) // option name resolved from id o2
	assert.True(t, res.CommentPosted)
	assert.True(t, anySeen(seen, "updateProjectV2ItemFieldValue"), "expected the field mutation")
	assert.True(t, anySeen(seen, "addComment"), "expected the comment mutation")
}

func TestSink_WriteBack_DraftSkipsComment(t *testing.T) {
	var seen []string
	a := newTestAdapter(t, sinkHandler(&seen))

	res, err := a.WriteBack(context.Background(), port.WriteBackRequest{
		ProjectURL:      sinkOrgURL,
		ItemID:          "ITEM_ID",
		ContentNodeID:   "", // draft: no content node => comment skipped cleanly
		StatusFieldName: "Status",
		OptionID:        "o2",
		Comment:         "hopeitworks: done",
	})
	require.NoError(t, err)
	assert.False(t, res.CommentPosted)
	assert.True(t, anySeen(seen, "updateProjectV2ItemFieldValue"), "status must still be set")
	assert.False(t, anySeen(seen, "addComment"), "a draft must not trigger a comment mutation")
}
