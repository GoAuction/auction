CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE IF NOT EXISTS items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- İlan temel bilgileri
    name        VARCHAR(255) NOT NULL,
    description TEXT,

    -- Satıcı: Identity servisindeki user id
    seller_id UUID NOT NULL,

    -- Alıcı: Identity servisindeki user id
    buyer_id UUID,

    -- Para birimi
    currency_code CHAR(3) NOT NULL DEFAULT 'TRY'
        CHECK (char_length(currency_code) = 3),

    -- Fiyat alanları
    start_price   NUMERIC(10,2) NOT NULL CHECK (start_price >= 0),
    current_price NUMERIC(10,2) NOT NULL CHECK (current_price >= 0) DEFAULT 0,
    bid_increment NUMERIC(10,2) NULL CHECK (bid_increment IS NULL OR bid_increment > 0),

    reserve_price NUMERIC(10,2) CHECK (reserve_price IS NULL OR reserve_price >= 0),
    buyout_price  NUMERIC(10,2) CHECK (buyout_price  IS NULL OR buyout_price  >= 0),
    end_price     NUMERIC(10,2) CHECK (end_price     IS NULL OR end_price     >= 0),

    -- Tarihler
    start_date TIMESTAMPTZ NOT NULL,
    end_date   TIMESTAMPTZ NOT NULL,

    -- Auction durumu
    status VARCHAR(32) NOT NULL DEFAULT 'draft',

    -- Zaman damgaları
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Mantıksal bütünlük
    CONSTRAINT items_dates_order CHECK (end_date > start_date),
    CONSTRAINT items_price_order CHECK (end_price IS NULL OR end_price >= start_price)
);

CREATE TRIGGER trg_items_updated_at
BEFORE UPDATE ON items
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

-- Indexler
CREATE INDEX idx_items_end_date    ON items(end_date);
CREATE INDEX idx_items_status      ON items(status);
CREATE INDEX idx_items_seller_id   ON items(seller_id);
CREATE INDEX idx_items_seller_stat ON items(seller_id, status, end_date DESC);
