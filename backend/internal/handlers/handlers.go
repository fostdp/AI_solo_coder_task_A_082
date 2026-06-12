package handlers

import (
	"context"
	"encoding/json"
	"log"
	"math"
	"net/http"
	"strconv"
	"time"

	"grotto-monitor/internal/config"
	"grotto-monitor/internal/database"
	"grotto-monitor/internal/models"
	"grotto-monitor/internal/services"
	"grotto-monitor/internal/websocket"
	"grotto-monitor/pkg/algorithms"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	cfg              *config.Config
	caveSvc          *services.CaveService
	pointSvc         *services.MonitoringPointService
	sensorSvc        *services.SensorDataService
	alertSvc         *services.AlertService
	wsManager        *websocket.Manager
	weatheringPred   *algorithms.WeatheringPredictor
	topsis           *algorithms.TOPSIS
}

func NewHandler(cfg *config.Config) *Handler {
	return &Handler{
		cfg:            cfg,
		caveSvc:        services.NewCaveService(),
		pointSvc:       services.NewMonitoringPointService(),
		sensorSvc:      services.NewSensorDataService(),
		alertSvc:       services.NewAlertService(),
		wsManager:      websocket.GetManager(),
		weatheringPred: algorithms.NewWeatheringPredictor(),
		topsis:         algorithms.NewTOPSIS(),
	}
}

func (h *Handler) respondJSON(c *gin.Context, status int, resp models.ApiResponse) {
	c.JSON(status, resp)
}

func (h *Handler) GetCaves(c *gin.Context) {
	caves, err := h.caveSvc.GetAll(c.Request.Context())
	if err != nil {
		h.respondJSON(c, http.StatusInternalServerError, models.ErrorResponse(500, "获取石窟列表失败: "+err.Error()))
		return
	}
	h.respondJSON(c, http.StatusOK, models.SuccessResponse(caves))
}

func (h *Handler) GetCave(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		h.respondJSON(c, http.StatusBadRequest, models.ErrorResponse(400, "无效的石窟ID"))
		return
	}
	cave, err := h.caveSvc.GetByID(c.Request.Context(), id)
	if err != nil {
		h.respondJSON(c, http.StatusNotFound, models.ErrorResponse(404, "石窟不存在"))
		return
	}
	h.respondJSON(c, http.StatusOK, models.SuccessResponse(cave))
}

func (h *Handler) GetCaveStatistics(c *gin.Context) {
	stats, err := h.caveSvc.GetStatistics(c.Request.Context())
	if err != nil {
		h.respondJSON(c, http.StatusInternalServerError, models.ErrorResponse(500, err.Error()))
		return
	}
	h.respondJSON(c, http.StatusOK, models.SuccessResponse(stats))
}

func (h *Handler) GetMonitoringPoints(c *gin.Context) {
	caveID, err := strconv.Atoi(c.Query("caveId"))
	if err != nil || caveID <= 0 {
		h.respondJSON(c, http.StatusBadRequest, models.ErrorResponse(400, "请提供有效的caveId"))
		return
	}
	points, err := h.pointSvc.GetByCaveID(c.Request.Context(), caveID)
	if err != nil {
		h.respondJSON(c, http.StatusInternalServerError, models.ErrorResponse(500, err.Error()))
		return
	}
	h.respondJSON(c, http.StatusOK, models.SuccessResponse(points))
}

func (h *Handler) GetMonitoringPoint(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		h.respondJSON(c, http.StatusBadRequest, models.ErrorResponse(400, "无效的监测点ID"))
		return
	}
	point, err := h.pointSvc.GetByID(c.Request.Context(), id)
	if err != nil {
		h.respondJSON(c, http.StatusNotFound, models.ErrorResponse(404, "监测点不存在"))
		return
	}
	h.respondJSON(c, http.StatusOK, models.SuccessResponse(point))
}

func (h *Handler) PostSensorData(c *gin.Context) {
	var data models.SensorData
	if err := c.ShouldBindJSON(&data); err != nil {
		h.respondJSON(c, http.StatusBadRequest, models.ErrorResponse(400, "数据格式错误: "+err.Error()))
		return
	}
	if data.PointID <= 0 {
		h.respondJSON(c, http.StatusBadRequest, models.ErrorResponse(400, "pointId无效"))
		return
	}
	if data.Time.IsZero() {
		data.Time = time.Now()
	}

	ctx := c.Request.Context()

	if err := h.sensorSvc.Insert(ctx, &data); err != nil {
		h.respondJSON(c, http.StatusInternalServerError, models.ErrorResponse(500, "数据写入失败: "+err.Error()))
		return
	}

	go h.checkAndTriggerAlerts(ctx, &data)

	latestData := map[string]interface{}{
		"pointId":        data.PointID,
		"time":           data.Time,
		"temperature":    data.Temperature,
		"humidity":       data.Humidity,
		"surfaceHardness": data.SurfaceHardness,
		"crackWidth":     data.CrackWidth,
	}
	h.wsManager.Broadcast("SENSOR_DATA", latestData)

	h.respondJSON(c, http.StatusOK, models.SuccessResponse(map[string]interface{}{
		"received": true,
		"time":     data.Time,
	}))
}

func (h *Handler) PostBatchSensorData(c *gin.Context) {
	var batch []models.SensorData
	if err := c.ShouldBindJSON(&batch); err != nil {
		h.respondJSON(c, http.StatusBadRequest, models.ErrorResponse(400, "数据格式错误: "+err.Error()))
		return
	}

	ctx := c.Request.Context()
	successCount := 0
	for i := range batch {
		if batch[i].Time.IsZero() {
			batch[i].Time = time.Now()
		}
		if err := h.sensorSvc.Insert(ctx, &batch[i]); err == nil {
			successCount++
			go h.checkAndTriggerAlerts(ctx, &batch[i])
		}
	}

	h.respondJSON(c, http.StatusOK, models.SuccessResponse(map[string]interface{}{
		"total":   len(batch),
		"success": successCount,
	}))
}

func (h *Handler) checkAndTriggerAlerts(ctx context.Context, data *models.SensorData) {
	point, err := h.pointSvc.GetByID(ctx, data.PointID)
	if err != nil {
		log.Printf("获取监测点失败: %v", err)
		return
	}

	crackThreshold := h.cfg.Alert.CrackThresholdMM
	hardnessThreshold := h.cfg.Alert.HardnessDropPercent

	if data.CrackWidth > crackThreshold {
		alertType := "CRACK_EXCESSIVE"
		exists, _ := h.alertSvc.CheckForExcessiveAlert(ctx, data.PointID, alertType, 12)
		if !exists {
			severity := "CRITICAL"
			if data.CrackWidth > crackThreshold*2 {
				severity = "CRITICAL"
			} else if data.CrackWidth > crackThreshold*1.5 {
				severity = "WARNING"
			}

			alert := &models.Alert{
				PointID:        data.PointID,
				AlertType:      alertType,
				Severity:       severity,
				Title:          "裂隙宽度超限告警",
				Message:        point.Name + " 监测点裂隙宽度达到 " + strconv.FormatFloat(data.CrackWidth, 'f', 4, 64) + "mm，超过阈值 " + strconv.FormatFloat(crackThreshold, 'f', 2, 64) + "mm，请立即检查！",
				CurrentValue:   data.CrackWidth,
				ThresholdValue: crackThreshold,
				SensorDataTime: data.Time,
			}
			if err := h.alertSvc.Insert(ctx, alert); err == nil {
				h.wsManager.BroadcastAlert(alert)
				log.Printf("告警触发 [裂隙超限]: %s - %s", point.Name, alert.Message)
			}
		}
	}

	if point.InitialHardness > 0 {
		hardnessDrop := (point.InitialHardness - data.SurfaceHardness) / point.InitialHardness * 100
		if hardnessDrop >= hardnessThreshold {
			alertType := "HARDNESS_DROP"
			exists, _ := h.alertSvc.CheckForExcessiveAlert(ctx, data.PointID, alertType, 12)
			if !exists {
				severity := "WARNING"
				if hardnessDrop > hardnessThreshold*2 {
					severity = "CRITICAL"
				}

				alert := &models.Alert{
					PointID:        data.PointID,
					AlertType:      alertType,
					Severity:       severity,
					Title:          "表面硬度下降告警",
					Message:        point.Name + " 监测点表面硬度已下降 " + strconv.FormatFloat(hardnessDrop, 'f', 2, 64) + "%，超过阈值 " + strconv.FormatFloat(hardnessThreshold, 'f', 1, 64) + "%，风化严重！",
					CurrentValue:   hardnessDrop,
					ThresholdValue: hardnessThreshold,
					SensorDataTime: data.Time,
				}
				if err := h.alertSvc.Insert(ctx, alert); err == nil {
					h.wsManager.BroadcastAlert(alert)
					log.Printf("告警触发 [硬度下降]: %s - %s", point.Name, alert.Message)
				}
			}
		}
	}

	if data.Humidity > 90 || data.Humidity < 20 {
		alertType := "EXTREME_HUMIDITY"
		exists, _ := h.alertSvc.CheckForExcessiveAlert(ctx, data.PointID, alertType, 24)
		if !exists {
			severity := "INFO"
			title := "湿度异常"
			msg := point.Name + " 湿度异常: " + strconv.FormatFloat(data.Humidity, 'f', 1, 64) + "%，"
			if data.Humidity > 90 {
				title = "高湿度告警"
				msg += "高湿环境可能加速化学风化"
				severity = "WARNING"
			} else {
				title = "低湿度告警"
				msg += "干燥环境可能导致表面粉化"
			}
			alert := &models.Alert{
				PointID:        data.PointID,
				AlertType:      alertType,
				Severity:       severity,
				Title:          title,
				Message:        msg,
				CurrentValue:   data.Humidity,
				ThresholdValue: 85,
				SensorDataTime: data.Time,
			}
			if err := h.alertSvc.Insert(ctx, alert); err == nil {
				h.wsManager.BroadcastAlert(alert)
			}
		}
	}
}

func (h *Handler) GetSensorLatest(c *gin.Context) {
	pointID, err := strconv.Atoi(c.Query("pointId"))
	if err != nil || pointID <= 0 {
		allLatest, err := h.sensorSvc.GetAllLatest(c.Request.Context())
		if err != nil {
			h.respondJSON(c, http.StatusInternalServerError, models.ErrorResponse(500, err.Error()))
			return
		}
		h.respondJSON(c, http.StatusOK, models.SuccessResponse(allLatest))
		return
	}

	data, err := h.sensorSvc.GetLatestByPoint(c.Request.Context(), pointID)
	if err != nil {
		h.respondJSON(c, http.StatusInternalServerError, models.ErrorResponse(500, err.Error()))
		return
	}
	h.respondJSON(c, http.StatusOK, models.SuccessResponse(data))
}

func (h *Handler) GetSensorHistory(c *gin.Context) {
	pointID, err := strconv.Atoi(c.Query("pointId"))
	if err != nil || pointID <= 0 {
		h.respondJSON(c, http.StatusBadRequest, models.ErrorResponse(400, "请提供有效的pointId"))
		return
	}

	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	if days <= 0 {
		days = 30
	}
	if days > 730 {
		days = 730
	}

	end := time.Now()
	start := end.AddDate(0, 0, -days)

	data, err := h.sensorSvc.GetHistoryByPoint(c.Request.Context(), pointID, start, end)
	if err != nil {
		h.respondJSON(c, http.StatusInternalServerError, models.ErrorResponse(500, err.Error()))
		return
	}

	h.respondJSON(c, http.StatusOK, models.SuccessResponse(data))
}

func (h *Handler) GetAlerts(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	ackStr := c.Query("acknowledged")

	var ackFilter *bool
	if ackStr != "" {
		b, _ := strconv.ParseBool(ackStr)
		ackFilter = &b
	}

	alerts, err := h.alertSvc.GetRecent(c.Request.Context(), limit, ackFilter)
	if err != nil {
		h.respondJSON(c, http.StatusInternalServerError, models.ErrorResponse(500, err.Error()))
		return
	}
	h.respondJSON(c, http.StatusOK, models.SuccessResponse(alerts))
}

func (h *Handler) AcknowledgeAlert(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		h.respondJSON(c, http.StatusBadRequest, models.ErrorResponse(400, "无效的告警ID"))
		return
	}
	user := c.DefaultPostForm("user", "system")
	if err := h.alertSvc.Acknowledge(c.Request.Context(), id, user); err != nil {
		h.respondJSON(c, http.StatusInternalServerError, models.ErrorResponse(500, err.Error()))
		return
	}
	h.respondJSON(c, http.StatusOK, models.SuccessResponse(map[string]interface{}{"acknowledged": true}))
}

func (h *Handler) GetWeatheringRates(c *gin.Context) {
	pointID, err := strconv.Atoi(c.Query("pointId"))
	if err != nil || pointID <= 0 {
		h.respondJSON(c, http.StatusBadRequest, models.ErrorResponse(400, "请提供有效的pointId"))
		return
	}

	rows, err := database.DB.Query(c.Request.Context(), `
		SELECT id, point_id, period_start, period_end, hardness_loss_rate, crack_growth_rate,
		       avg_temperature, avg_humidity, temperature_fluctuation, rainfall_total,
		       overall_rate, risk_level, calculation_date
		FROM weathering_rates WHERE point_id = $1 ORDER BY period_start DESC LIMIT 12
	`, pointID)
	if err != nil {
		h.respondJSON(c, http.StatusInternalServerError, models.ErrorResponse(500, err.Error()))
		return
	}
	defer rows.Close()

	var rates []models.WeatheringRate
	for rows.Next() {
		var r models.WeatheringRate
		err := rows.Scan(&r.ID, &r.PointID, &r.PeriodStart, &r.PeriodEnd, &r.HardnessLossRate,
			&r.CrackGrowthRate, &r.AvgTemperature, &r.AvgHumidity, &r.TemperatureFluctuation,
			&r.RainfallTotal, &r.OverallRate, &r.RiskLevel, &r.CalculationDate)
		if err != nil {
			continue
		}
		rates = append(rates, r)
	}

	if len(rates) == 0 {
		rates = h.calculateWeatheringRates(c.Request.Context(), pointID)
	}

	h.respondJSON(c, http.StatusOK, models.SuccessResponse(rates))
}

func (h *Handler) calculateWeatheringRates(ctx context.Context, pointID int) []models.WeatheringRate {
	rows, err := database.DB.Query(ctx, `
		SELECT time_bucket('30 days', time) AS bucket,
		       FIRST(surface_hardness, time) AS first_hardness,
		       LAST(surface_hardness, time) AS last_hardness,
		       FIRST(crack_width, time) AS first_crack,
		       LAST(crack_width, time) AS last_crack,
		       AVG(temperature) AS avg_temp,
		       MAX(temperature) - MIN(temperature) AS temp_fluct,
		       AVG(humidity) AS avg_hum,
		       SUM(rainfall) AS total_rain
		FROM sensor_data
		WHERE point_id = $1
		GROUP BY time_bucket('30 days', time)
		ORDER BY bucket DESC
		LIMIT 12
	`, pointID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var rates []models.WeatheringRate
	for rows.Next() {
		var (
			bucket, periodStart, periodEnd time.Time
			firstHard, lastHard            float64
			firstCrack, lastCrack          float64
			avgTemp, tempFluct             float64
			avgHum, totalRain              float64
		)
		err := rows.Scan(&bucket, &firstHard, &lastHard, &firstCrack, &lastCrack,
			&avgTemp, &tempFluct, &avgHum, &totalRain)
		if err != nil {
			continue
		}

		periodStart = bucket
		periodEnd = bucket.AddDate(0, 0, 30)

		hardnessLossRate := 0.0
		if firstHard > 0 {
			hardnessLossRate = (firstHard - lastHard) / firstHard / 30
		}

		crackGrowthRate := 0.0
		if lastCrack > firstCrack {
			crackGrowthRate = (lastCrack - firstCrack) / 30
		}

		overallRate := hardnessLossRate*0.5 + crackGrowthRate*50.0
		riskLevel := "LOW"
		switch {
		case overallRate > 0.01:
			riskLevel = "CRITICAL"
		case overallRate > 0.005:
			riskLevel = "HIGH"
		case overallRate > 0.002:
			riskLevel = "MEDIUM"
		}

		rates = append(rates, models.WeatheringRate{
			PointID:                pointID,
			PeriodStart:            periodStart.Format("2006-01-02"),
			PeriodEnd:              periodEnd.Format("2006-01-02"),
			HardnessLossRate:       round(hardnessLossRate, 8),
			CrackGrowthRate:        round(crackGrowthRate, 8),
			AvgTemperature:         round(avgTemp, 2),
			AvgHumidity:            round(avgHum, 2),
			TemperatureFluctuation: round(tempFluct, 2),
			RainfallTotal:          round(totalRain, 2),
			OverallRate:            round(overallRate, 8),
			RiskLevel:              riskLevel,
		})
	}
	return rates
}

func (h *Handler) PredictWeathering(c *gin.Context) {
	var req models.PredictionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondJSON(c, http.StatusBadRequest, models.ErrorResponse(400, "参数错误: "+err.Error()))
		return
	}

	ctx := c.Request.Context()
	point, err := h.pointSvc.GetByID(ctx, req.PointID)
	if err != nil {
		h.respondJSON(c, http.StatusNotFound, models.ErrorResponse(404, "监测点不存在"))
		return
	}

	rockType := req.RockType
	if rockType == "" {
		rockType = point.RockType
	}

	targetTemp := req.TargetTemperature
	targetHum := req.TargetHumidity
	tempRange := req.TemperatureRange

	if targetTemp == nil || targetHum == nil {
		latest, err := h.sensorSvc.GetLatestByPoint(ctx, req.PointID)
		if err == nil {
			if targetTemp == nil {
				t := latest.Temperature
				targetTemp = &t
			}
			if targetHum == nil {
				h := latest.Humidity
				targetHum = &h
			}
		}
		if tempRange == nil {
			r := 10.0
			tempRange = &r
		}
	}

	historicalAvgRainfall := 50.0
	rainfallRows, err := database.DB.Query(ctx, `
		SELECT COALESCE(SUM(rainfall)/30.0, 50) FROM sensor_data
		WHERE point_id = $1 AND time > NOW() - INTERVAL '90 days'
	`, req.PointID)
	if err == nil && rainfallRows.Next() {
		rainfallRows.Scan(&historicalAvgRainfall)
		rainfallRows.Close()
	}

	predRate, confidence := h.weatheringPred.Predict(rockType, *targetTemp, *targetHum, *tempRange, historicalAvgRainfall)

	currentRate := h.calculateCurrentWeatheringRate(ctx, req.PointID)

	suggestions := algorithms.GenerateProtectionSuggestions(predRate, currentRate, *targetTemp, *targetHum, rockType)

	prediction := &models.WeatheringPrediction{
		PointID:               req.PointID,
		ModelVersion:          "RF-v1.0",
		InputTemperature:      *targetTemp,
		InputHumidity:         *targetHum,
		InputTemperatureRange: *tempRange,
		PredictedRate:         predRate,
		ConfidenceLevel:       confidence * 100,
		PredictionDate:        time.Now(),
	}

	database.DB.Exec(ctx, `
		INSERT INTO weathering_predictions (point_id, model_version, input_temperature, input_humidity,
			input_temperature_range, predicted_rate, prediction_date, confidence_level)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, prediction.PointID, prediction.ModelVersion, prediction.InputTemperature,
		prediction.InputHumidity, prediction.InputTemperatureRange, prediction.PredictedRate,
		prediction.PredictionDate, prediction.ConfidenceLevel)

	h.respondJSON(c, http.StatusOK, models.SuccessResponse(map[string]interface{}{
		"prediction":     prediction,
		"currentRate":    currentRate,
		"riskMultiplier": round(predRate/math.Max(0.0001, currentRate), 4),
		"suggestions":    suggestions,
	}))
}

func (h *Handler) calculateCurrentWeatheringRate(ctx context.Context, pointID int) float64 {
	var firstHard, lastHard, firstCrack, lastCrack float64
	var cnt int
	row := database.DB.QueryRow(ctx, `
		SELECT
			FIRST(surface_hardness, time) as fh,
			LAST(surface_hardness, time) as lh,
			FIRST(crack_width, time) as fc,
			LAST(crack_width, time) as lc,
			COUNT(*) as cnt
		FROM sensor_data
		WHERE point_id = $1 AND time > NOW() - INTERVAL '90 days'
	`, pointID)
	err := row.Scan(&firstHard, &lastHard, &firstCrack, &lastCrack, &cnt)
	if err != nil || cnt < 2 || firstHard == 0 {
		return 0.005
	}
	days := 90.0
	hardnessLossRate := 0.0
	if firstHard > 0 {
		hardnessLossRate = (firstHard - lastHard) / firstHard / days
	}
	crackGrowthRate := 0.0
	if lastCrack > firstCrack {
		crackGrowthRate = (lastCrack - firstCrack) / days
	}
	return round(hardnessLossRate*0.5+crackGrowthRate*50.0, 8)
}

func (h *Handler) GetMaterials(c *gin.Context) {
	rows, err := database.DB.Query(c.Request.Context(), `
		SELECT id, name, category, manufacturer, weather_resistance, permeability, adhesion,
		       reversibility, durability, environmental_friendliness, cost_per_unit, coverage_rate,
		       applicable_rock_types, construction_difficulty, lifespan_years, description, status
		FROM protection_materials WHERE status = 1 ORDER BY id
	`)
	if err != nil {
		h.respondJSON(c, http.StatusInternalServerError, models.ErrorResponse(500, err.Error()))
		return
	}
	defer rows.Close()

	var materials []models.ProtectionMaterial
	for rows.Next() {
		var m models.ProtectionMaterial
		err := rows.Scan(&m.ID, &m.Name, &m.Category, &m.Manufacturer,
			&m.WeatherResistance, &m.Permeability, &m.Adhesion, &m.Reversibility,
			&m.Durability, &m.EnvironmentalFriendliness, &m.CostPerUnit, &m.CoverageRate,
			&m.ApplicableRockTypes, &m.ConstructionDifficulty, &m.LifespanYears, &m.Description, &m.Status)
		if err != nil {
			continue
		}
		materials = append(materials, m)
	}

	h.respondJSON(c, http.StatusOK, models.SuccessResponse(materials))
}

func (h *Handler) RecommendMaterials(c *gin.Context) {
	var req models.MaterialRecommendationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondJSON(c, http.StatusBadRequest, models.ErrorResponse(400, "参数错误: "+err.Error()))
		return
	}

	rows, err := database.DB.Query(c.Request.Context(), `
		SELECT id, name, category, weather_resistance, permeability, adhesion,
		       reversibility, durability, environmental_friendliness, cost_per_unit, coverage_rate,
		       applicable_rock_types, construction_difficulty, lifespan_years, description
		FROM protection_materials WHERE status = 1
	`)
	if err != nil {
		h.respondJSON(c, http.StatusInternalServerError, models.ErrorResponse(500, err.Error()))
		return
	}
	defer rows.Close()

	var candidates []algorithms.MaterialCandidate
	for rows.Next() {
		var c algorithms.MaterialCandidate
		err := rows.Scan(&c.ID, &c.Name, &c.Category,
			&c.WeatherResistance, &c.Permeability, &c.Adhesion, &c.Reversibility,
			&c.Durability, &c.EnvironmentalFriendliness, &c.CostPerUnit, &c.CoverageRate,
			&c.ApplicableRockTypes, &c.ConstructionDifficulty, &c.LifespanYears, &c.Description)
		if err != nil {
			continue
		}
		candidates = append(candidates, c)
	}

	h.topsis.ApplyPriorityCriteria(req.PriorityCriteria)
	results := h.topsis.Evaluate(candidates, req.RockType, req.BudgetLevel, req.ProtectionType)

	recommendations := algorithms.GenerateMaterialRecommendations(results)

	paramsJSON, _ := json.Marshal(req)
	resultsJSON, _ := json.Marshal(results)

	caveID := 0
	if req.CaveID != nil {
		caveID = *req.CaveID
	}
	pointID := (*int)(nil)
	if req.PointID != nil {
		pointID = req.PointID
	}

	database.DB.Exec(c.Request.Context(), `
		INSERT INTO material_recommendations (cave_id, point_id, request_params, results)
		VALUES ($1, $2, $3, $4)
	`, caveID, pointID, paramsJSON, resultsJSON)

	h.respondJSON(c, http.StatusOK, models.SuccessResponse(map[string]interface{}{
		"results":         results,
		"recommendations": recommendations,
		"rockType":        req.RockType,
		"budgetLevel":     req.BudgetLevel,
		"protectionType":  req.ProtectionType,
		"weights":         h.topsis.Weights,
	}))
}

func (h *Handler) WebSocketEndpoint(c *gin.Context) {
	h.wsManager.HandleConnection(c)
}

func (h *Handler) WSStatus(c *gin.Context) {
	h.respondJSON(c, http.StatusOK, models.SuccessResponse(map[string]interface{}{
		"connectedClients": h.wsManager.ClientCount(),
	}))
}

func (h *Handler) GetDashboardOverview(c *gin.Context) {
	ctx := c.Request.Context()
	var totalCaves, totalPoints, totalActiveAlerts int
	var latestDataTime time.Time

	database.DB.QueryRow(ctx, "SELECT COUNT(*) FROM caves").Scan(&totalCaves)
	database.DB.QueryRow(ctx, "SELECT COUNT(*) FROM monitoring_points WHERE status = 1").Scan(&totalPoints)
	database.DB.QueryRow(ctx, "SELECT COUNT(*) FROM alerts WHERE is_acknowledged = FALSE").Scan(&totalActiveAlerts)
	database.DB.QueryRow(ctx, "SELECT COALESCE(MAX(time), NOW()) FROM sensor_data").Scan(&latestDataTime)

	recentAlerts, _ := h.alertSvc.GetRecent(ctx, 5, nil)
	stats, _ := h.caveSvc.GetStatistics(ctx)

	h.respondJSON(c, http.StatusOK, models.SuccessResponse(map[string]interface{}{
		"totalCaves":        totalCaves,
		"totalPoints":       totalPoints,
		"totalActiveAlerts": totalActiveAlerts,
		"latestDataTime":    latestDataTime,
		"recentAlerts":      recentAlerts,
		"caveStats":         stats,
	}))
}

func round(val float64, precision int) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}
