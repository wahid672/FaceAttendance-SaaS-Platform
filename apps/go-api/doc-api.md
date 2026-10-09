# Dokumentasi API - FaceAttendance Go API

Dokumentasi resmi penggunaan REST API untuk backend service **FaceAttendance Go API**. Backend ini dibangun menggunakan Go (Gin Framework), PostgreSQL dengan ekstensi **pgvector**, serta integrasi microservice InsightFace (AI Engine).

Layanan ini dirancang multi-peran (**Siswa/Santri, Guru, Karyawan, dan Pegawai**) di bawah entitas **Users**.

---

## 1. Ikhtisar & Arsitektur

- **Base URL (Lokal)**: `http://localhost:8080`
- **Base URL (Docker/Caddy)**: `http://faceattendance-go-api:8080` atau domain via reverse proxy Caddy (`caddy_net`).
- **Dokumentasi Interaktif**: Akses **Swagger UI** langsung di `http://localhost:8080/docs`.
- **Format Pertukaran Data**: JSON (`application/json`) dan Multipart Form (`multipart/form-data`) untuk upload CSV dan foto wajah.
- **Skema Autentikasi**: JSON Web Token (JWT) dikirimkan melalui HTTP Header:
  ```http
  Authorization: Bearer <token_jwt>
  ```

### Alur Kerja Utama (Workflow)
```
1. Login Akun (/api/v1/auth/login)
   └── Dapatkan JWT Token
2. Manajemen Pengguna (Users) (/api/v1/users)
   ├── Tambah User Tunggal: POST /api/v1/users
   ├── Tambah User Massal (JSON): POST /api/v1/users/bulk
   ├── Import User Massal (CSV): POST /api/v1/users/import-csv
   ├── Hapus User Massal: DELETE /api/v1/users/bulk
   └── Hapus User Tunggal: DELETE /api/v1/users/:id
3. Registrasi Wajah / Enrollment (/api/v1/users/enroll-face)
   └── Upload 3-5 foto wajah -> Ekstraksi embedding 512 dimensi -> Simpan master vector di pgvector
4. Transaksi Presensi (/api/v1/attendance/check-in)
   └── Kirim selfie + koordinat GPS -> Validasi Geofence & Cosine Similarity -> Catat log presensi
5. Riwayat Presensi (/api/v1/attendance/history)
   └── Ambil log transaksi presensi
```

---

## 2. Ringkasan Endpoint

| No | Kategori | Method | Endpoint | Autentikasi | Deskripsi Singkat |
|---|---|---|---|---|---|
| 1 | Documentation | `GET` | `/docs` | Publik | **Swagger UI** Dokumentasi Interaktif OpenAPI 3.0 |
| 2 | Health | `GET` | `/health` | Publik | Status service, DB pgvector, & AI engine |
| 3 | Auth | `POST` | `/api/v1/auth/login` | Publik | Login user & penerbitan token JWT |
| 4 | Users | `POST` | `/api/v1/users` | Bearer Token | Tambah pengguna baru (Single) |
| 5 | Users | `POST` | `/api/v1/users/bulk` | Bearer Token | **Bulk Create** pengguna massal via JSON |
| 6 | Users | `POST` | `/api/v1/users/import-csv` | Bearer Token | **Import CSV** pengguna massal via file |
| 7 | Users | `GET` | `/api/v1/users/me` | Bearer Token | Ambil data profil user yang login |
| 8 | Users | `DELETE` | `/api/v1/users/:id` | Bearer Token | Hapus user (ditolak jika hapus diri sendiri) |
| 9 | Users | `DELETE` | `/api/v1/users/bulk` | Bearer Token | **Bulk Delete** hapus banyak user sekaligus |
| 10 | Users | `POST` | `/api/v1/users/enroll-face` | Bearer Token | Pendaftaran foto master wajah (vektor 512-d) |
| 11 | Attendance | `POST` | `/api/v1/attendance/check-in` | Bearer Token | Presensi masuk/pulang dengan selfie & GPS |
| 12 | Attendance | `GET` | `/api/v1/attendance/history` | Bearer Token | Riwayat riil transaksi absensi user |

> **Catatan Kompatibilitas**: Seluruh endpoint `/api/v1/users/*` juga dapat diakses menggunakan alias `/api/v1/employees/*` untuk menjamin kompatibilitas aplikasi klien lama.

---

## 3. Detail Spesifikasi Endpoint

### 3.1. Swagger UI (Dokumentasi Interaktif)

- **URL**: `/docs` atau `/docs/`
- **Method**: `GET`
- **OpenAPI JSON Spec**: `/docs/swagger.json`
- **Fitur**: Dilengkapi tombol **Authorize** untuk memasukkan JWT Bearer Token (`Bearer <token>`).

---

### 3.2. Health Check

- **URL**: `/health`
- **Method**: `GET`
- **Respons (200 OK)**:
```json
{
  "status": "ok",
  "service": "go-api",
  "database": "healthy",
  "ai_engine": "healthy",
  "version": "1.0.0"
}
```

---

### 3.3. Login Pengguna

- **URL**: `/api/v1/auth/login`
- **Method**: `POST`
- **Content-Type**: `application/json`

#### Akun Bawaan (Seed Database):
1. **Super Admin (Platform Owner SaaS)**:
   - **Email**: `wahidalimudin672@gmail.com`
   - **Password**: `Password123!`
   - **Role**: `superadmin` (Mengelola SaaS, platform branding/settings, dan seluruh tenant)
2. **Admin Lembaga Demo (Pondok Pesantren Demo)**:
   - **Email**: `admin@alhidayah.ponpes.id`
   - **Password**: `Password123!`
   - **Role**: `tenant_admin` (Mengelola santri/siswa/karyawan dan lokasi geofencing lembaga)

#### Request Body:
```json
{
  "email": "wahidalimudin672@gmail.com",
  "password": "Password123!"
}
```

#### Respons Sukses (200 OK) - Super Admin:
```json
{
  "success": true,
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "00000000-0000-0000-0000-000000000001",
    "role": "superadmin",
    "name": "Wahid Alimudin (Super Admin)",
    "email": "wahidalimudin672@gmail.com",
    "user_code": "SUPERADMIN-01",
    "is_active": true,
    "is_enrolled": false
  }
}
```

#### Respons Sukses (200 OK) - Admin Lembaga:
```json
{
  "success": true,
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "c0000000-0000-0000-0000-000000000001",
    "tenant_id": "a0000000-0000-0000-0000-000000000001",
    "office_id": "b0000000-0000-0000-0000-000000000001",
    "role": "tenant_admin",
    "name": "Ustadz Fauzan (Admin Lembaga)",
    "email": "admin@alhidayah.ponpes.id",
    "user_code": "ADM-001",
    "is_active": true,
    "is_enrolled": false
  },
  "tenant": {
    "id": "a0000000-0000-0000-0000-000000000001",
    "name": "Pondok Pesantren Al-Hidayah Demo",
    "subdomain": "alhidayah"
  }
}
```

---

### 3.4. Tambah Pengguna Baru (Single)

- **URL**: `/api/v1/users`
- **Method**: `POST`
- **Content-Type**: `application/json`
- **Autentikasi**: `Bearer <token>`

#### Request Body:
| Field | Tipe | Wajib | Keterangan |
|---|---|---|---|
| `name` | string | Ya | Nama lengkap pengguna (santri/siswa/pegawai) |
| `email` | string | Ya | Email unik pengguna |
| `password` | string | Ya | Password login (min 6 karakter) |
| `user_code` | string | Ya | NIS / NIK / NIP / Kode Pegawai |
| `office_id` | string (UUID) | Opsional | ID kantor penugasan geofence |

```json
{
  "name": "Ahmad Fauzi",
  "email": "ahmad@sekolah.com",
  "password": "Password123!",
  "user_code": "SISWA-001"
}
```

#### Respons Sukses (201 Created):
```json
{
  "success": true,
  "message": "User created successfully",
  "user": {
    "id": "f51950d2-97d8-4f05-87d4-0610fba0d540",
    "tenant_id": "a0000000-0000-0000-0000-000000000001",
    "office_id": null,
    "name": "Ahmad Fauzi",
    "email": "ahmad@sekolah.com",
    "user_code": "SISWA-001",
    "is_active": true,
    "is_enrolled": false,
    "created_at": "2026-10-09T09:30:00Z"
  }
}
```

---

### 3.5. Bulk Create Users (Upload Massal via JSON)

Menambahkan banyak pengguna sekaligus dalam satu request menggunakan transaksi database.

- **URL**: `/api/v1/users/bulk`
- **Method**: `POST`
- **Content-Type**: `application/json`
- **Autentikasi**: `Bearer <token>`

#### Request Body (bisa format wrapper `{ "users": [...] }` atau array langsung `[ {...} ]`):
```json
{
  "users": [
    {
      "name": "Ahmad Fauzi",
      "email": "ahmad@sekolah.com",
      "password": "Password123!",
      "user_code": "SISWA-001"
    },
    {
      "name": "Dewi Sartika",
      "email": "dewi@sekolah.com",
      "password": "Password123!",
      "user_code": "SISWA-002"
    }
  ]
}
```

#### Respons Sukses (201 Created):
```json
{
  "success": true,
  "message": "Bulk create users completed",
  "total_requested": 2,
  "success_count": 2,
  "failed_count": 0,
  "errors": [],
  "data": [
    {
      "id": "11111111-2222-3333-4444-555555555555",
      "tenant_id": "a0000000-0000-0000-0000-000000000001",
      "name": "Ahmad Fauzi",
      "email": "ahmad@sekolah.com",
      "user_code": "SISWA-001",
      "is_active": true
    },
    {
      "id": "22222222-3333-4444-5555-666666666666",
      "tenant_id": "a0000000-0000-0000-0000-000000000001",
      "name": "Dewi Sartika",
      "email": "dewi@sekolah.com",
      "user_code": "SISWA-002",
      "is_active": true
    }
  ]
}
```

---

### 3.6. Import Users via CSV (Upload Massal File)

Mengunggah file format `.csv` untuk import puluhan hingga ribuan pengguna sekaligus.

- **URL**: `/api/v1/users/import-csv`
- **Method**: `POST`
- **Content-Type**: `multipart/form-data`
- **Autentikasi**: `Bearer <token>`

#### Form-Data Parameter:
- `file`: File `.csv` (wajib)

#### Format Header File CSV:
```csv
name,email,password,user_code,office_id
Ahmad Fauzi,ahmad@sekolah.com,Password123!,SISWA-001,
Dewi Sartika,dewi@sekolah.com,Password123!,SISWA-002,
Budi Raharjo,budi.r@sekolah.com,Password123!,GURU-001,b0000000-0000-0000-0000-000000000001
```

#### Contoh cURL:
```bash
curl -X POST http://localhost:8080/api/v1/users/import-csv \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -F "file=@daftar_siswa.csv"
```

#### Respons Sukses (200 OK):
```json
{
  "success": true,
  "message": "Import users from CSV completed",
  "total_requested": 3,
  "success_count": 3,
  "failed_count": 0,
  "errors": [],
  "data": [ ... ]
}
```

---

### 3.7. Bulk Delete Users (Hapus Massal)

Menghapus banyak akun pengguna sekaligus berdasarkan daftar UUID.
> **Proteksi Diri Sendiri**: Jika array memuat ID akun sendiri yang sedang login, sistem secara otomatis **melewati akun tersebut** agar tidak terhapus.

- **URL**: `/api/v1/users/bulk`
- **Method**: `DELETE`
- **Content-Type**: `application/json`
- **Autentikasi**: `Bearer <token>`

#### Request Body:
```json
{
  "user_ids": [
    "11111111-2222-3333-4444-555555555555",
    "22222222-3333-4444-5555-666666666666"
  ]
}
```

#### Respons Sukses (200 OK):
```json
{
  "success": true,
  "message": "Bulk delete completed",
  "total_requested": 2,
  "deleted_count": 2,
  "skipped_self": false,
  "deleted_ids": [
    "11111111-2222-3333-4444-555555555555",
    "22222222-3333-4444-5555-666666666666"
  ]
}
```

---

### 3.8. Hapus Pengguna Tunggal (Single Delete)

- **URL**: `/api/v1/users/:id`
- **Method**: `DELETE`
- **Autentikasi**: `Bearer <token>`

Jika menghapus akun sendiri:
```json
{
  "success": false,
  "error": "Gagal: Tidak dapat menghapus akun diri sendiri"
}
```

---

### 3.9. Get Profile User

- **URL**: `/api/v1/users/me`
- **Method**: `GET`
- **Autentikasi**: `Bearer <token>`

---

### 3.10. Enroll Face (Pendaftaran Wajah Master)

- **URL**: `/api/v1/users/enroll-face`
- **Method**: `POST`
- **Content-Type**: `multipart/form-data`
- **Autentikasi**: `Bearer <token>`
- **Parameter**: `images` (file foto sampel 3-5 buah), `user_id` (opsional jika admin mendaftarkan user lain).

---

### 3.11. Presensi Check-In / Check-Out

- **URL**: `/api/v1/attendance/check-in`
- **Method**: `POST`
- **Content-Type**: `multipart/form-data`
- **Autentikasi**: `Bearer <token>`
- **Parameter**: `image` (selfie foto), `latitude`, `longitude`, `attendance_type` (`IN` / `OUT`).

---

### 3.12. Riwayat Presensi

- **URL**: `/api/v1/attendance/history`
- **Method**: `GET`
- **Query**: `limit` (default: 20)
- **Autentikasi**: `Bearer <token>`

---

## 4. Panduan Penggunaan Postman

File koleksi Postman siap pakai di:
📁 **`apps/go-api/postman_collection.json`**

### Folder Koleksi:
1. **0. Documentation & Health**: Swagger UI & Health Check
2. **1. Authentication**: Login (auto-token save ke `{{token}}`)
3. **2. Users**:
   - Create User (Single)
   - Bulk Create Users (JSON)
   - Import Users from CSV
   - Get Profile Me
   - Delete User (Single)
   - Bulk Delete Users
   - Enroll Face
4. **3. Attendance**:
   - Check-In / Check-Out
   - Attendance History
