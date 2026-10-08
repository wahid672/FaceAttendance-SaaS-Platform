-- ==========================================================
-- Migration: 000001_init_schema.down.sql
-- Description: Rollback all tables, views, and extensions.
-- ==========================================================

DROP VIEW IF EXISTS attendances;
DROP TABLE IF EXISTS attendance_logs;
DROP TABLE IF EXISTS employees;
DROP TABLE IF EXISTS offices;
DROP TABLE IF EXISTS tenants;
