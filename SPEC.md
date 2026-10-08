# SPEC.MD - ARSITEKTUR SAAS ABSENSI BERBASIS FACE RECOGNITION

Dokumen ini adalah Single Source of Truth (SSOT) untuk pengembangan sistem SaaS Absensi Wajah bertahap menggunakan agen Antigravity.

---

## 1. IKHTISAR SISTEM

* **Nama Sistem:** FaceAttendance SaaS Platform
* **Model Deployment:** Multi-tenant Monorepo (Containerized via Docker & GHCR)
* **Target Infrastruktur:** VPS Linux (Ubuntu 22.04/24.04), Docker Compose, Caddy Reverse Proxy (Auto-SSL)
* **Klien:**
  1. Android Native (Kotlin + CameraX + ML Kit) - Absensi & Perekaman Lapangan
  2. Web Portal (Next.js 14/15 App Router) - Dashboard HRD & Admin Tenant

---

## 2. STRUKTUR WORKSPACE / MONOREPO

```text
absensi-saas/
├── SPEC.md                       # File spesifikasi ini
├── .github/
│   └── workflows/
│       └── build-deploy.yml      # CI/CD otomatis build & push ke ghcr.io
├── apps/
│   ├── ai-engine/                # Microservice Python (Inference Face)
│   ├── go-api/                   # Core Backend Golang (Multi-tenant & Logic)
│   ├── web/                      # Admin & Tenant Dashboard (Next.js)
│   └── android/                  # Android Native Client (Kotlin)
├── deploy/
│   ├── docker-compose.prod.yml   # Konfigurasi container produksi
│   ├── Caddyfile                 # Routing domain & HTTPS
│   └── .env.example              # Template variabel lingkungan
└── migrations/                   # SQL Migration files (Postgres + pgvector)
```

---

## 3. SKEMA DATABASE & VEKTOR (POSTGRESQL + PGVECTOR)

Ekstensi yang wajib diaktifkan:
```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "vector";
```

### 3.1 Tabel Entitas Utama

```sql
-- 1. Tenant (Perusahaan Pengguna SaaS)
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    subdomain VARCHAR(50) UNIQUE NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 2. Kantor / Titik Geofencing
CREATE TABLE offices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    latitude DOUBLE PRECISION NOT NULL,
    longitude DOUBLE PRECISION NOT NULL,
    radius_meters INTEGER DEFAULT 50, -- Ambang batas toleransi jarak
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 3. Karyawan (Pengguna Aplikasi Android)
CREATE TABLE employees (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    office_id UUID REFERENCES offices(id),
    name VARCHAR(150) NOT NULL,
    email VARCHAR(150) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    employee_code VARCHAR(50) NOT NULL,
    face_embedding vector(512),         -- Master Vector dari InsightFace ArcFace
    face_registered_at TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Index HNSW untuk kecepatan pencarian vektor
CREATE INDEX idx_employees_face ON employees USING hnsw (face_embedding vector_cosine_ops);

-- 4. Transaksi Presensi
CREATE TABLE attendances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    employee_id UUID REFERENCES employees(id) ON DELETE CASCADE,
    clock_time TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    attendance_type VARCHAR(10) NOT NULL, -- 'IN' atau 'OUT'
    similarity_score FLOAT NOT NULL,      -- Skor pencocokan AI (0.0 - 1.0)
    latitude DOUBLE PRECISION NOT NULL,
    longitude DOUBLE PRECISION NOT NULL,
    distance_meters FLOAT NOT NULL,       -- Jarak dari titik kantor saat absensi
    device_id VARCHAR(100) NOT NULL,
    photo_url TEXT,
    is_valid BOOLEAN NOT NULL
);
```

---

## 4. KONTRAK ANTAR-SERVICE (API CONTRACTS)

### 4.1 Internal API: Python AI Service (`http://ai-engine:5000`)

* **Base Model:** InsightFace (`buffalo_l` / ArcFace ONNX, 512-dim output).
* **POST `/extract`:**
  * Request: `multipart/form-data` -> `image: File`
  * Response:
    ```json
    {
      "success": true,
      "detected_faces": 1,
      "embedding": [0.0241, -0.0512, "... (512 float values)"]
    }
    ```
* **POST `/enroll-merge`:**
  * Request: `multipart/form-data` -> `images: File[]` (3-5 foto pose wajah)
  * Response:
    ```json
    {
      "success": true,
      "valid_samples": 3,
      "averaged_embedding": [0.0121, -0.0401, "... (512 float values)"]
    }
    ```

### 4.2 Public Core API: Golang Service (`go-api:8080`)

* **POST `/api/v1/auth/login`:** Login JWT karyawan/admin.
* **POST `/api/v1/employee/enroll-face`:**
  * Input: 3–5 file foto wajah.
  * Alur: Validasi tenant/user -> Lempar ke AI Engine `/enroll-merge` -> Simpan vektor ke Postgres.
* **POST `/api/v1/attendance/check-in`:**
  * Input: `image: File`, `latitude: Float`, `longitude: Float`, `device_id: String`.
  * Alur:
    1. Validasi Haversine distance terhadap koordinat kantor.
    2. Kirim gambar ke AI Engine `/extract`.
    3. Query Postgres untuk hitung Cosine Similarity: `1 - (face_embedding <=> input_vector)`.
    4. Cek ambang batas: Jika `similarity >= 0.65` dan `distance <= office.radius_meters`, simpan dengan `is_valid = true`.

---

## 5. RENCANA KERJA TAHAP DEMI TAHAP (ROADMAP ANTIGRAVITY)

Gunakan pembagian sesi berikut saat memberi perintah ke agen Antigravity:

### Sesi 1: Python AI Engine (`apps/ai-engine`)
1. Inisialisasi FastAPI dengan dependency: `insightface`, `onnxruntime`, `opencv-python-headless`.
2. Implementasi `/extract` dan `/enroll-merge` dengan deteksi single face per gambar.
3. Buat `Dockerfile` berbasis `python:3.11-slim`.
4. **Kriteria Sukses:** Dapat merespons vektor 512 dimensi via Postman/cURL.

### Sesi 2: Database & Core API (`apps/go-api` & `migrations`)
1. Buat skrip migrasi Postgres dengan ekstensi `pgvector`.
2. Setup Golang HTTP Server (menggunakan Fiber / Gin / Chi) dan GORM atau SQLX.
3. Buat modul autentikasi JWT dan multi-tenancy context.
4. Buat handler `/enroll-face` dan `/check-in` dengan integrasi ke service Python.
5. Buat `Dockerfile` Go multi-stage build.
6. **Kriteria Sukses:** Alur absensi bisa dites end-to-end via cURL menggunakan dummy coordinates dan foto.

### Sesi 3: Infrastruktur & CI/CD (`deploy/` & `.github/`)
1. Konfigurasi `docker-compose.prod.yml` (Postgres pgvector, Redis, Go API, Python AI Engine, Caddy).
2. Konfigurasi `Caddyfile` untuk auto SSL.
3. Buat GitHub Actions Workflow untuk build & push image ke `ghcr.io`.
4. **Kriteria Sukses:** Semua container dapat saling terhubung dalam satu network bridge internal.

### Sesi 4: Android App Module (`apps/android`)
1. Setup proyek Kotlin dengan Jetpack Compose & CameraX.
2. Integrasi Google ML Kit Face Detection untuk *liveness check* (deteksi kedipan / `leftEyeOpenProbability` & `rightEyeOpenProbability`).
3. Proteksi Mock Location menggunakan `Location.isMock`.
4. Implementasi UI wizard perekaman 3 pose wajah dan integrasi Retrofit ke Go API.
5. **Kriteria Sukses:** HP Android dapat mengambil foto, lolos liveness, dan sukses check-in.

### Sesi 5: Web Admin Dashboard (`apps/web`)
1. Setup Next.js App Router dengan Tailwind CSS & shadcn/ui.
2. Halaman monitoring presensi kehadiran real-time.
3. Komponen setting radius geofence kantor berbasis peta interaktif (Leaflet).
4. Buat `Dockerfile` Next.js standalone build.