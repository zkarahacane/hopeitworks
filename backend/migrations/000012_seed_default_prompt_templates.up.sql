-- Seed default prompt templates for all existing projects.
-- Each project gets: implement, implement-retry, review, merge-conflict.

-- implement template
INSERT INTO prompt_templates (project_id, name, template_content, type)
SELECT p.id, 'implement', 'Implement story {{story_key}}: {{story_title}}

## Objective
{{story_objective}}

## Target Files
{{#each target_files}}
- {{this}}
{{/each}}

## Acceptance Criteria
{{acceptance_criteria}}

## Branch
{{branch_name}}', 'implement'
FROM projects p
WHERE NOT EXISTS (
    SELECT 1 FROM prompt_templates pt
    WHERE pt.project_id = p.id AND pt.name = 'implement'
);

-- implement-retry template
INSERT INTO prompt_templates (project_id, name, template_content, type)
SELECT p.id, 'implement-retry', 'Retry implementation for {{story_key}}: {{story_title}}

## Previous Error
{{error_context}}

## Existing Changes
{{diff_content}}

## Objective
{{story_objective}}

Fix the issues described above while preserving the existing changes.', 'retry'
FROM projects p
WHERE NOT EXISTS (
    SELECT 1 FROM prompt_templates pt
    WHERE pt.project_id = p.id AND pt.name = 'implement-retry'
);

-- review template
INSERT INTO prompt_templates (project_id, name, template_content, type)
SELECT p.id, 'review', 'Review changes for {{story_key}}: {{story_title}}

## Story Context
**Objective:** {{story_objective}}

**Acceptance Criteria:**
{{acceptance_criteria}}

## Changes to Review
{{diff_content}}

## Review Instructions
- Verify all acceptance criteria are met
- Check code quality and adherence to project conventions
- Flag any issues or suggest improvements', 'review'
FROM projects p
WHERE NOT EXISTS (
    SELECT 1 FROM prompt_templates pt
    WHERE pt.project_id = p.id AND pt.name = 'review'
);

-- merge-conflict template
INSERT INTO prompt_templates (project_id, name, template_content, type)
SELECT p.id, 'merge-conflict', 'Resolve merge conflict for {{story_key}}: {{story_title}}

## Story Context
**Objective:** {{story_objective}}

## Conflict Details
{{error_context}}

## Current Changes
{{diff_content}}

## Resolution Instructions
- Review the conflict markers in the diff
- Resolve conflicts while preserving the story objective
- Ensure all acceptance criteria remain satisfied after resolution', 'merge'
FROM projects p
WHERE NOT EXISTS (
    SELECT 1 FROM prompt_templates pt
    WHERE pt.project_id = p.id AND pt.name = 'merge-conflict'
);
