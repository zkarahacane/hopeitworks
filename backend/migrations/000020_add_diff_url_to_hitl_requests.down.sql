-- Remove diff_url column from hitl_requests table
ALTER TABLE hitl_requests DROP COLUMN IF EXISTS diff_url;
