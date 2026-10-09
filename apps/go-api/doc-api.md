# Dokumentasi API - FaceAttendance SaaS Go API

Dokumentasi resmi penggunaan REST API untuk backend service **FaceAttendance SaaS Go API**. Backend ini dibangun menggunakan Go (Gin Framework), PostgreSQL dengan ekstensi **pgvector**, serta integrasi microservice InsightFace (AI Engine).

Layanan ini dirancang multi-tenant dengan 3 tingkatan peran:
1. **Super Admin** (Pemilik Platform Global): Mengelola branding platform & registrasi seluruh lembaga/tenant.
2. **Tenant Admin** (Admin Lembaga/Sekolah/Pesantren): Mengelola kampus/kantor geofencing, data santri/siswa, dan rekap kehadiran.
3. **User** (Santri, Siswa, Guru, Pegawai, Karyawan): Kunci identitas utama adalah `user_code` (NIS, NISN, NIK), email bersifat opsional.

---

## 1. Ikhtisar & Arsitektur

- **Base URL (Lokal)**: `http://localhost:8080`
- **Base URL (Docker/Caddy)**: `http://faceattendance-go-api:8080`
- **Dokumentasi Interaktif**: Akses **Swagger UI** langsung di `http://localhost:8080/docs`
- **Format Pertukaran Data**: JSON (`application/json`) dan Multipart Form (`multipart/form-data`)
- **Skema Autentikasi**: Bearer Token JWT pada Header HTTP:
  ```http
  Authorization: Bearer <token_jwt>
  ```

---

## 2. Kredensial Bawaan (Seed Database)

| Peran (Role) | Email | Password | Keterangan |
|---|---|---|---|
| `superadmin` | `wahidalimudin672@gmail.com` | `Password123!` | Pemilik SaaS Global (Branding & Tenant CRUD) |
| `tenant_admin` | `admin@alhidayah.ponpes.id` | `Password123!` | Admin Pondok Pesantren Al-Hidayah Demo |

---

## 3. Ringkasan Endpoint

### A. Modul Publik & Autentikasi
| Method | Endpoint | Akses | Deskripsi |
|---|---|---|---|
| `GET` | `/docs` | Publik | Antarmuka interaktif **Swagger UI** OpenAPI 3.0 |
| `GET` | `/health` | Publik | Cek kesehatan service Go API, pgvector, dan AI engine |
| `GET` | `/api/v1/platform/settings` | Publik | Ambil nama aplikasi, logo, icon, dan branding SaaS |
| `POST` | `/api/v1/auth/login` | Publik | Login pengguna (Super Admin, Tenant Admin, User) |

### B. Modul Super Admin (Platform Owner)
| Method | Endpoint | Akses | Deskripsi |
|---|---|---|---|
| `PUT` | `/api/v1/superadmin/settings` | Super Admin | Ubah nama aplikasi, logo, icon, kontak support SaaS |
| `GET` | `/api/v1/superadmin/tenants` | Super Admin | List seluruh lembaga/tenant dengan pagination & pencarian |
| `POST` | `/api/v1/superadmin/tenants` | Super Admin | Registrasi lembaga baru + otomatis buat akun Tenant Admin pertamanya |
| `GET` | `/api/v1/superadmin/tenants/:id` | Super Admin | Detail data lembaga |
| `PUT` | `/api/v1/superadmin/tenants/:id` | Super Admin | Edit nama, subdomain, atau status aktif lembaga |

### C. Modul Geofencing Cabang / Kampus (Offices)
| Method | Endpoint | Akses | Deskripsi |
|---|---|---|---|
| `GET` | `/api/v1/offices` | Tenant Admin / User | List seluruh cabang/kampus milik lembaga |
| `POST` | `/api/v1/offices` | Tenant Admin | Tambah titik kampus/kantor geofencing baru |
| `GET` | `/api/v1/offices/:id` | Tenant Admin / User | Detail titik kampus |
| `PUT` | `/api/v1/offices/:id` | Tenant Admin | Ubah titik koordinat GPS atau radius meter |
| `DELETE` | `/api/v1/offices/:id` | Tenant Admin | Hapus titik cabang |

### D. Modul Pengguna (Users / Siswa / Santri / Pegawai)
| Method | Endpoint | Akses | Deskripsi |
|---|---|---|---|
| `GET` | `/api/v1/users` | Tenant Admin | List seluruh pengguna (filter: search, office_id, role, is_active, page, limit) |
| `GET` | `/api/v1/users/me` | Authenticated | Profil user yang sedang login |
| `GET` | `/api/v1/users/:id` | Tenant Admin | Detail 1 pengguna |
| `PUT` | `/api/v1/users/:id` | Tenant Admin | Update data pengguna (nama, NIS, email, kantor, password) |
| `POST` | `/api/v1/users` | Tenant Admin | Tambah 1 pengguna baru |
| `POST` | `/api/v1/users/bulk` | Tenant Admin | Tambah massal pengguna via JSON array |
| `POST` | `/api/v1/users/import-csv` | Tenant Admin | Import massal pengguna via upload file `.csv` |
| `DELETE` | `/api/v1/users/:id` | Tenant Admin | Hapus 1 user (proteksi anti-hapus akun sendiri) |
| `DELETE` | `/api/v1/users/bulk` | Tenant Admin | Hapus massal user via JSON array ID (auto-skip akun sendiri) |
| `POST` | `/api/v1/users/enroll-face` | Authenticated | Daftarkan master wajah AI (ekstraksi vektor 512 dimensi) |

### E. Modul Presensi & Rekapitulasi (Attendance)
| Method | Endpoint | Akses | Deskripsi |
|---|---|---|---|
| `POST` | `/api/v1/attendance/check-in` | Authenticated | Presensi selfie + koordinat GPS (AI Face Recognition) |
| `GET` | `/api/v1/attendance/history` | Authenticated | Riwayat riil absensi user sendiri |
| `GET` | `/api/v1/attendance/logs` | Tenant Admin | Log kehadiran seluruh santri/siswa (filter: tanggal, validitas, user_id) |
| `GET` | `/api/v1/attendance/summary` | Tenant Admin | Statistik ringkasan absensi hari ini (total santri, hadir, di luar radius) |

---

## 4. Contoh Penggunaan cURL & Payload

### 4.1. Buat Lembaga Baru oleh Super Admin (`POST /api/v1/superadmin/tenants`)
```bash
curl -X POST http://localhost:8080/api/v1/superadmin/tenants \
  -H "Authorization: Bearer <SUPERADMIN_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Pondok Pesantren Darussalam",
    "subdomain": "darussalam",
    "admin_name": "Kiai Haji Ahmad",
    "admin_email": "admin@darussalam.ponpes.id",
    "admin_password": "Password123!",
    "admin_user_code": "ADM-001"
  }'
```

### 4.2. Tambah Lokasi Kampus Baru (`POST /api/v1/offices`)
```bash
curl -X POST http://localhost:8080/api/v1/offices \
  -H "Authorization: Bearer <TENANT_ADMIN_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Kampus Putri Gedung B",
    "latitude": -6.208800,
    "longitude": 106.845600,
    "radius_meters": 75
  }'
```

### 4.3. List Pengguna Lembaga dengan Filter (`GET /api/v1/users`)
```bash
curl -X GET "http://localhost:8080/api/v1/users?page=1&limit=20&search=ahmad&is_active=true" \
  -H "Authorization: Bearer <TENANT_ADMIN_TOKEN>"
```

### 4.4. Rekap Log Absensi Semua Santri (`GET /api/v1/attendance/logs`)
```bash
curl -X GET "http://localhost:8080/api/v1/attendance/logs?start_date=2026-10-01&end_date=2026-10-09&is_valid=true&page=1&limit=50" \
  -H "Authorization: Bearer <TENANT_ADMIN_TOKEN>"
```

### 4.5. Ringkasan Dashboard Kehadiran Hari Ini (`GET /api/v1/attendance/summary`)
```bash
curl -X GET "http://localhost:8080/api/v1/attendance/summary?date=2026-10-09" \
  -H "Authorization: Bearer <TENANT_ADMIN_TOKEN>"
```
**Contoh Respons:**
```json
{
  "success": true,
  "data": {
    "date": "2026-10-09",
    "total_users": 150,
    "total_check_ins": 142,
    "valid_check_ins": 138,
    "invalid_check_ins": 4
  }
}
```
