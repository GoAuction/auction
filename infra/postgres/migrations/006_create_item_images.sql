-- Item images table for storing multiple images per item
CREATE TABLE IF NOT EXISTS item_images (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Associated item
    item_id UUID NOT NULL,

    -- Image URL (could be CDN path, S3 URL, etc.)
    url VARCHAR(500) NOT NULL,

    -- Display order for multiple images
    display_order INT NOT NULL DEFAULT 0,

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Foreign key to items
    CONSTRAINT fk_item_images_item FOREIGN KEY (item_id)
        REFERENCES items(id) ON DELETE CASCADE,

    -- Ensure display order is non-negative
    CONSTRAINT item_images_display_order_positive CHECK (display_order >= 0)
);

-- Trigger for updated_at
CREATE TRIGGER trg_item_images_updated_at
BEFORE UPDATE ON item_images
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

-- Indexes
CREATE INDEX idx_item_images_item_id ON item_images(item_id);
-- Composite index for retrieving images in display order
CREATE INDEX idx_item_images_item_order ON item_images(item_id, display_order ASC);
