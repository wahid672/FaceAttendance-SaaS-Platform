-- ==========================================================
-- Migration: 000002_seed_initial_user.down.sql
-- Description: Rollback initial seeded records
-- ==========================================================

DELETE FROM users WHERE email IN ('wahidalimudin672@gmail.com', 'admin@demo.sch.id');
DELETE FROM offices WHERE id = 'b0000000-0000-0000-0000-000000000001';
DELETE FROM tenants WHERE id = 'a0000000-0000-0000-0000-000000000001';
DELETE FROM platform_settings WHERE id = 1;
