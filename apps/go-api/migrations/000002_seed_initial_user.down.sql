-- ==========================================================
-- Migration: 000002_seed_initial_user.down.sql
-- Description: Rollback initial seeded records
-- ==========================================================

DELETE FROM employees WHERE email = 'wahidalimudin672@gmail.com';
DELETE FROM offices WHERE id = 'b0000000-0000-0000-0000-000000000001';
DELETE FROM tenants WHERE id = 'a0000000-0000-0000-0000-000000000001';
