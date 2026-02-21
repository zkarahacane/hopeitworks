-- Add diff_url column to hitl_requests table
-- This stores the PR URL for viewing diffs (e.g., GitHub PR URL)
ALTER TABLE hitl_requests ADD COLUMN diff_url TEXT;
