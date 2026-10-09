package docs

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const swaggerUIHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>FaceAttendance SaaS API - Swagger UI</title>
  <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui.css" />
  <style>
    html { box-sizing: border-box; overflow: -moz-scrollbars-vertical; overflow-y: scroll; }
    *, *:before, *:after { box-sizing: inherit; }
    body { margin:0; background: #fafafa; font-family: sans-serif; }
    .topbar { display: none !important; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui-bundle.js" crossorigin></script>
  <script src="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui-standalone-preset.js" crossorigin></script>
  <script>
    window.onload = function() {
      window.ui = SwaggerUIBundle({
        url: "/docs/swagger.json",
        dom_id: '#swagger-ui',
        deepLinking: true,
        presets: [
          SwaggerUIBundle.presets.apis,
          SwaggerUIStandalonePreset
        ],
        plugins: [
          SwaggerUIBundle.plugins.DownloadUrl
        ],
        layout: "BaseLayout",
        persistAuthorization: true
      });
    };
  </script>
</body>
</html>`

const swaggerJSON = `{
  "openapi": "3.0.3",
  "info": {
    "title": "FaceAttendance SaaS Go API",
    "description": "High-Performance Go REST API microservice untuk Face Attendance SaaS Platform dengan PostgreSQL pgvector dan InsightFace AI Engine.",
    "version": "1.0.0"
  },
  "servers": [
    {
      "url": "/",
      "description": "Current Server"
    }
  ],
  "components": {
    "securitySchemes": {
      "BearerAuth": {
        "type": "http",
        "scheme": "bearer",
        "bearerFormat": "JWT",
        "description": "Masukkan token JWT yang didapatkan dari /api/v1/auth/login"
      }
    },
    "schemas": {
      "LoginRequest": {
        "type": "object",
        "required": ["email", "password"],
        "properties": {
          "email": {
            "type": "string",
            "format": "email",
            "example": "wahidalimudin672@gmail.com"
          },
          "password": {
            "type": "string",
            "example": "Password123!"
          }
        }
      },
      "CreateEmployeeRequest": {
        "type": "object",
        "required": ["name", "email", "password", "employee_code"],
        "properties": {
          "name": {
            "type": "string",
            "example": "Siti Nurhaliza"
          },
          "email": {
            "type": "string",
            "format": "email",
            "example": "siti@techcorp.com"
          },
          "password": {
            "type": "string",
            "minLength": 6,
            "example": "Password123!"
          },
          "employee_code": {
            "type": "string",
            "example": "EMP-002"
          },
          "office_id": {
            "type": "string",
            "format": "uuid",
            "example": "b0000000-0000-0000-0000-000000000001",
            "description": "Opsional: ID kantor penugasan geofence"
          }
        }
      }
    }
  },
  "paths": {
    "/health": {
      "get": {
        "tags": ["Health Check"],
        "summary": "Pengecekan status kesehatan layanan",
        "description": "Memeriksa status hidup Go API, koneksi pgvector database, dan AI Engine.",
        "responses": {
          "200": {
            "description": "Status layanan OK / Degraded"
          }
        }
      }
    },
    "/api/v1/auth/login": {
      "post": {
        "tags": ["Authentication"],
        "summary": "Login Employee",
        "description": "Verifikasi kredensial email & password dan menerbitkan JWT token.",
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "$ref": "#/components/schemas/LoginRequest"
              }
            }
          }
        },
        "responses": {
          "200": {
            "description": "Login berhasil dan mengembalikan token"
          },
          "401": {
            "description": "Email atau password salah"
          }
        }
      }
    },
    "/api/v1/employees": {
      "post": {
        "tags": ["Employees"],
        "summary": "Tambah Karyawan Baru",
        "description": "Menambahkan data karyawan baru di bawah tenant yang sama. Password otomatis di-hash dengan bcrypt.",
        "security": [
          {
            "BearerAuth": []
          }
        ],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "$ref": "#/components/schemas/CreateEmployeeRequest"
              }
            }
          }
        },
        "responses": {
          "201": {
            "description": "Karyawan berhasil dibuat"
          },
          "400": {
            "description": "Validasi gagal / email sudah terdaftar"
          },
          "401": {
            "description": "Unauthorized"
          }
        }
      }
    },
    "/api/v1/employees/me": {
      "get": {
        "tags": ["Employees"],
        "summary": "Profil Saya",
        "description": "Mengambil data profil employee yang saat ini sedang login.",
        "security": [
          {
            "BearerAuth": []
          }
        ],
        "responses": {
          "200": {
            "description": "Data profil ditemukan"
          },
          "401": {
            "description": "Unauthorized"
          }
        }
      }
    },
    "/api/v1/employees/{id}": {
      "delete": {
        "tags": ["Employees"],
        "summary": "Hapus Karyawan",
        "description": "Menghapus akun karyawan berdasarkan ID. Gagal jika mencoba menghapus akun diri sendiri.",
        "security": [
          {
            "BearerAuth": []
          }
        ],
        "parameters": [
          {
            "name": "id",
            "in": "path",
            "required": true,
            "description": "UUID employee yang ingin dihapus",
            "schema": {
              "type": "string",
              "format": "uuid"
            }
          }
        ],
        "responses": {
          "200": {
            "description": "Karyawan berhasil dihapus"
          },
          "400": {
            "description": "Gagal: Tidak dapat menghapus akun pegawai diri sendiri / format UUID invalid"
          },
          "404": {
            "description": "Employee tidak ditemukan atau tenant mismatch"
          },
          "401": {
            "description": "Unauthorized"
          }
        }
      }
    },
    "/api/v1/employees/enroll-face": {
      "post": {
        "tags": ["Employees"],
        "summary": "Enroll Master Vector Wajah",
        "description": "Upload 3-5 foto wajah master untuk diekstraksi menjadi embedding 512-dimensi dan disimpan ke pgvector.",
        "security": [
          {
            "BearerAuth": []
          }
        ],
        "requestBody": {
          "required": true,
          "content": {
            "multipart/form-data": {
              "schema": {
                "type": "object",
                "properties": {
                  "images": {
                    "type": "array",
                    "items": {
                      "type": "string",
                      "format": "binary"
                    },
                    "description": "File foto sampel wajah (3-5 foto)"
                  },
                  "employee_id": {
                    "type": "string",
                    "format": "uuid",
                    "description": "Opsional jika admin mendaftarkan pegawai lain"
                  }
                }
              }
            }
          }
        },
        "responses": {
          "200": {
            "description": "Wajah berhasil didaftarkan"
          },
          "400": {
            "description": "Foto tidak valid atau wajah tidak terdeteksi"
          }
        }
      }
    },
    "/api/v1/attendance/check-in": {
      "post": {
        "tags": ["Attendance"],
        "summary": "Presensi Masuk / Pulang (Check-In)",
        "description": "Verifikasi presensi berbasis Geofence dan Face Match Cosine Similarity (pgvector).",
        "security": [
          {
            "BearerAuth": []
          }
        ],
        "requestBody": {
          "required": true,
          "content": {
            "multipart/form-data": {
              "schema": {
                "type": "object",
                "required": ["image", "latitude", "longitude"],
                "properties": {
                  "image": {
                    "type": "string",
                    "format": "binary",
                    "description": "Foto selfie saat absensi"
                  },
                  "latitude": {
                    "type": "string",
                    "example": "-6.2088",
                    "description": "Koordinat latitude GPS"
                  },
                  "longitude": {
                    "type": "string",
                    "example": "106.8456",
                    "description": "Koordinat longitude GPS"
                  },
                  "attendance_type": {
                    "type": "string",
                    "enum": ["IN", "OUT"],
                    "default": "IN"
                  },
                  "device_id": {
                    "type": "string",
                    "example": "samsung-s24-xyz"
                  }
                }
              }
            }
          }
        },
        "responses": {
          "200": {
            "description": "Presensi terverifikasi dan tercatat"
          },
          "400": {
            "description": "Presensi ditolak (diluar radius atau wajah tidak cocok)"
          }
        }
      }
    },
    "/api/v1/attendance/history": {
      "get": {
        "tags": ["Attendance"],
        "summary": "Riwayat Presensi",
        "description": "Mengambil log transaksi presensi milik user yang login.",
        "security": [
          {
            "BearerAuth": []
          }
        ],
        "parameters": [
          {
            "name": "limit",
            "in": "query",
            "schema": {
              "type": "integer",
              "default": 20
            },
            "description": "Batas jumlah data"
          }
        ],
        "responses": {
          "200": {
            "description": "Daftar riwayat presensi"
          }
        }
      }
    }
  }
}`

func RegisterSwaggerRoutes(router *gin.Engine) {
	docsGroup := router.Group("/docs")
	{
		docsGroup.GET("", func(c *gin.Context) {
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.String(http.StatusOK, swaggerUIHTML)
		})
		docsGroup.GET("/", func(c *gin.Context) {
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.String(http.StatusOK, swaggerUIHTML)
		})
		docsGroup.GET("/swagger.json", func(c *gin.Context) {
			c.Header("Content-Type", "application/json; charset=utf-8")
			c.String(http.StatusOK, swaggerJSON)
		})
	}
}
