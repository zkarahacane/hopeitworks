-- INC 4a: halt-gate (probe_halt) support on hitl_requests.
-- gate_type is already a free VARCHAR(50) with no CHECK, so 'probe_halt' needs
-- no schema change for the type itself. We add:
--   halt_reason       structured reason a probe halted the run (kind + value +
--                     threshold), so the resolution UI can suggest a remedy.
--   resolution_action the enriched halt-gate action taken by the human
--                     (resume | override | send_back | skip | abort), recorded
--                     alongside resolved_by for audit. Approve/reject leave it null.
ALTER TABLE hitl_requests ADD COLUMN halt_reason JSONB;
ALTER TABLE hitl_requests ADD COLUMN resolution_action VARCHAR(50);

-- Probe halts are resolved out of the project pending queue by gate_type; an
-- index on gate_type keeps the batch-triage listing cheap.
CREATE INDEX idx_hitl_requests_gate_type ON hitl_requests(gate_type);
