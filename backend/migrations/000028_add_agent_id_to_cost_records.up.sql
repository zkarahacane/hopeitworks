ALTER TABLE cost_records ADD COLUMN agent_id UUID REFERENCES agents(id) ON DELETE SET NULL;
CREATE INDEX idx_cost_records_agent_id ON cost_records(agent_id);
