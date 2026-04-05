-- Migration: Create receipt_splits table for expense splitting
-- This migration adds support for splitting receipts among multiple users

-- ============================================
-- RECEIPT SPLITS TABLE
-- ============================================

CREATE TABLE receipt_splits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    receipt_id UUID NOT NULL REFERENCES receipts(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount NUMERIC(14,2) NOT NULL DEFAULT 0,
    percentage NUMERIC(5,2),  -- Optional: for percentage-based splits (0-100)
    created_at TIMESTAMPTZ DEFAULT now()
);

-- Create indexes for efficient lookups
CREATE INDEX idx_receipt_splits_receipt_id ON receipt_splits(receipt_id);
CREATE INDEX idx_receipt_splits_user_id ON receipt_splits(user_id);

-- Ensure a user can only have one split per receipt
CREATE UNIQUE INDEX idx_receipt_splits_receipt_user ON receipt_splits(receipt_id, user_id);
