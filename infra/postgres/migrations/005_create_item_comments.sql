-- Item comments table for threaded discussions
CREATE TABLE IF NOT EXISTS item_comments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Associated item
    item_id UUID NOT NULL,

    -- Comment content
    content TEXT NOT NULL,

    -- User who created the comment (from identity service)
    user_id UUID NOT NULL,

    -- Parent comment for threaded discussions (NULL for top-level comments)
    parent_id UUID,

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Foreign keys
    CONSTRAINT fk_item_comments_item FOREIGN KEY (item_id)
        REFERENCES items(id) ON DELETE CASCADE,
    CONSTRAINT fk_item_comments_parent FOREIGN KEY (parent_id)
        REFERENCES item_comments(id) ON DELETE CASCADE,

    -- Prevent comment from being its own parent
    CONSTRAINT item_comments_not_self_parent CHECK (id != parent_id)
);

-- Trigger for updated_at
CREATE TRIGGER trg_item_comments_updated_at
BEFORE UPDATE ON item_comments
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

-- Indexes
CREATE INDEX idx_item_comments_item_id ON item_comments(item_id);
CREATE INDEX idx_item_comments_user_id ON item_comments(user_id);
CREATE INDEX idx_item_comments_parent_id ON item_comments(parent_id);
-- Composite index for retrieving top-level comments ordered by date
CREATE INDEX idx_item_comments_item_parent_created ON item_comments(item_id, parent_id, created_at DESC);
-- Full-text search on comment content
CREATE INDEX idx_item_comments_content_fts ON item_comments USING gin(to_tsvector('english', content));
