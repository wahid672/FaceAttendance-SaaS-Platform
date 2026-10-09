package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/faceattendance/go-api/internal/config"
	"github.com/faceattendance/go-api/internal/docs"
	"github.com/faceattendance/go-api/internal/handler"
	"github.com/faceattendance/go-api/internal/middleware"
	"github.com/faceattendance/go-api/internal/repository"
	"github.com/faceattendance/go-api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// Load application configuration
	cfg := config.Load()

	// Initialize PostgreSQL connection pool with pgvector support
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("[FATAL] Unable to parse DATABASE_URL: %v", err)
	}
	poolConfig.MaxConns = 30
	poolConfig.MinConns = 5
	poolConfig.MaxConnLifetime = 1 * time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute

	dbPool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		log.Fatalf("[FATAL] Unable to connect to database pool: %v", err)
	}
	defer dbPool.Close()

	// Ping database
	if err := dbPool.Ping(ctx); err != nil {
		log.Printf("[WARN] Warning: Database ping failed (will retry during operations): %v", err)
	} else {
		log.Println("[INFO] Successfully connected to PostgreSQL with pgvector.")
	}

	// Initialize Repositories
	settingsRepo := repository.NewPlatformSettingsRepository(dbPool)
	tenantRepo := repository.NewTenantRepository(dbPool)
	officeRepo := repository.NewOfficeRepository(dbPool)
	employeeRepo := repository.NewEmployeeRepository(dbPool)
	attendanceRepo := repository.NewAttendanceRepository(dbPool)

	// Initialize AI Client and Services
	aiClient := service.NewAIEngineClient(cfg.AIEngineURL)
	authService := service.NewAuthService(cfg, employeeRepo)
	platformService := service.NewPlatformService(settingsRepo, tenantRepo, employeeRepo)
	officeService := service.NewOfficeService(officeRepo)
	employeeService := service.NewEmployeeService(employeeRepo, aiClient)
	attendanceService := service.NewAttendanceService(cfg, employeeRepo, officeRepo, attendanceRepo, aiClient)

	// Initialize Handlers
	authHandler := handler.NewAuthHandler(authService)
	platformHandler := handler.NewPlatformHandler(platformService)
	officeHandler := handler.NewOfficeHandler(officeService)
	employeeHandler := handler.NewEmployeeHandler(employeeService)
	attendanceHandler := handler.NewAttendanceHandler(attendanceService)

	// Setup Gin Engine
	router := gin.Default()

	// Setup CORS
	router.Use(middleware.CORSMiddleware())

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		dbStatus := "healthy"
		if err := dbPool.Ping(c.Request.Context()); err != nil {
			dbStatus = "unreachable"
		}

		aiStatus, err := aiClient.CheckHealth(c.Request.Context())
		if err != nil && aiStatus == "" {
			aiStatus = "unreachable"
		}

		overallStatus := "ok"
		if dbStatus != "healthy" || aiStatus != "healthy" {
			overallStatus = "degraded"
		}

		c.JSON(http.StatusOK, gin.H{
			"status":    overallStatus,
			"service":   "go-api",
			"database":  dbStatus,
			"ai_engine": aiStatus,
			"version":   "1.0.0",
		})
	})

	// Swagger UI Documentation endpoint (/docs)
	docs.RegisterSwaggerRoutes(router)

	// API Routes (v1)
	apiV1 := router.Group("/api/v1")
	{
		// Public Platform Branding / Settings (Untuk Tampilan Login, Logo, Favicon)
		apiV1.GET("/platform/settings", platformHandler.GetSettings)

		// Public Auth routes
		authGroup := apiV1.Group("/auth")
		{
			authGroup.POST("/login", authHandler.Login)
		}

		// Protected routes (Perlu Bearer Token JWT)
		protected := apiV1.Group("")
		protected.Use(middleware.AuthMiddleware(authService))
		{
			// ==========================================
			// 1. Super Admin Modul (Platform & Tenant Management)
			// ==========================================
			superAdmin := protected.Group("/superadmin")
			superAdmin.Use(middleware.RequireSuperAdmin())
			{
				superAdmin.PUT("/settings", platformHandler.UpdateSettings)
				superAdmin.GET("/tenants", platformHandler.ListTenants)
				superAdmin.POST("/tenants", platformHandler.CreateTenant)
				superAdmin.GET("/tenants/:id", platformHandler.GetTenant)
				superAdmin.PUT("/tenants/:id", platformHandler.UpdateTenant)
			}

			// ==========================================
			// 2. Office Geofencing Modul (Cabang / Kampus)
			// ==========================================
			offices := protected.Group("/offices")
			{
				offices.GET("", officeHandler.ListOffices)
				offices.POST("", officeHandler.CreateOffice)
				offices.GET("/:id", officeHandler.GetOffice)
				offices.PUT("/:id", officeHandler.UpdateOffice)
				offices.DELETE("/:id", officeHandler.DeleteOffice)
			}

			// ==========================================
			// 3. Users endpoints (Mencakup Siswa, Santri, Guru, Karyawan, Pegawai)
			// ==========================================
			users := protected.Group("/users")
			{
				users.GET("", employeeHandler.ListUsers)
				users.GET("/me", employeeHandler.GetProfile)
				users.GET("/:id", employeeHandler.GetUser)
				users.PUT("/:id", employeeHandler.UpdateUser)
				users.POST("", employeeHandler.CreateUser)
				users.POST("/bulk", employeeHandler.BulkCreateUsers)
				users.POST("/import-csv", employeeHandler.ImportUsersCSV)
				users.DELETE("/bulk", employeeHandler.BulkDeleteUsers)
				users.DELETE("/:id", employeeHandler.DeleteUser)
				users.POST("/enroll-face", employeeHandler.EnrollFace)
			}

			// Alias /employees untuk backward-compatibility
			employees := protected.Group("/employees")
			{
				employees.GET("", employeeHandler.ListUsers)
				employees.GET("/me", employeeHandler.GetProfile)
				employees.GET("/:id", employeeHandler.GetUser)
				employees.PUT("/:id", employeeHandler.UpdateUser)
				employees.POST("", employeeHandler.CreateUser)
				employees.POST("/bulk", employeeHandler.BulkCreateUsers)
				employees.POST("/import-csv", employeeHandler.ImportUsersCSV)
				employees.DELETE("/bulk", employeeHandler.BulkDeleteUsers)
				employees.DELETE("/:id", employeeHandler.DeleteUser)
				employees.POST("/enroll-face", employeeHandler.EnrollFace)
			}

			// ==========================================
			// 4. Attendance Modul (Check-In & Rekap Logs)
			// ==========================================
			attendance := protected.Group("/attendance")
			{
				attendance.POST("/check-in", attendanceHandler.CheckIn)
				attendance.GET("/history", attendanceHandler.GetHistory)
				attendance.GET("/logs", attendanceHandler.GetLogs)
				attendance.GET("/summary", attendanceHandler.GetSummary)
			}
		}
	}

	// HTTP Server configuration
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in background goroutine
	go func() {
		log.Printf("[INFO] Server listening on http://0.0.0.0:%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[FATAL] ListenAndServe error: %v", err)
		}
	}()

	// Graceful shutdown handling
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("[INFO] Shutting down server gracefully...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("[FATAL] Server forced to shutdown: %v", err)
	}

	log.Println("[INFO] Server exited cleanly.")
}
