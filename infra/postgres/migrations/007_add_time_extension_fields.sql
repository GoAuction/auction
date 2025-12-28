-- Migration: Add auction time extension configuration fields
ALTER TABLE items
    ADD COLUMN extension_threshold_minutes INTEGER,
    ADD COLUMN extension_duration_minutes INTEGER,
    ADD COLUMN version INTEGER NOT NULL DEFAULT 1;

-- Constraints for positive values
ALTER TABLE items
    ADD CONSTRAINT items_extension_threshold_positive
        CHECK (extension_threshold_minutes IS NULL OR extension_threshold_minutes > 0),
    ADD CONSTRAINT items_extension_duration_positive
        CHECK (extension_duration_minutes IS NULL OR extension_duration_minutes > 0);

-- Comments for documentation
COMMENT ON COLUMN items.extension_threshold_minutes IS
    'Minutes before auction end when bids trigger time extension. NULL uses default (5 minutes).';
COMMENT ON COLUMN items.extension_duration_minutes IS
    'Minutes to extend auction when bid arrives in threshold window. NULL uses default (5 minutes).';
COMMENT ON COLUMN items.version IS
    'Optimistic locking version to prevent race conditions in concurrent updates.';

-- Index for optimistic locking performance
CREATE INDEX idx_items_version ON items(id, version);
