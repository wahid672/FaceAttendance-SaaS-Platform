-- ==========================================================
-- Migration: 000001_init_schema.up.sql
-- Description: Enable pgvector, create multi-tenant tables,
--              offices, employees with vector(512), HNSW index,
--              and attendance_logs.
-- ==========================================================

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "vector";

-- 1. Tenants (Perusahaan Pengguna SaaS)
CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    subdomain VARCHAR(50) UNIQUE NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 2. Offices (Titik Geofencing Kantor)
CREATE TABLE IF NOT EXISTS offices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    latitude DOUBLE PRECISION NOT NULL,
    longitude DOUBLE PRECISION NOT NULL,
    radius_meters INTEGER DEFAULT 50, -- Ambang batas toleransi jarak (meter)
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 3. Employees (Pengguna Aplikasi Android & Tenancy)
CREATE TABLE IF NOT EXISTS employees (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    office_id UUID REFERENCES offices(id) ON DELETE SET NULL,
    name VARCHAR(150) NOT NULL,
    email VARCHAR(150) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    employee_code VARCHAR(50) NOT NULL,
    face_embedding vector(512),         -- Master Vector dari InsightFace ArcFace (512-dim)
    face_registered_at TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Index HNSW untuk kecepatan pencarian vektor cosine distance
CREATE INDEX IF NOT EXISTS idx_employees_face ON employees USING hnsw (face_embedding vector_cosine_ops);
CREATE INDEX IF NOT EXISTS idx_employees_tenant ON employees(tenant_id);
CREATE INDEX IF NOT EXISTS idx_employees_email ON employees(email);

-- 4. Attendance Logs (Transaksi Presensi)
CREATE TABLE IF NOT EXISTS attendance_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    employee_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    clock_time TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    attendance_type VARCHAR(10) NOT NULL DEFAULT 'IN', -- 'IN' atau 'OUT'
    similarity_score DOUBLE PRECISION NOT NULL,        -- Skor pencocokan AI (0.0 - 1.0)
    latitude DOUBLE PRECISION NOT NULL,
    longitude DOUBLE PRECISION NOT NULL,
    distance_meters DOUBLE PRECISION NOT NULL,         -- Jarak dari titik kantor saat absensi
    device_id VARCHAR(100) NOT NULL,
    photo_url TEXT,
    is_valid BOOLEAN NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_attendance_logs_tenant ON attendance_logs(tenant_id);
CREATE INDEX IF NOT EXISTS idx_attendance_logs_employee ON attendance_logs(employee_id);
CREATE INDEX IF NOT EXISTS idx_attendance_logs_clock_time ON attendance_logs(clock_time);

-- View kompatibilitas untuk penamaan tabel 'attendances' pada SPEC.md
CREATE OR REPLACE VIEW attendances AS SELECT * FROM attendance_logs;
