package main

import (
	"compress/gzip"
	"context"
	"io"
	"log"
	"net/http"
	"net/http/pprof"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"grotto-monitor/backend/alarm_websocket"
	"grotto-monitor/backend/config"
	"grotto-monitor/backend/material_optimizer"
	"grotto-monitor/backend/repository"
	"grotto-monitor/backend/sensor_receiver"
	"grotto-monitor/backend/weathering_predictor"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	sensorDataReceived = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "sensor_data_received_total",
			Help: "Total number of sensor data points received",
		},
	)

	alertsTriggered = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "alerts_triggered_total",
			Help: "Total number of alerts triggered",
		},
		[]string{"site_id", "alert_type", "severity"},
	)

	activeWebSocketConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "active_websocket_connections",
			Help: "Current number of active WebSocket connections",
		},
	)

	databaseQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "database_query_duration_seconds",
			Help:    "Database query duration in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
		},
		[]string{"query_type"},
	)

	dbQueryDuration   *prometheus.HistogramVec
	dbQueryDurationMu sync.Mutex
)

func getOrCreateDBHistogram() *prometheus.HistogramVec {
	dbQueryDurationMu.Lock()
	defer dbQueryDurationMu.Unlock()
	if dbQueryDuration == nil {
		dbQueryDuration = databaseQueryDuration
	}
	return dbQueryDuration
}

func ObserveDBQuery(queryType string, duration time.Duration) {
	getOrCreateDBHistogram().WithLabelValues(queryType).Observe(duration.Seconds())
}

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func gzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		defer gz.Close()
		gzw := gzipResponseWriter{Writer: gz, ResponseWriter: w}
		next.ServeHTTP(gzw, r)
	})
}

func prometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		duration := time.Since(start)
		route := mux.CurrentRoute(r)
		path := ""
		if route != nil {
			path, _ = route.GetPathTemplate()
		}
		if path == "" {
			path = r.URL.Path
		}
		httpRequestsTotal.WithLabelValues(r.Method, path, strconv.Itoa(rec.status)).Inc()
		httpRequestDuration.WithLabelValues(r.Method, path).Observe(duration.Seconds())
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (rec *statusRecorder) WriteHeader(code int) {
	rec.status = code
	rec.ResponseWriter.WriteHeader(code)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func spaHandler(staticPath string, indexFile string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(staticPath, r.URL.Path)
		fi, err := os.Stat(path)
		if os.IsNotExist(err) || fi.IsDir() {
			http.ServeFile(w, r, filepath.Join(staticPath, indexFile))
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.ServeFile(w, r, path)
	}
}

func sensorDataCounterMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/api/data") {
			sensorDataReceived.Inc()
		}
	})
}

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

	go func() {
		for alert := range alertCh {
			alertsTriggered.WithLabelValues(
				strconv.Itoa(alert.SiteID),
				alert.AlertType,
				alert.Severity,
			).Inc()
		}
	}()

	r := mux.NewRouter()

	r.Use(corsMiddleware)
	r.Use(gzipMiddleware)
	r.Use(prometheusMiddleware)
	r.Use(sensorDataCounterMiddleware)

	apiRouter := r.PathPrefix("").Subrouter()
	sensorRcv.RegisterRoutes(apiRouter)
	predictor.RegisterRoutes(apiRouter)
	optimizer.RegisterRoutes(apiRouter)
	alarmHub.RegisterRoutes(apiRouter)

	r.Handle("/metrics", promhttp.Handler()).Methods("GET")

	pprofRouter := r.PathPrefix("/debug/pprof").Subrouter()
	pprofRouter.HandleFunc("/", pprof.Index)
	pprofRouter.HandleFunc("/cmdline", pprof.Cmdline)
	pprofRouter.HandleFunc("/profile", pprof.Profile)
	pprofRouter.HandleFunc("/symbol", pprof.Symbol)
	pprofRouter.HandleFunc("/trace", pprof.Trace)
	pprofRouter.HandleFunc("/heap", pprof.Handler("heap").ServeHTTP)
	pprofRouter.HandleFunc("/goroutine", pprof.Handler("goroutine").ServeHTTP)
	pprofRouter.HandleFunc("/threadcreate", pprof.Handler("threadcreate").ServeHTTP)
	pprofRouter.HandleFunc("/block", pprof.Handler("block").ServeHTTP)
	pprofRouter.HandleFunc("/mutex", pprof.Handler("mutex").ServeHTTP)

	staticPath := getEnv("STATIC_PATH", "../frontend")
	if _, err := os.Stat(staticPath); err == nil {
		log.Printf("Serving frontend static files from: %s", staticPath)
		r.PathPrefix("/").Handler(spaHandler(staticPath, "index.html"))
	} else {
		log.Printf("Frontend static files not found at %s, skipping static file serving", staticPath)
	}

	port := getEnv("PORT", "8080")
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Println("========================================")
	log.Println("石窟监测后端服务启动")
	log.Println("========================================")
	log.Printf("Server starting on port %s...", port)
	log.Printf("Config loaded from: %s", configPath)
	log.Printf("Modules: sensor_receiver | weathering_predictor | material_optimizer | alarm_websocket")
	log.Println("========================================")
	log.Println("API endpoints:")
	log.Println("  [sensor_receiver]")
	log.Println("    GET    /api/sites")
	log.Println("    GET    /api/sites/{id}")
	log.Println("    GET    /api/sites/{id}/sensors")
	log.Println("    GET    /api/sites/{id}/data")
	log.Println("    GET    /api/sites/{id}/weathering-rates")
	log.Println("    GET    /api/sensors/{id}/hourly")
	log.Println("    POST   /api/data")
	log.Println("  [weathering_predictor]")
	log.Println("    POST   /api/predict")
	log.Println("    POST   /api/predict/batch")
	log.Println("  [material_optimizer]")
	log.Println("    GET    /api/materials")
	log.Println("    POST   /api/topsis")
	log.Println("  [alarm_websocket]")
	log.Println("    WS     /ws/alerts")
	log.Println("    GET    /api/alerts")
	log.Println("    PUT    /api/alerts/{id}/acknowledge")
	log.Println("========================================")
	log.Println("Observability endpoints:")
	log.Println("    GET    /metrics                    (Prometheus metrics)")
	log.Println("    GET    /debug/pprof/               (pprof index)")
	log.Println("    GET    /debug/pprof/heap           (heap profile)")
	log.Println("    GET    /debug/pprof/goroutine      (goroutine profile)")
	log.Println("    GET    /debug/pprof/profile        (CPU profile)")
	log.Println("========================================")

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to start server: %v", err)
	}
}
