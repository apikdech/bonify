-- Migration: Add FX rates table and update receipt sources
-- This migration adds a dedicated fx_rates table and updates the receipt source enum

-- ============================================
-- FX RATES TABLE
-- ============================================

CREATE TABLE IF NOT EXISTS fx_rates (
    base_currency CHAR(3) NOT NULL,
    target_currency CHAR(3) NOT NULL,
    rate NUMERIC(18,8) NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT now(),
    PRIMARY KEY (base_currency, target_currency)
);

-- Create index for efficient lookups
CREATE INDEX idx_fx_rates_base ON fx_rates(base_currency);

-- ============================================
-- UPDATE RECEIPT SOURCE ENUM
-- ============================================

-- Update the receipts table to include 'ocr' as a valid source
ALTER TABLE receipts DROP CONSTRAINT IF EXISTS receipts_source_check;
ALTER TABLE receipts ADD CONSTRAINT receipts_source_check 
    CHECK (source IN ('telegram', 'discord', 'manual', 'ocr'));
