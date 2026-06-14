package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"grotto-monitor/backend/alarm_websocket"
	"grotto-monitor/backend/config"
	"grotto-monitor/backend/material_optimizer"
	"grotto-monitor/backend/repository"
	"grotto-monitor/backend/sensor_receiver"
	"grotto-monitor/backend/weathering_predictor"
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

	configPath := getEnv("CONFIG_PATH", "config/model_params.json")
	params, err := config.LoadModelParams(configPath)
	if err != nil {
		log.Printf("Failed to load config from %s, using defaults: %v", configPath, err)
		params = config.DefaultModelParams()
	}

	alertCh := make(chan alarm_websocket.AlertEvent, 256)

	sensorRcv := sensor_receiver.NewSensorReceiver(repo, alertCh, params.Alarm)
	predictor := weathering_predictor.NewWeatheringPredictor(repo, params.Prediction)
	optimizer := material_optimizer.NewMaterialOptimizer(repo, params.Topsis)
	alarmHub := alarm_websocket.NewHub(alertCh, repo, params.Alarm)

	go alarmHub.Run()

	r := mux.NewRouter()

	sensorRcv.RegisterRoutes(r)
	predictor.RegisterRoutes(r)
	optimizer.RegisterRoutes(r)
	alarmHub.RegisterRoutes(r)

	port := getEnv("PORT", "8080")
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	log.Printf("Server starting on port %s...", port)
	log.Printf("Config loaded from: %s", configPath)
	log.Printf("Modules: sensor_receiver | weathering_predictor | material_optimizer | alarm_websocket")
	log.Printf("API endpoints:")
	log.Printf("  [sensor_receiver]")
	log.Printf("    GET    /api/sites")
	log.Printf("    GET    /api/sites/{id}")
	log.Printf("    GET    /api/sites/{id}/sensors")
	log.Printf("    GET    /api/sites/{id}/data")
	log.Printf("    GET    /api/sites/{id}/weathering-rates")
	log.Printf("    GET    /api/sensors/{id}/hourly")
	log.Printf("    POST   /api/data")
	log.Printf("  [weathering_predictor]")
	log.Printf("    POST   /api/predict")
	log.Printf("    POST   /api/predict/batch")
	log.Printf("  [material_optimizer]")
	log.Printf("    GET    /api/materials")
	log.Printf("    POST   /api/topsis")
	log.Printf("  [alarm_websocket]")
	log.Printf("    WS     /ws/alerts")
	log.Printf("    GET    /api/alerts")
	log.Printf("    PUT    /api/alerts/{id}/acknowledge")

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to start server: %v", err)
	}
}
