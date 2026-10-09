-- ==========================================================
-- Migration: 000001_init_schema.up.sql
-- Description: Multi-tenant SaaS schema:
--              pgvector, platform_settings, tenants, offices,
--              users (with roles & optional email), and attendance_logs.
-- ==========================================================

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "vector";

-- 1. Platform Settings (Pengaturan SaaS & Branding Global - Super Admin)
CREATE TABLE IF NOT EXISTS platform_settings (
    id INTEGER PRIMARY KEY DEFAULT 1,
    app_name VARCHAR(100) NOT NULL DEFAULT 'FaceAttendance SaaS',
    logo_url TEXT,
    favicon_url TEXT,
    company_name VARCHAR(150) DEFAULT 'SaaS Provider',
    support_email VARCHAR(150),
    footer_text TEXT,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT single_row_check CHECK (id = 1)
);

-- 2. Tenants (Instansi / Lembaga / Sekolah / Pondok Pesantren Pengguna SaaS)
CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    subdomain VARCHAR(50) UNIQUE NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 3. Offices (Titik Geofencing Kampus / Gedung / Cabang)
CREATE TABLE IF NOT EXISTS offices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    latitude DOUBLE PRECISION NOT NULL,
    longitude DOUBLE PRECISION NOT NULL,
    radius_meters INTEGER DEFAULT 50, -- Ambang batas toleransi jarak (meter)
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 4. Users (Pengguna: Super Admin, Tenant Admin, dan User / Siswa / Santri / Karyawan)
-- Catatan Arsitektur:
--   - Super Admin: tenant_id NULL, role = 'superadmin', wajib email
--   - Tenant Admin: tenant_id terisi, role = 'tenant_admin', wajib email
--   - User / Siswa / Santri: role = 'user', email OPSIONAL (NULL), identitas kunci = user_code
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE, -- Nullable untuk Super Admin
    office_id UUID REFERENCES offices(id) ON DELETE SET NULL, -- Nullable
    role VARCHAR(30) NOT NULL DEFAULT 'user',                 -- 'superadmin', 'tenant_admin', 'user'
    name VARCHAR(150) NOT NULL,
    email VARCHAR(150),                                      -- Nullable, hanya wajib untuk Admin
    password_hash VARCHAR(255) NOT NULL,
    user_code VARCHAR(50) NOT NULL,                          -- NIS / NISN / NIK / ID Unik
    face_embedding vector(512),                              -- Master Vector AI ArcFace (512-dim)
    face_registered_at TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Unique index email (hanya untuk email yang tidak NULL)
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_unique ON users(email) WHERE email IS NOT NULL;

-- Unique user_code per tenant (user biasa)
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_tenant_code_unique ON users(tenant_id, user_code) WHERE tenant_id IS NOT NULL;

-- Index HNSW untuk kecepatan pencarian vektor cosine distance
CREATE INDEX IF NOT EXISTS idx_users_face ON users USING hnsw (face_embedding vector_cosine_ops);
CREATE INDEX IF NOT EXISTS idx_users_tenant ON users(tenant_id);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);

-- 5. Attendance Logs (Transaksi Presensi Wajah)
CREATE TABLE IF NOT EXISTS attendance_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    clock_time TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    attendance_type VARCHAR(10) NOT NULL DEFAULT 'IN', -- 'IN' atau 'OUT'
    similarity_score DOUBLE PRECISION NOT NULL,        -- Skor kecocokan AI (0.0 - 1.0)
    latitude DOUBLE PRECISION NOT NULL,
    longitude DOUBLE PRECISION NOT NULL,
    distance_meters DOUBLE PRECISION NOT NULL,         -- Jarak dari titik kantor saat absensi
    device_id VARCHAR(100) NOT NULL,
    photo_url TEXT,
    is_valid BOOLEAN NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_attendance_logs_tenant ON attendance_logs(tenant_id);
CREATE INDEX IF NOT EXISTS idx_attendance_logs_user ON attendance_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_attendance_logs_clock_time ON attendance_logs(clock_time);
