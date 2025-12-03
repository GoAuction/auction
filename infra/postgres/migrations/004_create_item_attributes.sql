-- Item attributes table for flexible key-value metadata
CREATE TABLE IF NOT EXISTS item_attributes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    item_id UUID NOT NULL,

    -- Attribute key (e.g., "brand", "color", "condition")
    key VARCHAR(100) NOT NULL,

    -- Attribute value
    value TEXT NOT NULL,

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Foreign key to items
    CONSTRAINT fk_item_attributes_item FOREIGN KEY (item_id)
        REFERENCES items(id) ON DELETE CASCADE
);

-- Trigger for updated_at
CREATE TRIGGER trg_item_attributes_updated_at
BEFORE UPDATE ON item_attributes
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

-- Indexes
CREATE INDEX idx_item_attributes_item_id ON item_attributes(item_id);
CREATE INDEX idx_item_attributes_key ON item_attributes(key);
-- GIN index for efficient search on key-value pairs
CREATE INDEX idx_item_attributes_key_value ON item_attributes USING gin(to_tsvector('english', key || ' ' || value));
