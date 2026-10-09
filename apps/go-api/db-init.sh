#!/usr/bin/env bash

# ==============================================================================
# Script: db-init.sh
# Deskripsi: Menginisialisasi migrasi database pgvector & seed user awal
# Target Folder: apps/go-api
# ==============================================================================

set -e

# Warna output terminal
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Direktori kerja
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MIGRATIONS_DIR="${SCRIPT_DIR}/migrations"

echo -e "${BLUE}======================================================${NC}"
echo -e "${BLUE}   FaceAttendance SaaS - Database Migration & Seed    ${NC}"
echo -e "${BLUE}======================================================${NC}"

# 1. Muat file .env jika ada
if [ -f "${SCRIPT_DIR}/.env" ]; then
    echo -e "${YELLOW}[INFO] Memuat konfigurasi dari .env...${NC}"
    # Export variabel tanpa komentar
    export $(grep -v '^#' "${SCRIPT_DIR}/.env" | xargs)
fi

# Konfigurasi Default
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${POSTGRES_USER:-postgres}"
DB_PASSWORD="${POSTGRES_PASSWORD:-postgres}"
DB_NAME="${POSTGRES_DB:-face_attendance}"
CONTAINER_NAME="${CONTAINER_NAME:-faceattendance-postgres}"

# Deteksi mode eksekusi (Docker container vs local psql)
USE_DOCKER=false

if command -v docker &> /dev/null && docker ps --format '{{.Names}}' | grep -Eq "^${CONTAINER_NAME}\$"; then
    USE_DOCKER=true
    echo -e "${GREEN}[OK] Terdeteksi container Docker '${CONTAINER_NAME}' aktif.${NC}"
    echo -e "${BLUE}[INFO] Menjalankan migrasi melalui Docker exec...${NC}"
elif command -v psql &> /dev/null; then
    echo -e "${GREEN}[OK] Terdeteksi client psql lokal.${NC}"
    echo -e "${BLUE}[INFO] Menghubungkan ke ${DB_HOST}:${DB_PORT}/${DB_NAME}...${NC}"
else
    echo -e "${YELLOW}[WARN] Client 'psql' lokal atau container '${CONTAINER_NAME}' tidak aktif langsung.${NC}"
    echo -e "${YELLOW}       Mencoba koneksi psql bawaan lingkungan...${NC}"
fi

# Fungsi eksekusi SQL
execute_sql_file() {
    local sql_file="$1"
    local file_name=$(basename "$sql_file")
    echo -e "${YELLOW}--> Menjalankan migrasi: ${file_name}...${NC}"

    if [ "$USE_DOCKER" = true ]; then
        docker exec -i "${CONTAINER_NAME}" psql -U "${DB_USER}" -d "${DB_NAME}" < "$sql_file"
    else
        PGPASSWORD="${DB_PASSWORD}" psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${DB_NAME}" -f "$sql_file"
    fi
}

execute_sql_query() {
    local query="$1"
    if [ "$USE_DOCKER" = true ]; then
        docker exec -i "${CONTAINER_NAME}" psql -U "${DB_USER}" -d "${DB_NAME}" -c "$query"
    else
        PGPASSWORD="${DB_PASSWORD}" psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${DB_NAME}" -c "$query"
    fi
}

# 2. Periksa ketersediaan folder migrasi
if [ ! -d "${MIGRATIONS_DIR}" ]; then
    echo -e "${RED}[ERROR] Folder migrasi tidak ditemukan di: ${MIGRATIONS_DIR}${NC}"
    exit 1
fi

# 3. Jalankan semua file migrasi *.up.sql secara berurutan
echo -e "\n${BLUE}--- Langkah 1: Eksekusi Migrasi Skema Database (*.up.sql) ---${NC}"
shopt -s nullglob
migration_files=("${MIGRATIONS_DIR}"/*.up.sql)
shopt -u nullglob

if [ ${#migration_files[@]} -eq 0 ]; then
    echo -e "${RED}[ERROR] Tidak ditemukan file *.up.sql di folder ${MIGRATIONS_DIR}${NC}"
    exit 1
fi

# Urutkan file berdasarkan nama
IFS=$'\n' sorted_files=($(sort <<<"${migration_files[*]}"))
unset IFS

for file in "${sorted_files[@]}"; do
    execute_sql_file "$file"
done

echo -e "${GREEN}[SUCCESS] Semua migrasi skema database berhasil diterapkan!${NC}"

# 4. Pastikan Data Login Awal (Seed Admin / Employee) Terdaftar
echo -e "\n${BLUE}--- Langkah 2: Verifikasi & Upsert Kredensial Pengguna ---${NC}"

LOGIN_EMAIL="wahidalimudin672@gmail.com"
LOGIN_PASS_PLAIN="Password123!"
# Bcrypt hash dari "Password123!" dengan cost 10
BCRYPT_HASH="\$2a\$10\$jUNN.tPJ.rPU9lkAuyFFZ.ygSOfC7TqlVn/Z9XOBVWlTEI0GIC68i"

SEED_QUERY=$(cat <<EOF
-- Pastikan tenant default ada
INSERT INTO tenants (id, name, subdomain, is_active)
VALUES (
    'a0000000-0000-0000-0000-000000000001',
    'TechCorp Indonesia',
    'techcorp',
    true
)
ON CONFLICT (subdomain) DO UPDATE SET is_active = true;

-- Pastikan office geofencing default ada
INSERT INTO offices (id, tenant_id, name, latitude, longitude, radius_meters)
VALUES (
    'b0000000-0000-0000-0000-000000000001',
    'a0000000-0000-0000-0000-000000000001',
    'Kantor Pusat Jakarta',
    -6.208800,
    106.845600,
    100
)
ON CONFLICT (id) DO NOTHING;

-- Upsert akun employee Wahid Alimudin
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
    '${LOGIN_EMAIL}',
    '${BCRYPT_HASH}',
    'EMP-001',
    true
)
ON CONFLICT (email) DO UPDATE SET
    password_hash = EXCLUDED.password_hash,
    name = EXCLUDED.name,
    is_active = true,
    office_id = EXCLUDED.office_id;
EOF
)

execute_sql_query "${SEED_QUERY}"

echo -e "${GREEN}[SUCCESS] Akun login pengguna berhasil diinisialisasi/diperbarui!${NC}"

# 5. Tampilkan ringkasan data di database
echo -e "\n${BLUE}--- Ringkasan Data Database ---${NC}"
execute_sql_query "SELECT id, name, subdomain, is_active FROM tenants;"
execute_sql_query "SELECT id, name, latitude, longitude, radius_meters FROM offices;"
execute_sql_query "SELECT id, name, email, employee_code, is_active, (face_embedding IS NOT NULL) AS has_face FROM employees;"

echo -e "\n${GREEN}======================================================${NC}"
echo -e "${GREEN}   Inisialisasi Database Selesai!                    ${NC}"
echo -e "${GREEN}======================================================${NC}"
echo -e "Kredensial Login yang dapat digunakan di API:"
echo -e "  - Endpoint : ${YELLOW}POST /api/v1/auth/login${NC}"
echo -e "  - Email    : ${YELLOW}${LOGIN_EMAIL}${NC}"
echo -e "  - Password : ${YELLOW}${LOGIN_PASS_PLAIN}${NC}"
echo -e "======================================================\n"
