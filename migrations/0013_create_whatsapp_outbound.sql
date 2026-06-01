-- +goose Up
CREATE TABLE whatsapp_outbound (
  id BIGSERIAL PRIMARY KEY,
  thread_id UUID NOT NULL REFERENCES whatsapp_threads(id) ON DELETE CASCADE,
  to_number VARCHAR(20) NOT NULL,
  message_body TEXT NOT NULL,
  media_url VARCHAR(500),
  wa_message_id VARCHAR(100),
  status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, sent, delivered, read, failed
  error_message TEXT,
  scheduled_at TIMESTAMP,
  sent_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  created_by UUID REFERENCES users(id) ON DELETE SET NULL,
  metadata JSONB
);

CREATE INDEX idx_whatsapp_outbound_thread ON whatsapp_outbound(thread_id);
CREATE INDEX idx_whatsapp_outbound_status ON whatsapp_outbound(status);
CREATE INDEX idx_whatsapp_outbound_created_at ON whatsapp_outbound(created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS whatsapp_outbound;
