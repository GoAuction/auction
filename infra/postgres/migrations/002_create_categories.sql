-- Categories table for hierarchical item categorization
CREATE TABLE IF NOT EXISTS categories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Category name
    name VARCHAR(100) NOT NULL,

    -- Optional description
    description TEXT,

    -- Parent category for hierarchical structure (self-referencing)
    parent_id UUID,

    -- Category status (active, inactive, archived)
    status VARCHAR(32) NOT NULL DEFAULT 'active',

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Foreign key to parent category
    CONSTRAINT fk_categories_parent FOREIGN KEY (parent_id)
        REFERENCES categories(id) ON DELETE CASCADE,

    -- Prevent category from being its own parent
    CONSTRAINT categories_not_self_parent CHECK (id != parent_id)
);

-- Trigger for updated_at
CREATE TRIGGER trg_categories_updated_at
BEFORE UPDATE ON categories
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

-- Indexes
CREATE INDEX idx_categories_parent_id ON categories(parent_id);
CREATE INDEX idx_categories_status ON categories(status);
CREATE INDEX idx_categories_name ON categories(name);
