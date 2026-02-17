-- Remove default templates seeded by migration 000012.
DELETE FROM prompt_templates
WHERE name IN ('implement', 'implement-retry', 'review', 'merge-conflict');
