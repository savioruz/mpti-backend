BEGIN;

CREATE TABLE IF NOT EXISTS fields (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    location_id UUID REFERENCES locations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(100) NOT NULL,
    price NUMERIC(12, 2) NOT NULL,
    description TEXT DEFAULT NULL,
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP DEFAULT now(),
    deleted_at TIMESTAMP DEFAULT NULL
);

CREATE INDEX idx_fields_location_id ON fields(location_id);
CREATE UNIQUE INDEX idx_fields_name ON fields(name, location_id);
CREATE INDEX idx_fields_deleted_at ON fields USING btree (deleted_at ASC NULLS LAST);

COMMIT;