-- Remove default templates seeded by migration 000012.
-- Only deletes templates whose content matches the seeded defaults,
-- to avoid removing user-created templates with the same names.
DELETE FROM prompt_templates
WHERE (name = 'implement' AND template_content LIKE 'Implement story {{story_key}}%')
   OR (name = 'implement-retry' AND template_content LIKE 'Retry implementation for {{story_key}}%')
   OR (name = 'review' AND template_content LIKE 'Review changes for {{story_key}}%')
   OR (name = 'merge-conflict' AND template_content LIKE 'Resolve merge conflict for {{story_key}}%');
