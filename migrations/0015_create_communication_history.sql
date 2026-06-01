-- +goose Up
CREATE TABLE communication_history (
  id BIGSERIAL PRIMARY KEY,
  lead_id UUID NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
  contact_id UUID NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
  communication_type VARCHAR(20) NOT NULL, -- whatsapp_inbound, whatsapp_outbound, email_sent, email_received, call
  direction VARCHAR(20), -- inbound, outbound
  body TEXT,
  from_identifier VARCHAR(255), -- phone number, email, name
  to_identifier VARCHAR(255),
  external_id VARCHAR(200), -- wa_message_id, email_message_id
  status VARCHAR(20), -- pending, sent, delivered, read, failed, received
  metadata JSONB,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  created_by UUID REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX idx_comm_by_lead ON communication_history(lead_id);
CREATE INDEX idx_comm_by_contact ON communication_history(contact_id);
CREATE INDEX idx_comm_by_type ON communication_history(communication_type);
CREATE INDEX idx_comm_by_created ON communication_history(created_at DESC);
CREATE INDEX idx_comm_by_external_id ON communication_history(external_id);

-- +goose Down
DROP TABLE IF EXISTS communication_history;
