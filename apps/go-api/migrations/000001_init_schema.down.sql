-- ==========================================================
-- Migration: 000001_init_schema.down.sql
-- Description: Rollback all SaaS tables, indexes, and schema.
-- ==========================================================

DROP TABLE IF EXISTS attendance_logs CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS offices CASCADE;
DROP TABLE IF EXISTS tenants CASCADE;
DROP TABLE IF EXISTS platform_settings CASCADE;
