-- +goose Up
CREATE TABLE email_history (
  id BIGSERIAL PRIMARY KEY,
  from_email VARCHAR(255) NOT NULL,
  to_email VARCHAR(255) NOT NULL,
  subject VARCHAR(500) NOT NULL,
  body TEXT,
  html_body TEXT,
  status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, sent, failed, bounced
  error_message TEXT,
  related_to VARCHAR(50), -- invoice, proposal, followup
  related_id BIGINT,
  sent_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  created_by UUID REFERENCES users(id) ON DELETE SET NULL,
  metadata JSONB
);

CREATE INDEX idx_email_status ON email_history(status);
CREATE INDEX idx_email_related ON email_history(related_to, related_id);
CREATE INDEX idx_email_to ON email_history(to_email);
CREATE INDEX idx_email_created_at ON email_history(created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS email_history;
