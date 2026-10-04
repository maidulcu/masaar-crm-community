-- +goose Up
-- WhatsApp fixes:
--  * media columns so inbound media can be recorded (and downloaded later) instead of dropped
--  * index for delivery-receipt lookups by Meta message id
--  * normalise contacts.phone_wa to E.164 ("+971…") and merge contacts that differ only by
--    formatting (WhatsApp reports "971…", agents type "+971…", imports have spaces, …).

ALTER TABLE whatsapp_messages ADD COLUMN IF NOT EXISTS wa_media_id TEXT;
ALTER TABLE whatsapp_messages ADD COLUMN IF NOT EXISTS media_mime  TEXT;

-- Signed media URLs are routinely longer than 500 characters.
ALTER TABLE whatsapp_outbound ALTER COLUMN media_url TYPE TEXT;

CREATE INDEX IF NOT EXISTS idx_wa_outbound_wa_message_id
    ON whatsapp_outbound(wa_message_id) WHERE wa_message_id IS NOT NULL;

-- +goose StatementBegin
DO $$
DECLARE
    grp  RECORD;
    dup  RECORD;
    keep UUID;
    t    RECORD;
    kt   UUID;
BEGIN
    -- Canonical E.164 form of every contact number we can interpret (NULL = leave untouched,
    -- e.g. national-format numbers whose country is unknown).
    CREATE TEMP TABLE _contact_norm ON COMMIT DROP AS
    SELECT id, company_id, created_at,
           CASE
               WHEN d ~ '^00[1-9][0-9]{6,14}$' THEN '+' || substr(d, 3)
               WHEN d ~ '^[1-9][0-9]{6,14}$'   THEN '+' || d
           END AS n
    FROM (SELECT id, company_id, created_at, regexp_replace(phone_wa, '[^0-9]', '', 'g') AS d FROM contacts) s;

    -- Merge contacts that normalise to the same number within a company. The oldest survives.
    FOR grp IN
        SELECT company_id, n FROM _contact_norm WHERE n IS NOT NULL
        GROUP BY company_id, n HAVING count(*) > 1
    LOOP
        SELECT id INTO keep FROM _contact_norm
         WHERE company_id = grp.company_id AND n = grp.n
         ORDER BY created_at, id LIMIT 1;

        FOR dup IN
            SELECT id FROM _contact_norm WHERE company_id = grp.company_id AND n = grp.n AND id <> keep
        LOOP
            UPDATE leads                 SET contact_id = keep WHERE contact_id = dup.id;
            UPDATE offers                SET contact_id = keep WHERE contact_id = dup.id;
            UPDATE viewings              SET contact_id = keep WHERE contact_id = dup.id;
            UPDATE communication_history SET contact_id = keep WHERE contact_id = dup.id;

            -- Threads are unique per (contact, WhatsApp account): fold into the survivor's
            -- thread when it has one for the same account, otherwise just re-parent.
            FOR t IN SELECT id, wa_account_id FROM whatsapp_threads WHERE contact_id = dup.id LOOP
                SELECT id INTO kt FROM whatsapp_threads WHERE contact_id = keep AND wa_account_id = t.wa_account_id;
                IF kt IS NULL THEN
                    UPDATE whatsapp_threads SET contact_id = keep WHERE id = t.id;
                ELSE
                    UPDATE whatsapp_messages  SET thread_id = kt WHERE thread_id = t.id;
                    UPDATE whatsapp_outbound  SET thread_id = kt WHERE thread_id = t.id;
                    UPDATE whatsapp_threads k SET
                        message_count   = k.message_count + d.message_count,
                        last_message_at = GREATEST(k.last_message_at, d.last_message_at),
                        ai_summary      = COALESCE(NULLIF(k.ai_summary, ''), d.ai_summary)
                    FROM whatsapp_threads d WHERE k.id = kt AND d.id = t.id;
                    DELETE FROM whatsapp_threads WHERE id = t.id;
                END IF;
            END LOOP;

            -- Keep whatever the duplicate knew that the survivor does not.
            UPDATE contacts k SET
                email       = COALESCE(NULLIF(k.email, ''), d.email),
                assigned_to = COALESCE(k.assigned_to, d.assigned_to),
                lead_score  = GREATEST(k.lead_score, d.lead_score),
                full_name   = CASE
                    WHEN btrim(k.full_name) = '' OR regexp_replace(k.full_name, '[^0-9]', '', 'g') = regexp_replace(k.phone_wa, '[^0-9]', '', 'g')
                    THEN d.full_name ELSE k.full_name END
            FROM contacts d WHERE k.id = keep AND d.id = dup.id;

            DELETE FROM contacts WHERE id = dup.id;
        END LOOP;
    END LOOP;

    UPDATE contacts c SET phone_wa = n.n
    FROM _contact_norm n
    WHERE n.id = c.id AND n.n IS NOT NULL AND c.phone_wa <> n.n;
END
$$;
-- +goose StatementEnd

-- +goose Down
-- Merged contacts and normalised numbers cannot be split apart again; only the additive schema
-- changes are reverted.
DROP INDEX IF EXISTS idx_wa_outbound_wa_message_id;
ALTER TABLE whatsapp_messages DROP COLUMN IF EXISTS media_mime;
ALTER TABLE whatsapp_messages DROP COLUMN IF EXISTS wa_media_id;
