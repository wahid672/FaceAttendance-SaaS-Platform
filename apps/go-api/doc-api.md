# Dokumentasi API - FaceAttendance Go API

Dokumentasi resmi penggunaan REST API untuk backend service **FaceAttendance Go API**. Backend ini dibangun menggunakan Go (Gin Framework), PostgreSQL dengan ekstensi **pgvector**, serta integrasi microservice InsightFace (AI Engine).

---

## 1. Ikhtisar & Arsitektur

- **Base URL (Lokal)**: `http://localhost:8080`
- **Base URL (Docker/Caddy)**: `http://faceattendance-go-api:8080` atau domain via reverse proxy Caddy (`caddy_net`).
- **Format Pertukaran Data**: JSON (`application/json`) dan Multipart Form (`multipart/form-data`) untuk upload citra wajah.
- **Skema Autentikasi**: JSON Web Token (JWT) dikirimkan melalui HTTP Header:
  ```http
  Authorization: Bearer <token_jwt>
  ```

### Alur Kerja Utama (Workflow)
```
1. Login Akun (/auth/login)
   └── Dapatkan JWT Token & simpan di client (Android / Web)
2. Cek Profil & Status Enrolled (/employees/me)
   ├── Jika is_enrolled == false -> Jalankan Face Enrollment
   └── Jika is_enrolled == true  -> Siap melakukan presensi
3. Registrasi Wajah / Enrollment (/employees/enroll-face)
   └── Upload 3-5 foto wajah -> Ekstraksi embedding 512 dimensi -> Simpan master vector di pgvector
4. Transaksi Presensi (/attendance/check-in)
   └── Kirim selfie + koordinat GPS -> Validasi Geofence & Cosine Similarity -> Catat log presensi
5. Riwayat Presensi (/attendance/history)
   └── Ambil log presensi pegawai
```

---

## 2. Ringkasan Endpoint

| No | Kategori | Method | Endpoint | Autentikasi | Deskripsi Singkat |
|---|---|---|---|---|---|
| 1 | Documentation | `GET` | `/docs` | Publik | **Swagger UI** Dokumentasi Interaktif OpenAPI |
| 2 | Health | `GET` | `/health` | Publik | Status service, DB pgvector, & AI engine |
| 3 | Auth | `POST` | `/api/v1/auth/login` | Publik | Login employee & penerbitan token JWT |
| 4 | Employee | `POST` | `/api/v1/employees` | Bearer Token | Menambahkan pegawai / karyawan baru |
| 5 | Employee | `GET` | `/api/v1/employees/me` | Bearer Token | Ambil data profil employee yang login |
| 6 | Employee | `DELETE` | `/api/v1/employees/:id` | Bearer Token | Hapus akun pegawai (gagal jika hapus diri sendiri) |
| 7 | Employee | `POST` | `/api/v1/employees/enroll-face` | Bearer Token | Pendaftaran foto master wajah (vektor 512-d) |
| 8 | Attendance | `POST` | `/api/v1/attendance/check-in` | Bearer Token | Presensi masuk/pulang dengan selfie & GPS |
| 9 | Attendance | `GET` | `/api/v1/attendance/history` | Bearer Token | Riwayat riil transaksi absensi employee |

---

## 3. Detail Spesifikasi Endpoint

### 3.1. Swagger UI (Dokumentasi Interaktif)

Mengakses antarmuka interaktif OpenAPI / Swagger UI untuk melihat spesifikasi skema, model request/response, dan melakukan uji coba API (*Try it out*) langsung dari browser.

- **URL**: `/docs` atau `/docs/`
- **Method**: `GET`
- **Autentikasi**: Tidak ada (Publik)
- **OpenAPI JSON Spec**: `/docs/swagger.json`
- **Fitur**: Dilengkapi tombol **Authorize** untuk memasukkan JWT Bearer Token (`Bearer <token>`).

---

### 3.2. Health Check

Memeriksa kesehatan service Go API, konektivitas connection pool PostgreSQL (pgvector), dan microservice AI Engine (InsightFace).

- **URL**: `/health`
- **Method**: `GET`
- **Autentikasi**: Tidak ada

#### Contoh Request cURL:
```bash
curl -X GET http://localhost:8080/health
```

#### Respons Sukses (200 OK):
```json
{
  "status": "ok",
  "service": "go-api",
  "database": "healthy",
  "ai_engine": "healthy",
  "version": "1.0.0"
}
```
> **Catatan Status**:
> - `status`: `"ok"` (jika DB dan AI Engine siap) atau `"degraded"` (jika salah satu dependensi mengalami kendala).
> - `database`: `"healthy"` atau `"unreachable"`.
> - `ai_engine`: `"healthy"`, `"model_not_ready"`, `"unhealthy"`, atau `"unreachable"`.

---

### 3.3. Login Employee

Melakukan verifikasi kredensial email & password employee multi-tenant.

- **URL**: `/api/v1/auth/login`
- **Method**: `POST`
- **Content-Type**: `application/json`
- **Autentikasi**: Tidak ada

#### Request Body:
| Field | Tipe | Wajib | Keterangan |
|---|---|---|---|
| `email` | string | Ya | Email terdaftar pegawai |
| `password` | string | Ya | Password akun pegawai |

```json
{
  "email": "wahidalimudin672@gmail.com",
  "password": "Password123!"
}
```

#### Contoh Request cURL:
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "wahidalimudin672@gmail.com",
    "password": "Password123!"
  }'
```

#### Respons Sukses (200 OK):
```json
{
  "success": true,
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "employee": {
    "id": "c0000000-0000-0000-0000-000000000001",
    "tenant_id": "a0000000-0000-0000-0000-000000000001",
    "office_id": "b0000000-0000-0000-0000-000000000001",
    "name": "Wahid Alimudin",
    "email": "wahidalimudin672@gmail.com",
    "employee_code": "EMP-001",
    "is_active": true,
    "is_enrolled": false
  },
  "tenant": {
    "id": "a0000000-0000-0000-0000-000000000001",
    "name": "TechCorp Indonesia",
    "subdomain": "techcorp"
  }
}
```

#### Respons Gagal (401 Unauthorized):
```json
{
  "success": false,
  "error": "invalid email or password"
}
```

---

### 3.4. Create Employee (Tambah Karyawan Baru)

Mendaftarkan akun karyawan/pegawai baru di bawah tenant yang sama. Password akan otomatis di-hash menggunakan algoritma **bcrypt**.

- **URL**: `/api/v1/employees`
- **Method**: `POST`
- **Content-Type**: `application/json`
- **Autentikasi**: `Bearer <token>`

#### Request Body:
| Field | Tipe | Wajib | Keterangan |
|---|---|---|---|
| `name` | string | Ya | Nama lengkap pegawai |
| `email` | string | Ya | Alamat email unik pegawai |
| `password` | string | Ya | Password login (minimal 6 karakter) |
| `employee_code` | string | Ya | NIK / Nomor induk pegawai (contoh: `EMP-002`) |
| `office_id` | string (UUID) | Opsional | ID kantor penugasan geofence |

```json
{
  "name": "Siti Nurhaliza",
  "email": "siti@techcorp.com",
  "password": "Password123!",
  "employee_code": "EMP-002",
  "office_id": "b0000000-0000-0000-0000-000000000001"
}
```

#### Contoh Request cURL:
```bash
curl -X POST http://localhost:8080/api/v1/employees \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Siti Nurhaliza",
    "email": "siti@techcorp.com",
    "password": "Password123!",
    "employee_code": "EMP-002",
    "office_id": "b0000000-0000-0000-0000-000000000001"
  }'
```

#### Respons Sukses (201 Created):
```json
{
  "success": true,
  "message": "Employee created successfully",
  "employee": {
    "id": "f51950d2-97d8-4f05-87d4-0610fba0d540",
    "tenant_id": "a0000000-0000-0000-0000-000000000001",
    "office_id": "b0000000-0000-0000-0000-000000000001",
    "name": "Siti Nurhaliza",
    "email": "siti@techcorp.com",
    "employee_code": "EMP-002",
    "is_active": true,
    "is_enrolled": false,
    "created_at": "2026-10-09T09:10:00Z"
  }
}
```

#### Respons Gagal (400 Bad Request):
```json
{
  "success": false,
  "error": "email is already registered"
}
```

---

### 3.5. Get Profile Employee

Mengambil data profil lengkap employee yang saat ini sedang login, termasuk status pendaftaran wajah (`is_enrolled`).

- **URL**: `/api/v1/employees/me`
- **Method**: `GET`
- **Autentikasi**: `Bearer <token>`

#### Contoh Request cURL:
```bash
curl -X GET http://localhost:8080/api/v1/employees/me \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

#### Respons Sukses (200 OK):
```json
{
  "success": true,
  "employee": {
    "id": "c0000000-0000-0000-0000-000000000001",
    "tenant_id": "a0000000-0000-0000-0000-000000000001",
    "office_id": "b0000000-0000-0000-0000-000000000001",
    "name": "Wahid Alimudin",
    "email": "wahidalimudin672@gmail.com",
    "employee_code": "EMP-001",
    "is_active": true,
    "is_enrolled": false,
    "face_registered_at": null
  }
}
```

---

### 3.6. Delete Employee (Hapus Karyawan)

Menghapus akun karyawan berdasarkan ID UUID.
> **Validasi Proteksi Diri Sendiri**: Sistem menolak dan menggagalkan operasi jika pegawai mencoba menghapus akunnya sendiri (`target_id == caller_id`).

- **URL**: `/api/v1/employees/:id`
- **Method**: `DELETE`
- **Autentikasi**: `Bearer <token>`

#### URL Parameter:
| Parameter | Tipe | Wajib | Keterangan |
|---|---|---|---|
| `id` | string (UUID) | Ya | ID unik karyawan yang ingin dihapus |

#### Contoh Request cURL:
```bash
curl -X DELETE http://localhost:8080/api/v1/employees/f51950d2-97d8-4f05-87d4-0610fba0d540 \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

#### Respons Sukses (200 OK):
```json
{
  "success": true,
  "message": "Employee deleted successfully",
  "deleted_id": "f51950d2-97d8-4f05-87d4-0610fba0d540"
}
```

#### Respons Gagal - Hapus Diri Sendiri (400 Bad Request):
```json
{
  "success": false,
  "error": "Gagal: Tidak dapat menghapus akun pegawai diri sendiri"
}
```

#### Respons Gagal - Tidak Ditemukan (404 Not Found):
```json
{
  "success": false,
  "error": "employee not found or tenant mismatch"
}
```

---

### 3.7. Enroll Face (Pendaftaran Wajah Master)

Mendaftarkan sampel foto wajah pegawai ke sistem. AI Engine akan mengekstraksi vektor embedding representatif 512 dimensi dan menyimpannya di kolom `face_embedding` pada tabel `employees` PostgreSQL pgvector.

- **URL**: `/api/v1/employees/enroll-face`
- **Method**: `POST`
- **Content-Type**: `multipart/form-data`
- **Autentikasi**: `Bearer <token>`

#### Form-Data Parameter:
| Field | Tipe | Wajib | Keterangan |
|---|---|---|---|
| `images` | file (multiple) | Ya (salah satu) | 3-5 file gambar sampel wajah (JPEG/PNG). Disarankan multi-angle untuk akurasi optimal. |
| `image` / `file` | file (single) | Alternatif | Jika hanya mengunggah 1 file foto sampel. |
| `employee_id` | string (UUID) | Opsional | Khusus admin jika ingin mendaftarkan foto untuk pegawai lain. Jika kosong, otomatis akun sendiri. |

#### Contoh Request cURL:
```bash
curl -X POST http://localhost:8080/api/v1/employees/enroll-face \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -F "images=@/path/to/face1.jpg" \
  -F "images=@/path/to/face2.jpg" \
  -F "images=@/path/to/face3.jpg"
```

#### Respons Sukses (200 OK):
```json
{
  "success": true,
  "message": "Face enrolled and registered successfully",
  "employee_id": "c56a4180-65aa-42ec-a945-5fd21dec0538",
  "samples_processed": 3,
  "embedding_dimensions": 512
}
```

#### Respons Gagal (400 Bad Request):
```json
{
  "success": false,
  "error": "No face detected in uploaded image"
}
```

---

### 3.8. Attendance Check-In / Check-Out

Melakukan transaksi presensi masuk atau pulang. Endpoint ini melakukan **dua tahap validasi simultan**:
1. **Validasi Geofencing**: Menghitung jarak GPS koordinat pegawai terhadap kantor yang ditugaskan menggunakan formula Haversine. Jarak harus $\le \text{radius\_meters}$ kantor.
2. **Validasi Wajah (Face Match)**: Mengekstraksi vektor 512-dimensi dari foto selfie presensi via AI Engine, kemudian menghitung **Cosine Similarity** di PostgreSQL pgvector (`1 - (face_embedding <=> selfie_vector)`). Skor harus $\ge 0.65$ (`SIMILARITY_THRESHOLD`).

- **URL**: `/api/v1/attendance/check-in`
- **Method**: `POST`
- **Content-Type**: `multipart/form-data`
- **Autentikasi**: `Bearer <token>`

#### Form-Data Parameter:
| Field | Tipe | Wajib | Keterangan |
|---|---|---|---|
| `image` (atau `photo`) | file | Ya | Foto selfie wajah saat melakukan absensi |
| `latitude` | float (string) | Ya | Titik latitude GPS (contoh: `-6.2088`) |
| `longitude` | float (string) | Ya | Titik longitude GPS (contoh: `106.8456`) |
| `attendance_type` | string | Opsional | Nilai: `IN` (Masuk) atau `OUT` (Pulang). Default: `IN` |
| `device_id` | string | Opsional | Identifier hardware perangkat (contoh: `android-uuid-1234`) |

#### Contoh Request cURL:
```bash
curl -X POST http://localhost:8080/api/v1/attendance/check-in \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -F "image=@/path/to/selfie.jpg" \
  -F "latitude=-6.2088" \
  -F "longitude=106.8456" \
  -F "attendance_type=IN" \
  -F "device_id=samsung-galaxy-s24"
```

#### Respons Sukses & Terverifikasi (200 OK):
> Diberikan jika `is_valid` bernilai `true` (lolos geofence dan skor kemiripan wajah memenuhi batas).
```json
{
  "success": true,
  "message": "Attendance recorded and verified successfully",
  "data": {
    "attendance_id": "b95bbf8c-4bc7-43cf-aa34-d3434eafe2b0",
    "clock_time": "2026-10-09T08:00:15.123456Z",
    "attendance_type": "IN",
    "similarity_score": 0.8842,
    "distance_meters": 12.5,
    "allowed_radius": 50,
    "is_valid": true,
    "validation_notes": []
  }
}
```

#### Respons Ditolak / Tidak Valid (400 Bad Request):
> Catatan: Data presensi tetap tersimpan ke tabel `attendance_logs` dengan status `is_valid: false` untuk audit transparansi.
```json
{
  "success": false,
  "message": "Attendance rejected: verification criteria not met",
  "data": {
    "attendance_id": "e42fa418-8685-4ae0-ba4f-ee35ffbdc622",
    "clock_time": "2026-10-09T08:05:00.654321Z",
    "attendance_type": "IN",
    "similarity_score": 0.4851,
    "distance_meters": 1250.4,
    "allowed_radius": 50,
    "is_valid": false,
    "validation_notes": [
      "Out of geofence bounds: current distance 1250.4m exceeds office limit 50m",
      "Face match failed: similarity score 0.4851 is below minimum threshold 0.65"
    ]
  }
}
```

---

### 3.9. Riwayat Presensi (Attendance History)

Mengambil daftar transaksi presensi milik pegawai yang login, diurutkan dari yang paling baru.

- **URL**: `/api/v1/attendance/history`
- **Method**: `GET`
- **Autentikasi**: `Bearer <token>`

#### Query Parameters:
| Parameter | Tipe | Default | Keterangan |
|---|---|---|---|
| `limit` | integer | `20` | Jumlah maksimal rekaman yang diambil |

#### Contoh Request cURL:
```bash
curl -X GET "http://localhost:8080/api/v1/attendance/history?limit=10" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

#### Respons Sukses (200 OK):
```json
{
  "success": true,
  "count": 2,
  "data": [
    {
      "id": "b95bbf8c-4bc7-43cf-aa34-d3434eafe2b0",
      "tenant_id": "e02b740e-7c57-4ea2-a164-97217db52f14",
      "employee_id": "c56a4180-65aa-42ec-a945-5fd21dec0538",
      "clock_time": "2026-10-09T08:00:15Z",
      "attendance_type": "IN",
      "similarity_score": 0.8842,
      "latitude": -6.2088,
      "longitude": 106.8456,
      "distance_meters": 12.5,
      "device_id": "samsung-galaxy-s24",
      "photo_url": null,
      "is_valid": true,
      "created_at": "2026-10-09T08:00:15Z"
    },
    {
      "id": "e42fa418-8685-4ae0-ba4f-ee35ffbdc622",
      "tenant_id": "e02b740e-7c57-4ea2-a164-97217db52f14",
      "employee_id": "c56a4180-65aa-42ec-a945-5fd21dec0538",
      "clock_time": "2026-10-08T17:01:20Z",
      "attendance_type": "OUT",
      "similarity_score": 0.8610,
      "latitude": -6.2087,
      "longitude": 106.8455,
      "distance_meters": 8.1,
      "device_id": "samsung-galaxy-s24",
      "photo_url": null,
      "is_valid": true,
      "created_at": "2026-10-08T17:01:20Z"
    }
  ]
}
```

---

## 4. Panduan Penggunaan Postman

Telah disediakan file koleksi Postman siap pakai di:
📁 **`apps/go-api/postman_collection.json`**

### Langkah Import:
1. Buka aplikasi **Postman**.
2. Klik tombol **Import** (di pojok kiri atas).
3. Pilih atau Drag-and-drop file `apps/go-api/postman_collection.json`.
4. Koleksi bernama **`FaceAttendance SaaS Go API`** akan langsung muncul di workspace Postman Anda.

### Variabel Koleksi (Collection Variables):
- `base_url`: Default bernilai `http://localhost:8080`.
- `token`: Disimpan otomatis saat request **Login Employee** berhasil dijalankan.

### Fitur Otomatisasi (Auto-Token Extraction):
Pada request **Login Employee**, terdapat script *Tests* bawaan:
```javascript
if (pm.response.code === 200) {
    var jsonData = pm.response.json();
    if (jsonData.token) {
        pm.collectionVariables.set("token", jsonData.token);
        console.log("Token JWT berhasil disimpan ke variabel koleksi.");
    }
}
```
Sehingga Anda **tidak perlu meng-copy token secara manual**. Request lain (`/me`, `/enroll-face`, `/check-in`, `/history`) langsung menggunakan token tersebut via `{{token}}`.

---

## 5. Inisialisasi Database & Kredensial Awal (db-init.sh)

Tersedia script otomatisasi inisialisasi database dan migrasi schema di:
📁 **`apps/go-api/db-init.sh`**

### Cara Menjalankan Script Migrasi:
```bash
cd apps/go-api
chmod +x db-init.sh
./db-init.sh
```
> Script akan otomatis mendeteksi apakah PostgreSQL berjalan di dalam Docker container (`faceattendance-postgres`) atau PostgreSQL lokal via `psql`.

### Akun Login Awal Hasil Inisialisasi (Seed):
- **Email**: `wahidalimudin672@gmail.com`
- **Password**: `Password123!`
- **Nama Pegawai**: `Wahid Alimudin`
- **Role / Employee Code**: `EMP-001`
- **Status**: Aktif (`is_active: true`)

