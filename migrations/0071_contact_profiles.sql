-- +goose Up
-- Richer contacts: B2B organisations, profile fields, and working custom fields.

CREATE TABLE organisations (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id       UUID        NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    name             TEXT        NOT NULL,
    trade_license_no TEXT        NOT NULL DEFAULT '',
    trn              TEXT        NOT NULL DEFAULT '',
    address          TEXT        NOT NULL DEFAULT '',
    phone            TEXT        NOT NULL DEFAULT '',
    email            TEXT        NOT NULL DEFAULT '',
    website          TEXT        NOT NULL DEFAULT '',
    notes            TEXT        NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_organisations_company_name ON organisations(company_id, lower(name));

ALTER TABLE contacts
    ADD COLUMN IF NOT EXISTS secondary_phones       TEXT[]      NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS nationality            CHAR(2)     NOT NULL DEFAULT '', -- ISO 3166-1 alpha-2
    ADD COLUMN IF NOT EXISTS source                 TEXT        NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS preferred_contact_time TEXT        NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS address                TEXT        NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS notes                  TEXT        NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS job_title              TEXT        NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS organisation_id        UUID        REFERENCES organisations(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_contacts_organisation ON contacts(organisation_id) WHERE organisation_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_contacts_company_email ON contacts(company_id, lower(email)) WHERE email IS NOT NULL AND email <> '';

-- Custom fields (tables from 0030, never wired up): defaults, company scoping on values,
-- one value per field and record, and allowed types.
ALTER TABLE custom_field_definitions
    ALTER COLUMN id SET DEFAULT gen_random_uuid(),
    ADD COLUMN IF NOT EXISTS options TEXT[] NOT NULL DEFAULT '{}';
UPDATE custom_field_definitions SET field_type = 'text' WHERE field_type IS NULL OR field_type NOT IN ('text','number','date','select','boolean');
ALTER TABLE custom_field_definitions
    ALTER COLUMN field_type SET DEFAULT 'text',
    ALTER COLUMN field_type SET NOT NULL,
    ADD CONSTRAINT custom_field_definitions_type_check CHECK (field_type IN ('text','number','date','select','boolean')),
    ADD CONSTRAINT custom_field_definitions_entity_check CHECK (entity_type IN ('contact','lead','deal','listing'));

ALTER TABLE custom_field_values
    ALTER COLUMN id SET DEFAULT gen_random_uuid(),
    ADD COLUMN IF NOT EXISTS company_id UUID REFERENCES companies(id) ON DELETE CASCADE;
UPDATE custom_field_values v SET company_id = d.company_id FROM custom_field_definitions d WHERE d.id = v.field_id;
DELETE FROM custom_field_values WHERE company_id IS NULL;
ALTER TABLE custom_field_values
    ALTER COLUMN company_id SET NOT NULL,
    ADD CONSTRAINT custom_field_values_field_fkey FOREIGN KEY (field_id) REFERENCES custom_field_definitions(id) ON DELETE CASCADE;
CREATE UNIQUE INDEX IF NOT EXISTS idx_custom_field_values_unique ON custom_field_values(field_id, entity_id);

-- +goose Down
DROP INDEX IF EXISTS idx_custom_field_values_unique;
ALTER TABLE custom_field_values DROP CONSTRAINT IF EXISTS custom_field_values_field_fkey, DROP COLUMN IF EXISTS company_id;
ALTER TABLE custom_field_definitions
    DROP CONSTRAINT IF EXISTS custom_field_definitions_entity_check,
    DROP CONSTRAINT IF EXISTS custom_field_definitions_type_check,
    ALTER COLUMN field_type DROP NOT NULL,
    DROP COLUMN IF EXISTS options;
DROP INDEX IF EXISTS idx_contacts_company_email;
DROP INDEX IF EXISTS idx_contacts_organisation;
ALTER TABLE contacts
    DROP COLUMN IF EXISTS organisation_id,
    DROP COLUMN IF EXISTS job_title,
    DROP COLUMN IF EXISTS notes,
    DROP COLUMN IF EXISTS address,
    DROP COLUMN IF EXISTS preferred_contact_time,
    DROP COLUMN IF EXISTS source,
    DROP COLUMN IF EXISTS nationality,
    DROP COLUMN IF EXISTS secondary_phones;
DROP TABLE IF EXISTS organisations;
