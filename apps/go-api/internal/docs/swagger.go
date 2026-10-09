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
    "description": "Enterprise-grade Go REST API untuk Face Attendance SaaS Platform (Santri, Siswa, Guru, Pegawai, Karyawan) dengan pgvector Cosine Distance & InsightFace AI Engine.",
    "version": "1.0.0"
  },
  "servers": [
    {
      "url": "/",
      "description": "Base Server"
    }
  ],
  "components": {
    "securitySchemes": {
      "BearerAuth": {
        "type": "http",
        "scheme": "bearer",
        "bearerFormat": "JWT"
      }
    },
    "schemas": {
      "LoginRequest": {
        "type": "object",
        "required": ["email", "password"],
        "properties": {
          "email": { "type": "string", "format": "email", "example": "wahidalimudin672@gmail.com" },
          "password": { "type": "string", "example": "Password123!" }
        }
      },
      "CreateTenantRequest": {
        "type": "object",
        "required": ["name", "subdomain", "admin_email", "admin_password"],
        "properties": {
          "name": { "type": "string", "example": "Pondok Pesantren Al-Hidayah" },
          "subdomain": { "type": "string", "example": "alhidayah" },
          "admin_name": { "type": "string", "example": "Ustadz Fauzan" },
          "admin_email": { "type": "string", "format": "email", "example": "admin@alhidayah.ponpes.id" },
          "admin_password": { "type": "string", "minLength": 6, "example": "Password123!" },
          "admin_user_code": { "type": "string", "example": "ADM-001" }
        }
      },
      "CreateOfficeRequest": {
        "type": "object",
        "required": ["name", "latitude", "longitude"],
        "properties": {
          "name": { "type": "string", "example": "Kampus Utama" },
          "latitude": { "type": "number", "format": "double", "example": -6.2088 },
          "longitude": { "type": "number", "format": "double", "example": 106.8456 },
          "radius_meters": { "type": "integer", "example": 100 }
        }
      },
      "CreateUserRequest": {
        "type": "object",
        "required": ["name", "password"],
        "properties": {
          "name": { "type": "string", "example": "Ahmad Dahlan" },
          "email": { "type": "string", "format": "email", "example": "ahmad@sekolah.sch.id", "description": "Opsional untuk siswa/santri" },
          "password": { "type": "string", "minLength": 6, "example": "Password123!" },
          "user_code": { "type": "string", "example": "SISWA-001", "description": "NIS / NISN / NIK / ID Santri" },
          "office_id": { "type": "string", "format": "uuid", "example": "b0000000-0000-0000-0000-000000000001" }
        }
      },
      "BulkCreateUsersRequest": {
        "type": "object",
        "required": ["users"],
        "properties": {
          "users": {
            "type": "array",
            "items": { "$ref": "#/components/schemas/CreateUserRequest" }
          }
        }
      },
      "BulkDeleteUsersRequest": {
        "type": "object",
        "required": ["user_ids"],
        "properties": {
          "user_ids": {
            "type": "array",
            "items": { "type": "string", "format": "uuid" },
            "example": ["c0000000-0000-0000-0000-000000000002"]
          }
        }
      }
    }
  },
  "paths": {
    "/health": {
      "get": {
        "tags": ["System"],
        "summary": "Health check service, PostgreSQL pgvector, and AI Engine",
        "responses": { "200": { "description": "Health status" } }
      }
    },
    "/api/v1/platform/settings": {
      "get": {
        "tags": ["Platform Branding"],
        "summary": "Get platform settings and branding (Public)",
        "responses": { "200": { "description": "Platform settings" } }
      }
    },
    "/api/v1/auth/login": {
      "post": {
        "tags": ["Authentication"],
        "summary": "User authentication (Super Admin, Tenant Admin, User)",
        "requestBody": {
          "required": true,
          "content": {
            "application/json": { "schema": { "$ref": "#/components/schemas/LoginRequest" } }
          }
        },
        "responses": { "200": { "description": "Token JWT and user profile" } }
      }
    },
    "/api/v1/superadmin/settings": {
      "put": {
        "tags": ["Super Admin"],
        "summary": "Update platform branding and configuration",
        "security": [{ "BearerAuth": [] }],
        "responses": { "200": { "description": "Settings updated" } }
      }
    },
    "/api/v1/superadmin/tenants": {
      "get": {
        "tags": ["Super Admin"],
        "summary": "List all tenants with pagination and search",
        "security": [{ "BearerAuth": [] }],
        "parameters": [
          { "name": "page", "in": "query", "schema": { "type": "integer", "default": 1 } },
          { "name": "limit", "in": "query", "schema": { "type": "integer", "default": 20 } },
          { "name": "search", "in": "query", "schema": { "type": "string" } }
        ],
        "responses": { "200": { "description": "List of tenants" } }
      },
      "post": {
        "tags": ["Super Admin"],
        "summary": "Create new tenant institution + initial tenant admin",
        "security": [{ "BearerAuth": [] }],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": { "schema": { "$ref": "#/components/schemas/CreateTenantRequest" } }
          }
        },
        "responses": { "201": { "description": "Tenant created" } }
      }
    },
    "/api/v1/superadmin/tenants/{id}": {
      "get": {
        "tags": ["Super Admin"],
        "summary": "Get tenant detail",
        "security": [{ "BearerAuth": [] }],
        "parameters": [{ "name": "id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }],
        "responses": { "200": { "description": "Tenant detail" } }
      },
      "put": {
        "tags": ["Super Admin"],
        "summary": "Update tenant name, subdomain, or active status",
        "security": [{ "BearerAuth": [] }],
        "parameters": [{ "name": "id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }],
        "responses": { "200": { "description": "Tenant updated" } }
      }
    },
    "/api/v1/offices": {
      "get": {
        "tags": ["Offices & Geofencing"],
        "summary": "List all campus/offices for tenant",
        "security": [{ "BearerAuth": [] }],
        "responses": { "200": { "description": "List of offices" } }
      },
      "post": {
        "tags": ["Offices & Geofencing"],
        "summary": "Create new campus/office geofencing location",
        "security": [{ "BearerAuth": [] }],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": { "schema": { "$ref": "#/components/schemas/CreateOfficeRequest" } }
          }
        },
        "responses": { "201": { "description": "Office created" } }
      }
    },
    "/api/v1/offices/{id}": {
      "get": {
        "tags": ["Offices & Geofencing"],
        "summary": "Get office detail",
        "security": [{ "BearerAuth": [] }],
        "parameters": [{ "name": "id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }],
        "responses": { "200": { "description": "Office detail" } }
      },
      "put": {
        "tags": ["Offices & Geofencing"],
        "summary": "Update office coordinates or radius",
        "security": [{ "BearerAuth": [] }],
        "parameters": [{ "name": "id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }],
        "responses": { "200": { "description": "Office updated" } }
      },
      "delete": {
        "tags": ["Offices & Geofencing"],
        "summary": "Delete office location",
        "security": [{ "BearerAuth": [] }],
        "parameters": [{ "name": "id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }],
        "responses": { "200": { "description": "Office deleted" } }
      }
    },
    "/api/v1/users": {
      "get": {
        "tags": ["Users Management"],
        "summary": "List users (students/staff) with pagination and filters",
        "security": [{ "BearerAuth": [] }],
        "parameters": [
          { "name": "page", "in": "query", "schema": { "type": "integer", "default": 1 } },
          { "name": "limit", "in": "query", "schema": { "type": "integer", "default": 20 } },
          { "name": "search", "in": "query", "schema": { "type": "string" } },
          { "name": "office_id", "in": "query", "schema": { "type": "string", "format": "uuid" } },
          { "name": "role", "in": "query", "schema": { "type": "string" } },
          { "name": "is_active", "in": "query", "schema": { "type": "boolean" } }
        ],
        "responses": { "200": { "description": "List of users" } }
      },
      "post": {
        "tags": ["Users Management"],
        "summary": "Create single user",
        "security": [{ "BearerAuth": [] }],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": { "schema": { "$ref": "#/components/schemas/CreateUserRequest" } }
          }
        },
        "responses": { "201": { "description": "User created" } }
      }
    },
    "/api/v1/users/{id}": {
      "get": {
        "tags": ["Users Management"],
        "summary": "Get user detail",
        "security": [{ "BearerAuth": [] }],
        "parameters": [{ "name": "id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }],
        "responses": { "200": { "description": "User detail" } }
      },
      "put": {
        "tags": ["Users Management"],
        "summary": "Update user data",
        "security": [{ "BearerAuth": [] }],
        "parameters": [{ "name": "id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }],
        "responses": { "200": { "description": "User updated" } }
      },
      "delete": {
        "tags": ["Users Management"],
        "summary": "Delete single user",
        "security": [{ "BearerAuth": [] }],
        "parameters": [{ "name": "id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }],
        "responses": { "200": { "description": "User deleted" } }
      }
    },
    "/api/v1/users/bulk": {
      "post": {
        "tags": ["Users Management"],
        "summary": "Bulk create users via JSON",
        "security": [{ "BearerAuth": [] }],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": { "schema": { "$ref": "#/components/schemas/BulkCreateUsersRequest" } }
          }
        },
        "responses": { "201": { "description": "Bulk create completed" } }
      },
      "delete": {
        "tags": ["Users Management"],
        "summary": "Bulk delete users by IDs",
        "security": [{ "BearerAuth": [] }],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": { "schema": { "$ref": "#/components/schemas/BulkDeleteUsersRequest" } }
          }
        },
        "responses": { "200": { "description": "Bulk delete completed" } }
      }
    },
    "/api/v1/users/import-csv": {
      "post": {
        "tags": ["Users Management"],
        "summary": "Import users via CSV file upload",
        "security": [{ "BearerAuth": [] }],
        "requestBody": {
          "required": true,
          "content": {
            "multipart/form-data": {
              "schema": {
                "type": "object",
                "properties": {
                  "file": { "type": "string", "format": "binary", "description": "CSV file (name,email,password,user_code,office_id)" }
                }
              }
            }
          }
        },
        "responses": { "201": { "description": "Import completed" } }
      }
    },
    "/api/v1/users/me": {
      "get": {
        "tags": ["Users Management"],
        "summary": "Get caller profile",
        "security": [{ "BearerAuth": [] }],
        "responses": { "200": { "description": "Current user profile" } }
      }
    },
    "/api/v1/users/enroll-face": {
      "post": {
        "tags": ["Users Management"],
        "summary": "Register master face photos for AI recognition",
        "security": [{ "BearerAuth": [] }],
        "requestBody": {
          "required": true,
          "content": {
            "multipart/form-data": {
              "schema": {
                "type": "object",
                "properties": {
                  "images": { "type": "array", "items": { "type": "string", "format": "binary" }, "description": "3-5 selfie photos" },
                  "user_id": { "type": "string", "format": "uuid", "description": "Target user ID (admin only)" }
                }
              }
            }
          }
        },
        "responses": { "200": { "description": "Face registered" } }
      }
    },
    "/api/v1/attendance/check-in": {
      "post": {
        "tags": ["Attendance"],
        "summary": "Perform face recognition check-in with GPS validation",
        "security": [{ "BearerAuth": [] }],
        "requestBody": {
          "required": true,
          "content": {
            "multipart/form-data": {
              "schema": {
                "type": "object",
                "required": ["image", "latitude", "longitude", "device_id"],
                "properties": {
                  "image": { "type": "string", "format": "binary" },
                  "latitude": { "type": "number", "format": "double" },
                  "longitude": { "type": "number", "format": "double" },
                  "device_id": { "type": "string" },
                  "type": { "type": "string", "enum": ["IN", "OUT"], "default": "IN" }
                }
              }
            }
          }
        },
        "responses": { "200": { "description": "Attendance recorded" } }
      }
    },
    "/api/v1/attendance/history": {
      "get": {
        "tags": ["Attendance"],
        "summary": "Get personal attendance history",
        "security": [{ "BearerAuth": [] }],
        "parameters": [{ "name": "limit", "in": "query", "schema": { "type": "integer", "default": 20 } }],
        "responses": { "200": { "description": "Personal attendance history" } }
      }
    },
    "/api/v1/attendance/logs": {
      "get": {
        "tags": ["Attendance"],
        "summary": "List all attendance logs for tenant (Admin)",
        "security": [{ "BearerAuth": [] }],
        "parameters": [
          { "name": "page", "in": "query", "schema": { "type": "integer", "default": 1 } },
          { "name": "limit", "in": "query", "schema": { "type": "integer", "default": 20 } },
          { "name": "start_date", "in": "query", "schema": { "type": "string", "format": "date", "example": "2026-10-01" } },
          { "name": "end_date", "in": "query", "schema": { "type": "string", "format": "date", "example": "2026-10-09" } },
          { "name": "user_id", "in": "query", "schema": { "type": "string", "format": "uuid" } },
          { "name": "is_valid", "in": "query", "schema": { "type": "boolean" } }
        ],
        "responses": { "200": { "description": "Attendance logs" } }
      }
    },
    "/api/v1/attendance/summary": {
      "get": {
        "tags": ["Attendance"],
        "summary": "Get daily dashboard attendance summary statistics",
        "security": [{ "BearerAuth": [] }],
        "parameters": [
          { "name": "date", "in": "query", "schema": { "type": "string", "format": "date", "example": "2026-10-09" } }
        ],
        "responses": { "200": { "description": "Summary statistics" } }
      }
    }
  }
}`

func RegisterSwaggerRoutes(router *gin.Engine) {
	router.GET("/docs", func(c *gin.Context) {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, swaggerUIHTML)
	})

	router.GET("/docs/", func(c *gin.Context) {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, swaggerUIHTML)
	})

	router.GET("/docs/swagger.json", func(c *gin.Context) {
		c.Header("Content-Type", "application/json; charset=utf-8")
		c.String(http.StatusOK, swaggerJSON)
	})
}
