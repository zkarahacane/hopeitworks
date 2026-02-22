-- Seed the merge prompt template for all existing projects where it does not already exist.
-- This template was missing from the original 000012 seed migration.

INSERT INTO prompt_templates (project_id, name, template_content, type)
SELECT p.id, 'merge', 'Merge changes for {{story_key}}: {{story_title}}

## Story Context
**Objective:** {{story_objective}}
**Branch:** {{branch_name}}

## Merge Steps
1. Check that CI is green on the feature branch: gh pr checks or gh run list --branch {{branch_name}}
2. Rebase the feature branch on develop: git fetch origin develop && git rebase origin/develop
3. Push the rebased branch: git push --force-with-lease
4. Create a PR following conventional commit format: gh pr create --title "feat(scope): summary" --body "..."
5. Squash merge after CI passes: gh pr merge --squash --auto
6. Verify that CI passes on develop after merge', 'merge'
FROM projects p
WHERE NOT EXISTS (
    SELECT 1 FROM prompt_templates pt
    WHERE pt.project_id = p.id AND pt.name = 'merge'
);
