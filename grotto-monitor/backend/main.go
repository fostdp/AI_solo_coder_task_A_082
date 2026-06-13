package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"grotto-monitor/backend/handlers"
	"grotto-monitor/backend/repository"
	"grotto-monitor/backend/services"
	"grotto-monitor/backend/websocket"
)

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return intValue
}

func main() {
	ctx := context.Background()

	dbConfig := repository.Config{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnvInt("DB_PORT", 5432),
		User:     getEnv("DB_USER", "postgres"),
		Password: getEnv("DB_PASSWORD", "postgres"),
		DBName:   getEnv("DB_NAME", "grotto_monitor"),
		SSLMode:  getEnv("DB_SSLMODE", "disable"),
	}

	db, err := repository.Connect(ctx, dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	log.Println("Successfully connected to database")

	repo := repository.NewRepository(db)

	predictionSvc := &services.WeatheringPredictionService{}
	topsisSvc := &services.TOPSISService{}

	hub := websocket.NewHub()
	go hub.Run()

	handler := handlers.NewHandler(repo, predictionSvc, topsisSvc, hub)

	r := mux.NewRouter()
	handler.RegisterRoutes(r)

	port := getEnv("PORT", "8080")
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	log.Printf("Server starting on port %s...", port)
	log.Printf("API endpoints:")
	log.Printf("  GET    /api/sites")
	log.Printf("  GET    /api/sites/{id}")
	log.Printf("  GET    /api/sites/{id}/sensors")
	log.Printf("  GET    /api/sites/{id}/data")
	log.Printf("  GET    /api/sites/{id}/weathering-rates")
	log.Printf("  GET    /api/sensors/{id}/hourly")
	log.Printf("  POST   /api/data")
	log.Printf("  GET    /api/alerts")
	log.Printf("  PUT    /api/alerts/{id}/acknowledge")
	log.Printf("  GET    /api/materials")
	log.Printf("  POST   /api/predict")
	log.Printf("  POST   /api/predict/batch")
	log.Printf("  POST   /api/topsis")
	log.Printf("  WS     /ws/alerts")

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to start server: %v", err)
	}
}
