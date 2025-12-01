-- Item-Category junction table (many-to-many relationship)
CREATE TABLE IF NOT EXISTS item_categories (
    item_id UUID NOT NULL,
    category_id UUID NOT NULL,

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Composite primary key
    PRIMARY KEY (item_id, category_id),

    -- Foreign keys
    CONSTRAINT fk_item_categories_item FOREIGN KEY (item_id)
        REFERENCES items(id) ON DELETE CASCADE,
    CONSTRAINT fk_item_categories_category FOREIGN KEY (category_id)
        REFERENCES categories(id) ON DELETE CASCADE
);

-- Trigger for updated_at
CREATE TRIGGER trg_item_categories_updated_at
BEFORE UPDATE ON item_categories
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

-- Indexes for efficient lookups in both directions
CREATE INDEX idx_item_categories_item_id ON item_categories(item_id);
CREATE INDEX idx_item_categories_category_id ON item_categories(category_id);
