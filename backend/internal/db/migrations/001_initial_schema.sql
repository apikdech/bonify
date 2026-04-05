-- Initial schema migration for Receipt Manager
-- Creates all tables, indexes, triggers, and seeds initial data

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================
-- SCHEMA MIGRATIONS TABLE
-- ============================================

CREATE TABLE IF NOT EXISTS schema_migrations (
    filename TEXT PRIMARY KEY,
    applied_at TIMESTAMPTZ DEFAULT now()
);

-- ============================================
-- TRIGGER FUNCTION
-- ============================================

-- Create trigger function for updating updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ============================================
-- USERS TABLE
-- ============================================

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    telegram_id TEXT UNIQUE,
    discord_id TEXT UNIQUE,
    role TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('member', 'admin')),
    llm_provider TEXT,
    llm_model TEXT,
    home_currency CHAR(3) NOT NULL DEFAULT 'IDR',
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

-- Create trigger for updated_at
CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================
-- SYSTEM SETTINGS TABLE
-- ============================================

CREATE TABLE system_settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT now(),
    updated_by UUID REFERENCES users(id)
);

-- Create trigger for updated_at
CREATE TRIGGER update_system_settings_updated_at
    BEFORE UPDATE ON system_settings
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================
-- RECEIPTS TABLE
-- ============================================

CREATE TABLE receipts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title TEXT,
    source TEXT NOT NULL DEFAULT 'manual' CHECK (source IN ('telegram', 'discord', 'manual')),
    image_url TEXT,
    ocr_confidence NUMERIC(3,2),
    currency CHAR(3) NOT NULL DEFAULT 'IDR',
    payment_method TEXT CHECK (payment_method IN ('cash', 'card', 'qris', 'transfer', 'unknown')),
    subtotal NUMERIC(14,2) NOT NULL DEFAULT 0,
    total NUMERIC(14,2) NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'confirmed' CHECK (status IN ('confirmed', 'pending_review', 'rejected')),
    notes TEXT,
    receipt_date DATE,
    paid_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

-- Create indexes for receipts
CREATE INDEX idx_receipts_user_id ON receipts(user_id);
CREATE INDEX idx_receipts_receipt_date ON receipts(receipt_date DESC);
CREATE INDEX idx_receipts_status ON receipts(status);
CREATE INDEX idx_receipts_currency ON receipts(currency);

-- Create trigger for updated_at
CREATE TRIGGER update_receipts_updated_at
    BEFORE UPDATE ON receipts
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================
-- RECEIPT ITEMS TABLE
-- ============================================

CREATE TABLE receipt_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    receipt_id UUID NOT NULL REFERENCES receipts(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    quantity INT NOT NULL DEFAULT 1,
    unit_price NUMERIC(14,2) NOT NULL,
    discount NUMERIC(14,2) NOT NULL DEFAULT 0,
    subtotal NUMERIC(14,2) NOT NULL
);

-- Create index for receipt_items
CREATE INDEX idx_receipt_items_receipt_id ON receipt_items(receipt_id);

-- ============================================
-- RECEIPT FEES TABLE
-- ============================================

CREATE TABLE receipt_fees (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    receipt_id UUID NOT NULL REFERENCES receipts(id) ON DELETE CASCADE,
    label TEXT NOT NULL,
    fee_type TEXT NOT NULL CHECK (fee_type IN ('tax', 'service', 'delivery', 'tip', 'other')),
    amount NUMERIC(14,2) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

-- Create index for receipt_fees
CREATE INDEX idx_receipt_fees_receipt_id ON receipt_fees(receipt_id);

-- Create trigger for updated_at
CREATE TRIGGER update_receipt_fees_updated_at
    BEFORE UPDATE ON receipt_fees
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================
-- EXCHANGE RATES TABLE
-- ============================================

CREATE TABLE exchange_rates (
    base_currency CHAR(3) NOT NULL,
    target_currency CHAR(3) NOT NULL,
    rate NUMERIC(18,8) NOT NULL,
    fetched_at TIMESTAMPTZ DEFAULT now(),
    PRIMARY KEY (base_currency, target_currency)
);

-- ============================================
-- TAGS TABLE
-- ============================================

CREATE TABLE tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    color CHAR(7) NOT NULL DEFAULT '#6366f1',
    UNIQUE (user_id, name)
);

-- Create index for tags
CREATE INDEX idx_tags_user_id ON tags(user_id);

-- ============================================
-- RECEIPT TAGS TABLE (join table)
-- ============================================

CREATE TABLE receipt_tags (
    receipt_id UUID NOT NULL REFERENCES receipts(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (receipt_id, tag_id)
);

-- ============================================
-- BUDGETS TABLE
-- ============================================

CREATE TABLE budgets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tag_id UUID REFERENCES tags(id) ON DELETE SET NULL,
    month CHAR(7) NOT NULL,  -- YYYY-MM format
    amount_limit NUMERIC(14,2) NOT NULL,
    UNIQUE (user_id, tag_id, month)
);

-- ============================================
-- SEED SYSTEM SETTINGS
-- ============================================

INSERT INTO system_settings (key, value) VALUES
    ('llm_provider', 'openai'),
    ('llm_model', 'gpt-4'),
    ('ocr_threshold', '0.85'),
    ('fx_base_currency', 'USD'),
    ('fx_provider', 'exchangerate-api');
