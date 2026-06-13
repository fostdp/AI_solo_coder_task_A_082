package models

import "time"

type GrottoSite struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Location    string    `json:"location"`
	RockType    string    `json:"rock_type"`
	CreatedAt   time.Time `json:"created_at"`
	Description string    `json:"description"`
	Latitude    float64   `json:"latitude"`
	Longitude   float64   `json:"longitude"`
}

type Sensor struct {
	ID            int       `json:"id"`
	SiteID        int       `json:"site_id"`
	SensorCode    string    `json:"sensor_code"`
	SensorType    string    `json:"sensor_type"`
	Type          string    `json:"type"`
	PositionX     float64   `json:"position_x"`
	PositionY     float64   `json:"position_y"`
	PositionZ     float64   `json:"position_z"`
	InstalledAt   time.Time `json:"installed_at"`
	Status        string    `json:"status"`
	BaselineValue float64   `json:"baseline_value"`
}

type MonitoringData struct {
	Time     time.Time `json:"time"`
	SensorID int       `json:"sensor_id"`
	SiteID   int       `json:"site_id"`
	Value    float64   `json:"value"`
}

type MonitoringDataInput struct {
	Time     time.Time `json:"time"`
	SensorID int       `json:"sensor_id"`
	SiteID   int       `json:"site_id"`
	Value    float64   `json:"value"`
}

type MonitoringDataInsert struct {
	Time     time.Time `json:"time"`
	SensorID int       `json:"sensor_id"`
	SiteID   int       `json:"site_id"`
	Value    float64   `json:"value"`
}

type ProtectionMaterial struct {
	ID                   int     `json:"id"`
	Name                 string  `json:"name"`
	Category             string  `json:"category"`
	PenetrationDepth     float64 `json:"penetration_depth"`
	Breathability        float64 `json:"breathability"`
	WeatheringResistance float64 `json:"weathering_resistance"`
	Compatibility        float64 `json:"compatibility"`
	Cost                 float64 `json:"cost"`
	DurabilityYears      float64 `json:"durability_years"`
	Description          string  `json:"description"`
}

type Alert struct {
	ID           int       `json:"id"`
	SiteID       int       `json:"site_id"`
	SensorID     int       `json:"sensor_id"`
	AlertType    string    `json:"alert_type"`
	Severity     string    `json:"severity"`
	Message      string    `json:"message"`
	Value        float64   `json:"value"`
	Threshold    float64   `json:"threshold"`
	TriggeredAt  time.Time `json:"triggered_at"`
	Acknowledged bool      `json:"acknowledged"`
}

type AlertFilter struct {
	SiteID       *int  `json:"site_id"`
	Acknowledged *bool `json:"acknowledged"`
}

type MonitoringHourly struct {
	Bucket   time.Time `json:"bucket"`
	SensorID int       `json:"sensor_id"`
	SiteID   int       `json:"site_id"`
	AvgValue float64   `json:"avg_value"`
	MinValue float64   `json:"min_value"`
	MaxValue float64   `json:"max_value"`
}

type WeatheringPredictionRequest struct {
	SiteID      int     `json:"site_id"`
	Temperature float64 `json:"temperature"`
	Humidity    float64 `json:"humidity"`
}

type PredictionRequest struct {
	SiteID      int     `json:"site_id"`
	Temperature float64 `json:"temperature"`
	Humidity    float64 `json:"humidity"`
}

type WeatheringPredictionResponse struct {
	PredictedRate  float64 `json:"predicted_rate"`
	RiskLevel      string  `json:"risk_level"`
	Recommendation string  `json:"recommendation"`
	Confidence     float64 `json:"confidence"`
}

type PredictionResult struct {
	PredictedRate  float64 `json:"predicted_rate"`
	RiskLevel      string  `json:"risk_level"`
	Recommendation string  `json:"recommendation"`
	Confidence     float64 `json:"confidence"`
}

type BatchPredictionRequest struct {
	SiteID       int                          `json:"site_id"`
	Combinations []WeatheringPredictionRequest `json:"combinations"`
}

type BatchPredictionResponse struct {
	Predictions []WeatheringPredictionResponse `json:"predictions"`
}

type TOPSISRequest struct {
	SiteID     int                `json:"site_id"`
	RockType   string             `json:"rock_type"`
	Priorities map[string]float64 `json:"priorities"`
}

type TOPSISResult struct {
	MaterialID   int     `json:"material_id"`
	MaterialName string  `json:"material_name"`
	TOPSISScore  float64 `json:"topsis_score"`
	Rank         int     `json:"rank"`
}

type TOPSISResponse struct {
	Results          []TOPSISResult `json:"results"`
	DecisionMatrix   [][]float64    `json:"decision_matrix"`
	NormalizedMatrix [][]float64    `json:"normalized_matrix"`
	WeightedMatrix   [][]float64    `json:"weighted_matrix"`
}

type WeatheringRatePoint struct {
	Time         time.Time `json:"time"`
	Rate         float64   `json:"rate"`
	Temperature  float64   `json:"temperature"`
	Humidity     float64   `json:"humidity"`
}
