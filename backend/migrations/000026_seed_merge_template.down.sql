-- Remove the merge template seeded by migration 000024.
DELETE FROM prompt_templates WHERE name = 'merge';
