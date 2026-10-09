-- ==========================================================
-- Migration: 000002_seed_initial_user.up.sql
-- Description: Seed initial tenant, office, and default login
--              User: wahidalimudin672@gmail.com
--              Password: Password123!
-- ==========================================================

-- 1. Default Tenant
INSERT INTO tenants (id, name, subdomain, is_active)
VALUES (
    'a0000000-0000-0000-0000-000000000001',
    'TechCorp Indonesia',
    'techcorp',
    true
)
ON CONFLICT (subdomain) DO UPDATE SET
    is_active = true;

-- 2. Default Office (Jakarta HQ - Geofence coordinate)
INSERT INTO offices (id, tenant_id, name, latitude, longitude, radius_meters)
VALUES (
    'b0000000-0000-0000-0000-000000000001',
    'a0000000-0000-0000-0000-000000000001',
    'Kantor Pusat Jakarta',
    -6.208800,
    106.845600,
    100
)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    latitude = EXCLUDED.latitude,
    longitude = EXCLUDED.longitude,
    radius_meters = EXCLUDED.radius_meters;

-- 3. Default Employee
-- Email: wahidalimudin672@gmail.com
-- Password: Password123!
-- Bcrypt Hash ($2a$10$jUNN.tPJ.rPU9lkAuyFFZ.ygSOfC7TqlVn/Z9XOBVWlTEI0GIC68i)
INSERT INTO employees (
    id,
    tenant_id,
    office_id,
    name,
    email,
    password_hash,
    employee_code,
    is_active
)
VALUES (
    'c0000000-0000-0000-0000-000000000001',
    'a0000000-0000-0000-0000-000000000001',
    'b0000000-0000-0000-0000-000000000001',
    'Wahid Alimudin',
    'wahidalimudin672@gmail.com',
    '$2a$10$jUNN.tPJ.rPU9lkAuyFFZ.ygSOfC7TqlVn/Z9XOBVWlTEI0GIC68i',
    'EMP-001',
    true
)
ON CONFLICT (email) DO UPDATE SET
    password_hash = EXCLUDED.password_hash,
    name = EXCLUDED.name,
    is_active = true,
    office_id = EXCLUDED.office_id;
