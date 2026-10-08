package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port                     string
	DatabaseURL              string
	JWTSecret                string
	AIEngineURL              string
	SimilarityThreshold      float64
}

func Load() *Config {
	port := getEnv("PORT", "8080")
	dbURL := getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/face_attendance?sslmode=disable")
	jwtSecret := getEnv("JWT_SECRET", "super-secret-jwt-key-replace-in-production")
	aiEngineURL := getEnv("AI_ENGINE_URL", "http://ai-engine:5000")

	threshStr := getEnv("SIMILARITY_THRESHOLD", "0.65")
	thresh, err := strconv.ParseFloat(threshStr, 64)
	if err != nil {
		thresh = 0.65
	}

	return &Config{
		Port:                port,
		DatabaseURL:         dbURL,
		JWTSecret:           jwtSecret,
		AIEngineURL:         aiEngineURL,
		SimilarityThreshold: thresh,
	}
}

func getEnv(key, defaultVal string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return val
}
