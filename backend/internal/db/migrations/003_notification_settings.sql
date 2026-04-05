-- Migration: Add notification settings columns to users table
-- This migration adds notification preference columns to the users table

-- ============================================
-- NOTIFICATION SETTINGS
-- ============================================

-- Add notification preference columns with default TRUE (opted in by default)
ALTER TABLE users ADD COLUMN IF NOT EXISTS notify_on_parse BOOLEAN DEFAULT true;
ALTER TABLE users ADD COLUMN IF NOT EXISTS notify_on_pending_review BOOLEAN DEFAULT true;
ALTER TABLE users ADD COLUMN IF NOT EXISTS notify_budget_alerts BOOLEAN DEFAULT true;
