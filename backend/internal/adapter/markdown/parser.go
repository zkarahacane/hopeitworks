package markdown

import (
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// FrontmatterFields represents the parsed YAML frontmatter of a story block.
type FrontmatterFields struct {
	Key       string   `yaml:"key"`
	Epic      string   `yaml:"epic"`
	DependsOn []string `yaml:"depends_on"`
	Scope     string   `yaml:"scope"`
	Status    string   `yaml:"status"`
}

// ParsedStory holds the result of parsing one story block from markdown.
// ParseError is non-nil if YAML frontmatter failed to parse.
type ParsedStory struct {
	Key                string
	Title              string
	Epic               string
	DependsOn          []string
	Scope              string
	Status             string
	AcceptanceCriteria string
	ParseError         error
}

const frontmatterDelimiter = "---"

var titleRegex = regexp.MustCompile(`(?m)^# (.+)$`)

// ParseStoryMarkdown splits a markdown document into individual story blocks
// and parses the YAML frontmatter and title from each.
// Blocks are delimited by lines consisting solely of "---".
// Returns one ParsedStory per detected block (with ParseError set for invalid YAML).
func ParseStoryMarkdown(content string) []ParsedStory {
	blocks := splitIntoBlocks(content)
	stories := make([]ParsedStory, 0, len(blocks))

	for _, block := range blocks {
		story := parseBlock(block)
		stories = append(stories, story)
	}

	return stories
}

// splitIntoBlocks splits raw markdown content into individual story blocks.
// Each block is expected to start with "---" (frontmatter open), followed by
// YAML content, then "---" (frontmatter close), then the markdown body.
func splitIntoBlocks(content string) []storyBlock {
	lines := strings.Split(content, "\n")
	var blocks []storyBlock

	i := 0
	for i < len(lines) {
		// Skip empty lines between blocks
		if strings.TrimSpace(lines[i]) != frontmatterDelimiter {
			i++
			continue
		}

		// Found opening "---", start collecting frontmatter
		fmStart := i + 1
		i++

		// Find closing "---"
		fmEnd := -1
		for i < len(lines) {
			if strings.TrimSpace(lines[i]) == frontmatterDelimiter {
				fmEnd = i
				i++
				break
			}
			i++
		}

		if fmEnd == -1 {
			// No closing "---" found, skip this incomplete block
			break
		}

		frontmatter := strings.Join(lines[fmStart:fmEnd], "\n")

		// Collect body until next "---" or end of content
		bodyStart := i
		bodyEnd := len(lines)
		for j := i; j < len(lines); j++ {
			if strings.TrimSpace(lines[j]) == frontmatterDelimiter {
				bodyEnd = j
				break
			}
		}

		body := strings.Join(lines[bodyStart:bodyEnd], "\n")
		i = bodyEnd

		blocks = append(blocks, storyBlock{
			frontmatter: frontmatter,
			body:        body,
		})
	}

	return blocks
}

type storyBlock struct {
	frontmatter string
	body        string
}

// parseBlock parses a single story block into a ParsedStory.
func parseBlock(block storyBlock) ParsedStory {
	var fields FrontmatterFields
	if err := yaml.Unmarshal([]byte(block.frontmatter), &fields); err != nil {
		return ParsedStory{
			ParseError: err,
		}
	}

	title, acceptanceCriteria := extractTitleAndBody(block.body)

	return ParsedStory{
		Key:                fields.Key,
		Title:              title,
		Epic:               fields.Epic,
		DependsOn:          fields.DependsOn,
		Scope:              fields.Scope,
		Status:             fields.Status,
		AcceptanceCriteria: acceptanceCriteria,
	}
}

// extractTitleAndBody extracts the first H1 heading as the title and the
// remaining content as acceptance criteria.
func extractTitleAndBody(body string) (title, acceptanceCriteria string) {
	loc := titleRegex.FindStringIndex(body)
	if loc == nil {
		return "", strings.TrimSpace(body)
	}

	match := titleRegex.FindStringSubmatch(body)
	title = strings.TrimSpace(match[1])

	// Everything after the title line becomes the acceptance criteria
	remaining := body[loc[1]:]
	acceptanceCriteria = strings.TrimSpace(remaining)

	return title, acceptanceCriteria
}
