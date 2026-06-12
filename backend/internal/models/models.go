package models

import "time"

type Cave struct {
	ID          int       `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Location    string    `json:"location" db:"location"`
	Province    string    `json:"province" db:"province"`
	Latitude    float64   `json:"latitude" db:"latitude"`
	Longitude   float64   `json:"longitude" db:"longitude"`
	RockType    string    `json:"rockType" db:"rock_type"`
	Dynasty     string    `json:"dynasty" db:"dynasty"`
	Description string    `json:"description" db:"description"`
	CreatedAt   time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time `json:"updatedAt" db:"updated_at"`
}

type CaveStatistics struct {
	CaveID       int    `json:"caveId" db:"cave_id"`
	CaveName     string `json:"caveName" db:"cave_name"`
	Province     string `json:"province" db:"province"`
	RockType     string `json:"rockType" db:"rock_type"`
	TotalPoints  int    `json:"totalPoints" db:"total_points"`
	ActiveAlerts int    `json:"activeAlerts" db:"active_alerts"`
}

type MonitoringPoint struct {
	ID                 int       `json:"id" db:"id"`
	CaveID             int       `json:"caveId" db:"cave_id"`
	Name               string    `json:"name" db:"name"`
	Code               string    `json:"code" db:"code"`
	PositionX          float64   `json:"positionX" db:"position_x"`
	PositionY          float64   `json:"positionY" db:"position_y"`
	PositionZ          float64   `json:"positionZ" db:"position_z"`
	RockType           string    `json:"rockType" db:"rock_type"`
	InitialHardness    float64   `json:"initialHardness" db:"initial_hardness"`
	InitialCrackWidth  float64   `json:"initialCrackWidth" db:"initial_crack_width"`
	AreaDescription    string    `json:"areaDescription" db:"area_description"`
	Status             int       `json:"status" db:"status"`
	CreatedAt          time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt          time.Time `json:"updatedAt" db:"updated_at"`
}

type SensorData struct {
	Time              time.Time `json:"time" db:"time"`
	PointID           int       `json:"pointId" db:"point_id"`
	Temperature       float64   `json:"temperature" db:"temperature"`
	Humidity          float64   `json:"humidity" db:"humidity"`
	SurfaceHardness   float64   `json:"surfaceHardness" db:"surface_hardness"`
	CrackWidth        float64   `json:"crackWidth" db:"crack_width"`
	WindSpeed         float64   `json:"windSpeed,omitempty" db:"wind_speed"`
	Rainfall          float64   `json:"rainfall,omitempty" db:"rainfall"`
	SolarRadiation    float64   `json:"solarRadiation,omitempty" db:"solar_radiation"`
	CO2Concentration  float64   `json:"co2Concentration,omitempty" db:"co2_concentration"`
	Vibration         float64   `json:"vibration,omitempty" db:"vibration"`
}

type LatestSensorData struct {
	PointID         int       `json:"pointId" db:"point_id"`
	Time            time.Time `json:"time" db:"time"`
	Temperature     float64   `json:"temperature" db:"temperature"`
	Humidity        float64   `json:"humidity" db:"humidity"`
	SurfaceHardness float64   `json:"surfaceHardness" db:"surface_hardness"`
	CrackWidth      float64   `json:"crackWidth" db:"crack_width"`
}

type WeatheringRate struct {
	ID                     int       `json:"id" db:"id"`
	PointID                int       `json:"pointId" db:"point_id"`
	PeriodStart            string    `json:"periodStart" db:"period_start"`
	PeriodEnd              string    `json:"periodEnd" db:"period_end"`
	HardnessLossRate       float64   `json:"hardnessLossRate" db:"hardness_loss_rate"`
	CrackGrowthRate        float64   `json:"crackGrowthRate" db:"crack_growth_rate"`
	AvgTemperature         float64   `json:"avgTemperature" db:"avg_temperature"`
	AvgHumidity            float64   `json:"avgHumidity" db:"avg_humidity"`
	TemperatureFluctuation float64   `json:"temperatureFluctuation" db:"temperature_fluctuation"`
	RainfallTotal          float64   `json:"rainfallTotal" db:"rainfall_total"`
	OverallRate            float64   `json:"overallRate" db:"overall_rate"`
	RiskLevel              string    `json:"riskLevel" db:"risk_level"`
	CalculationDate        time.Time `json:"calculationDate" db:"calculation_date"`
}

type ProtectionMaterial struct {
	ID                       int       `json:"id" db:"id"`
	Name                     string    `json:"name" db:"name"`
	Category                 string    `json:"category" db:"category"`
	Manufacturer             string    `json:"manufacturer" db:"manufacturer"`
	WeatherResistance        float64   `json:"weatherResistance" db:"weather_resistance"`
	Permeability             float64   `json:"permeability" db:"permeability"`
	Adhesion                 float64   `json:"adhesion" db:"adhesion"`
	Reversibility            float64   `json:"reversibility" db:"reversibility"`
	Durability               float64   `json:"durability" db:"durability"`
	EnvironmentalFriendliness float64  `json:"environmentalFriendliness" db:"environmental_friendliness"`
	CostPerUnit              float64   `json:"costPerUnit" db:"cost_per_unit"`
	CoverageRate             float64   `json:"coverageRate" db:"coverage_rate"`
	ApplicableRockTypes      string    `json:"applicableRockTypes" db:"applicable_rock_types"`
	ConstructionDifficulty   int       `json:"constructionDifficulty" db:"construction_difficulty"`
	LifespanYears            float64   `json:"lifespanYears" db:"lifespan_years"`
	Description              string    `json:"description" db:"description"`
	Status                   int       `json:"status" db:"status"`
	CreatedAt                time.Time `json:"createdAt" db:"created_at"`
}

type MaterialScore struct {
	ID            int     `json:"id"`
	Name          string  `json:"name"`
	Category      string  `json:"category"`
	TOPSISScore   float64 `json:"topsisScore"`
	NormalizedCost float64 `json:"normalizedCost"`
	Rank          int     `json:"rank"`
}

type Alert struct {
	ID             int       `json:"id" db:"id"`
	PointID        int       `json:"pointId" db:"point_id"`
	AlertType      string    `json:"alertType" db:"alert_type"`
	Severity       string    `json:"severity" db:"severity"`
	Title          string    `json:"title" db:"title"`
	Message        string    `json:"message" db:"message"`
	CurrentValue   float64   `json:"currentValue" db:"current_value"`
	ThresholdValue float64   `json:"thresholdValue" db:"threshold_value"`
	SensorDataTime time.Time `json:"sensorDataTime" db:"sensor_data_time"`
	IsAcknowledged bool      `json:"isAcknowledged" db:"is_acknowledged"`
	AcknowledgedAt *time.Time `json:"acknowledgedAt,omitempty" db:"acknowledged_at"`
	AcknowledgedBy *string    `json:"acknowledgedBy,omitempty" db:"acknowledged_by"`
	CreatedAt      time.Time `json:"createdAt" db:"created_at"`
}

type WeatheringPrediction struct {
	ID                    int       `json:"id" db:"id"`
	PointID               int       `json:"pointId" db:"point_id"`
	ModelVersion          string    `json:"modelVersion" db:"model_version"`
	InputTemperature      float64   `json:"inputTemperature" db:"input_temperature"`
	InputHumidity         float64   `json:"inputHumidity" db:"input_humidity"`
	InputTemperatureRange float64   `json:"inputTemperatureRange,omitempty" db:"input_temperature_range"`
	PredictedRate         float64   `json:"predictedRate" db:"predicted_rate"`
	PredictionDate        time.Time `json:"predictionDate" db:"prediction_date"`
	ConfidenceLevel       float64   `json:"confidenceLevel" db:"confidence_level"`
}

type PredictionRequest struct {
	PointID               int      `json:"pointId" binding:"required"`
	TargetTemperature     *float64 `json:"targetTemperature"`
	TargetHumidity        *float64 `json:"targetHumidity"`
	TemperatureRange      *float64 `json:"temperatureRange"`
	RockType              string   `json:"rockType"`
}

type MaterialRecommendationRequest struct {
	CaveID                 *int     `json:"caveId"`
	PointID                *int     `json:"pointId"`
	RockType               string   `json:"rockType" binding:"required"`
	ClimateZone            string   `json:"climateZone"`
	BudgetLevel            string   `json:"budgetLevel"` // LOW, MEDIUM, HIGH
	PriorityCriteria       []string `json:"priorityCriteria"`
	ProtectionType         string   `json:"protectionType"` // SURFACE, REINFORCEMENT, WATERPROOF
}

type ApiResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Total   int         `json:"total,omitempty"`
}

type WebSocketMessage struct {
	Type    string      `json:"type"` // ALERT, DATA, STATUS
	Payload interface{} `json:"payload"`
	Time    time.Time   `json:"time"`
}

type Pagination struct {
	Page     int `form:"page" binding:"min=1"`
	PageSize int `form:"pageSize" binding:"min=1,max=100"`
}

func (p *Pagination) Default() {
	if p.Page == 0 {
		p.Page = 1
	}
	if p.PageSize == 0 {
		p.PageSize = 20
	}
}

func (p *Pagination) Offset() int {
	return (p.Page - 1) * p.PageSize
}

func SuccessResponse(data interface{}) ApiResponse {
	return ApiResponse{
		Code:    0,
		Message: "success",
		Data:    data,
	}
}

func SuccessResponseWithTotal(data interface{}, total int) ApiResponse {
	return ApiResponse{
		Code:    0,
		Message: "success",
		Data:    data,
		Total:   total,
	}
}

func ErrorResponse(code int, message string) ApiResponse {
	return ApiResponse{
		Code:    code,
		Message: message,
	}
}
