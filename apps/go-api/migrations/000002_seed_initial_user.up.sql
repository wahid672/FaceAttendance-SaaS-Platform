-- ==========================================================
-- Migration: 000002_seed_initial_user.up.sql
-- Description: Seed initial platform settings, superadmin,
--              demo tenant, office, and tenant admin.
--              Superadmin Email: wahidalimudin672@gmail.com
--              Password: Password123!
-- ==========================================================

-- 1. Default Platform Settings (Branding Platform SaaS)
INSERT INTO platform_settings (
    id,
    app_name,
    logo_url,
    favicon_url,
    company_name,
    support_email,
    footer_text
) VALUES (
    1,
    'FaceAttendance SaaS Platform',
    'https://raw.githubusercontent.com/faceattendance/assets/main/logo.png',
    'https://raw.githubusercontent.com/faceattendance/assets/main/favicon.ico',
    'PT Face Attendance Nusantara',
    'support@faceattendance.id',
    '© 2026 FaceAttendance SaaS Platform. All rights reserved.'
)
ON CONFLICT (id) DO UPDATE SET
    app_name = EXCLUDED.app_name,
    company_name = EXCLUDED.company_name;

-- 2. Super Admin (Global SaaS Owner)
-- Email: wahidalimudin672@gmail.com
-- Password: Password123!
-- Bcrypt Hash: $2a$10$jUNN.tPJ.rPU9lkAuyFFZ.ygSOfC7TqlVn/Z9XOBVWlTEI0GIC68i
INSERT INTO users (
    id,
    tenant_id,
    office_id,
    role,
    name,
    email,
    password_hash,
    user_code,
    is_active
) VALUES (
    '00000000-0000-0000-0000-000000000001',
    NULL,
    NULL,
    'superadmin',
    'Wahid Alimudin (Super Admin)',
    'wahidalimudin672@gmail.com',
    '$2a$10$jUNN.tPJ.rPU9lkAuyFFZ.ygSOfC7TqlVn/Z9XOBVWlTEI0GIC68i',
    'SUPERADMIN-01',
    true
)
ON CONFLICT (email) DO UPDATE SET
    role = 'superadmin',
    password_hash = EXCLUDED.password_hash,
    name = EXCLUDED.name,
    is_active = true;

-- 3. Initial Demo Tenant (Lembaga / Pondok Pesantren Demo)
INSERT INTO tenants (id, name, subdomain, is_active)
VALUES (
    'a0000000-0000-0000-0000-000000000001',
    'Pondok Pesantren Al-Hidayah Demo',
    'alhidayah',
    true
)
ON CONFLICT (subdomain) DO UPDATE SET
    is_active = true;

-- 4. Initial Demo Office (Kampus / Gedung Utama)
INSERT INTO offices (id, tenant_id, name, latitude, longitude, radius_meters)
VALUES (
    'b0000000-0000-0000-0000-000000000001',
    'a0000000-0000-0000-0000-000000000001',
    'Kampus Utama Pusat',
    -6.208800,
    106.845600,
    100
)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    latitude = EXCLUDED.latitude,
    longitude = EXCLUDED.longitude,
    radius_meters = EXCLUDED.radius_meters;

-- 5. Tenant Admin (Admin Lembaga Demo)
INSERT INTO users (
    id,
    tenant_id,
    office_id,
    role,
    name,
    email,
    password_hash,
    user_code,
    is_active
) VALUES (
    'c0000000-0000-0000-0000-000000000001',
    'a0000000-0000-0000-0000-000000000001',
    'b0000000-0000-0000-0000-000000000001',
    'tenant_admin',
    'Ustadz Fauzan (Admin Lembaga)',
    'admin@alhidayah.ponpes.id',
    '$2a$10$jUNN.tPJ.rPU9lkAuyFFZ.ygSOfC7TqlVn/Z9XOBVWlTEI0GIC68i',
    'ADM-001',
    true
)
ON CONFLICT (email) DO UPDATE SET
    role = 'tenant_admin',
    password_hash = EXCLUDED.password_hash,
    name = EXCLUDED.name,
    is_active = true,
    office_id = EXCLUDED.office_id;
