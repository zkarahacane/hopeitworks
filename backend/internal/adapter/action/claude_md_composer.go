package action

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CLAUDEMDComposer reads and concatenates CLAUDE.md template files
// based on the story scope.
type CLAUDEMDComposer struct {
	basePath string
}

// NewCLAUDEMDComposer creates a new composer with the given base path.
func NewCLAUDEMDComposer(basePath string) *CLAUDEMDComposer {
	return &CLAUDEMDComposer{basePath: basePath}
}

// Compose builds the CLAUDE.md content for the given scope.
// Composition rule: base.md + (backend.md | frontend.md | nothing) + project.md
func (c *CLAUDEMDComposer) Compose(scope string) (string, error) {
	files := []string{"base.md"}

	switch strings.ToLower(scope) {
	case "backend":
		files = append(files, "backend.md")
	case "frontend":
		files = append(files, "frontend.md")
		// "shared", "", or any other value: no scope-specific file
	}

	files = append(files, "project.md")

	var parts []string
	for _, f := range files {
		path := filepath.Join(c.basePath, f)
		content, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("read CLAUDE.md template %q: %w", path, err)
		}
		parts = append(parts, strings.TrimSpace(string(content)))
	}

	return strings.Join(parts, "\n\n"), nil
}
