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
	cfg := config.Load()

	log.Printf("[INFO] Starting FaceAttendance Go API on port %s...", cfg.Port)
	log.Printf("[INFO] AI Engine URL: %s", cfg.AIEngineURL)
	log.Printf("[INFO] Similarity threshold: %.2f", cfg.SimilarityThreshold)

	// Context for database connection and server shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Initialize PostgreSQL Connection Pool
	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("[FATAL] Failed to parse database config: %v", err)
	}
	poolConfig.MaxConns = 25
	poolConfig.MinConns = 5
	poolConfig.MaxConnLifetime = 1 * time.Hour
	poolConfig.MaxConnIdleTime = 15 * time.Minute

	dbPool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		log.Fatalf("[FATAL] Unable to create database connection pool: %v", err)
	}
	defer dbPool.Close()

	// Ping database
	if err := dbPool.Ping(ctx); err != nil {
		log.Printf("[WARN] Warning: Database ping failed (will retry during operations): %v", err)
	} else {
		log.Println("[INFO] Successfully connected to PostgreSQL with pgvector.")
	}

	// Initialize Repositories
	tenantRepo := repository.NewTenantRepository(dbPool)
	_ = tenantRepo // preserved for tenant management endpoints
	officeRepo := repository.NewOfficeRepository(dbPool)
	employeeRepo := repository.NewEmployeeRepository(dbPool)
	attendanceRepo := repository.NewAttendanceRepository(dbPool)

	// Initialize AI Client and Services
	aiClient := service.NewAIEngineClient(cfg.AIEngineURL)
	authService := service.NewAuthService(cfg, employeeRepo)
	employeeService := service.NewEmployeeService(employeeRepo, aiClient)
	attendanceService := service.NewAttendanceService(cfg, employeeRepo, officeRepo, attendanceRepo, aiClient)

	// Initialize Handlers
	authHandler := handler.NewAuthHandler(authService)
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
		// Public Auth routes
		authGroup := apiV1.Group("/auth")
		{
			authGroup.POST("/login", authHandler.Login)
		}

		// Protected Employee & Attendance routes
		protected := apiV1.Group("")
		protected.Use(middleware.AuthMiddleware(authService))
		{
			// Employees
			employees := protected.Group("/employees")
			{
				employees.POST("", employeeHandler.CreateEmployee)
				employees.GET("/me", employeeHandler.GetProfile)
				employees.POST("/enroll-face", employeeHandler.EnrollFace)
				employees.DELETE("/:id", employeeHandler.DeleteEmployee)
			}

			// Attendance
			attendance := protected.Group("/attendance")
			{
				attendance.POST("/check-in", attendanceHandler.CheckIn)
				attendance.GET("/history", attendanceHandler.GetHistory)
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
