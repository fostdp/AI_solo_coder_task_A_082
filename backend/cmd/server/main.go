package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"grotto-monitor/internal/config"
	"grotto-monitor/internal/database"
	"grotto-monitor/internal/handlers"
	"grotto-monitor/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()

	log.Println("========================================")
	log.Println("  石窟风化监测系统 - Go后端服务")
	log.Println("========================================")

	if err := database.Init(cfg.Database); err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}
	defer database.Close()

	h := handlers.NewHandler(cfg)

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	r.Use(middleware.CORS())
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	setupRoutes(r, h)

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      r,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}

	go func() {
		log.Printf("✓ 服务启动中，监听端口: %s", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("服务启动失败: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("\n正在关闭服务...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("服务强制关闭: %v", err)
	}

	log.Println("✓ 服务已安全关闭")
}

func setupRoutes(r *gin.Engine, h *handlers.Handler) {
	api := r.Group("/api/v1")
	{
		api.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status":  "ok",
				"service": "grotto-monitor-api",
				"time":    time.Now().Format(time.RFC3339),
			})
		})

		api.GET("/dashboard/overview", h.GetDashboardOverview)

		caves := api.Group("/caves")
		{
			caves.GET("", h.GetCaves)
			caves.GET("/statistics", h.GetCaveStatistics)
			caves.GET("/:id", h.GetCave)
		}

		points := api.Group("/points")
		{
			points.GET("", h.GetMonitoringPoints)
			points.GET("/:id", h.GetMonitoringPoint)
		}

		sensors := api.Group("/sensors")
		{
			sensors.GET("/latest", h.GetSensorLatest)
			sensors.GET("/history", h.GetSensorHistory)
			sensors.POST("/data", h.PostSensorData)
			sensors.POST("/batch", h.PostBatchSensorData)
		}

		weathering := api.Group("/weathering")
		{
			weathering.GET("/rates", h.GetWeatheringRates)
			weathering.POST("/predict", h.PredictWeathering)
		}

		materials := api.Group("/materials")
		{
			materials.GET("", h.GetMaterials)
			materials.POST("/recommend", h.RecommendMaterials)
		}

		alerts := api.Group("/alerts")
		{
			alerts.GET("", h.GetAlerts)
			alerts.POST("/:id/acknowledge", h.AcknowledgeAlert)
		}

		ws := api.Group("/ws")
		{
			ws.GET("", h.WebSocketEndpoint)
			ws.GET("/status", h.WSStatus)
			ws.GET("/offline-messages", h.GetOfflineMessages)
		}
	}

	log.Println("✓ API路由注册完成")
}
