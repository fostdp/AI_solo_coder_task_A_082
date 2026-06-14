package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type PhysicalModelParams struct {
	TempCenter           float64 `json:"temp_center"`
	TempQuadraticCoeff   float64 `json:"temp_quadratic_coeff"`
	TempAmplitudeCoeff   float64 `json:"temp_amplitude_coeff"`
	HumidityQuadraticCoeff float64 `json:"humidity_quadratic_coeff"`
	HumidityAnomalyCoeff float64 `json:"humidity_anomaly_coeff"`
	InteractionCoeff     float64 `json:"interaction_coeff"`
	ElevationBase        float64 `json:"elevation_base"`
	ElevationCoeff       float64 `json:"elevation_coeff"`
	AridityCoeff         float64 `json:"aridity_coeff"`
	SeasonalStressCoeff  float64 `json:"seasonal_stress_coeff"`
	ClimateModifierLow   float64 `json:"climate_modifier_low"`
	ClimateModifierRange float64 `json:"climate_modifier_range"`
	FreezeThawCoeff      float64 `json:"freeze_thaw_coeff"`
	FreezeThawTempRef    float64 `json:"freeze_thaw_temp_ref"`
	SaltCrystCoeff       float64 `json:"salt_cryst_coeff"`
	SaltCrystHumLow      float64 `json:"salt_cryst_hum_low"`
	SaltCrystHumHigh     float64 `json:"salt_cryst_hum_high"`
	SaltCrystTempLow     float64 `json:"salt_cryst_temp_low"`
	NoiseLow             float64 `json:"noise_low"`
	NoiseRange           float64 `json:"noise_range"`
}

type PredictionParams struct {
	SourceDomainTrees       int                   `json:"source_domain_trees"`
	TargetDomainTrees       int                   `json:"target_domain_trees"`
	SourceMaxDepth          int                   `json:"source_max_depth"`
	TargetMaxDepth          int                   `json:"target_max_depth"`
	SourceMinSplit          int                   `json:"source_min_split"`
	TargetMinSplit          int                   `json:"target_min_split"`
	RegressionWeight        float64               `json:"regression_weight"`
	ForestWeight            float64               `json:"forest_weight"`
	BaseSourceWeight        float64               `json:"base_source_weight"`
	BaseTargetWeight        float64               `json:"base_target_weight"`
	DataAdequacyTargetBoost float64               `json:"data_adequacy_target_boost"`
	ConfidenceBase          float64               `json:"confidence_base"`
	ConfidenceDataFactor    float64               `json:"confidence_data_factor"`
	ConfidenceExtremeFactor float64               `json:"confidence_extreme_factor"`
	MinConfidence           float64               `json:"min_confidence"`
	MaxConfidence           float64               `json:"max_confidence"`
	MinRate                 float64               `json:"min_rate"`
	MaxRate                 float64               `json:"max_rate"`
	RateAdjustHardnessWeight float64              `json:"rate_adjust_hardness_weight"`
	RateAdjustBaseWeight    float64               `json:"rate_adjust_base_weight"`
	PhysicalModel           PhysicalModelParams   `json:"physical_model"`
	RockTypeRates           map[string]float64    `json:"rock_type_rates"`
	RiskThresholds          map[string]float64    `json:"risk_thresholds"`
}

type TOPSISParams struct {
	KnnK                     int                `json:"knn_k"`
	SensitivityPerturbation  float64            `json:"sensitivity_perturbation"`
	StabilitySensitivityScale float64           `json:"stability_sensitivity_scale"`
	DefaultWeights           map[string]float64 `json:"default_weights"`
	AttributeRanges          map[string][2]float64 `json:"attribute_ranges"`
	NonBenefitAttributes     []string           `json:"non_benefit_attributes"`
}

type AlarmParams struct {
	CrackWidthThreshold    float64 `json:"crack_width_threshold"`
	HardnessDropPercent    float64 `json:"hardness_drop_percent"`
	WebSocketBufferSize    int     `json:"websocket_buffer_size"`
	BroadcastBufferSize    int     `json:"broadcast_buffer_size"`
	PingIntervalSeconds    int     `json:"ping_interval_seconds"`
	PongTimeoutSeconds     int     `json:"pong_timeout_seconds"`
	WriteTimeoutSeconds    int     `json:"write_timeout_seconds"`
	OfflineMaxPerClient    int     `json:"offline_max_per_client"`
	OfflineTTLHours        int     `json:"offline_ttl_hours"`
	CleanupIntervalMinutes int     `json:"cleanup_interval_minutes"`
	OfflinePersistPath     string  `json:"offline_persist_path"`
}

type ModelParams struct {
	Prediction PredictionParams `json:"prediction"`
	Topsis     TOPSISParams     `json:"topsis"`
	Alarm      AlarmParams      `json:"alarm"`
}

func LoadModelParams(path string) (*ModelParams, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file %s: %w", path, err)
	}
	var params ModelParams
	if err := json.Unmarshal(data, &params); err != nil {
		return nil, fmt.Errorf("parsing config file %s: %w", path, err)
	}
	return &params, nil
}

func DefaultModelParams() *ModelParams {
	return &ModelParams{
		Prediction: PredictionParams{
			SourceDomainTrees: 80, TargetDomainTrees: 40,
			SourceMaxDepth: 12, TargetMaxDepth: 10,
			SourceMinSplit: 8, TargetMinSplit: 6,
			RegressionWeight: 0.35, ForestWeight: 0.65,
			BaseSourceWeight: 0.45, BaseTargetWeight: 0.55,
			DataAdequacyTargetBoost: 0.25,
			ConfidenceBase: 0.4, ConfidenceDataFactor: 0.3, ConfidenceExtremeFactor: 0.3,
			MinConfidence: 0.3, MaxConfidence: 0.98,
			MinRate: 0.1, MaxRate: 3.0,
			RateAdjustHardnessWeight: 0.4, RateAdjustBaseWeight: 0.6,
			PhysicalModel: PhysicalModelParams{
				TempCenter: 20.0, TempQuadraticCoeff: 0.015, TempAmplitudeCoeff: 0.008,
				HumidityQuadraticCoeff: 0.25, HumidityAnomalyCoeff: 0.005,
				InteractionCoeff: 0.003, ElevationBase: 1000.0, ElevationCoeff: 0.0001,
				AridityCoeff: 0.3, SeasonalStressCoeff: 0.15,
				ClimateModifierLow: 0.7, ClimateModifierRange: 0.6,
				FreezeThawCoeff: 0.3, FreezeThawTempRef: -20.0,
				SaltCrystCoeff: 0.2, SaltCrystHumLow: 70.0, SaltCrystHumHigh: 90.0, SaltCrystTempLow: 20.0,
				NoiseLow: 0.92, NoiseRange: 0.16,
			},
			RockTypeRates: map[string]float64{
				"砂岩": 0.8, "石灰岩": 1.0, "花岗岩": 0.5,
				"砂砾岩": 0.9, "砂岩夹泥岩": 0.85, "火山岩": 0.7, "变质岩": 0.6,
			},
			RiskThresholds: map[string]float64{"低风险": 0.3, "中等风险": 0.6, "高风险": 0.8},
		},
		Topsis: TOPSISParams{
			KnnK: 3, SensitivityPerturbation: 0.2, StabilitySensitivityScale: 2.0,
			DefaultWeights: map[string]float64{
				"penetration_depth": 0.15, "breathability": 0.20,
				"weathering_resistance": 0.25, "compatibility": 0.20,
				"cost": 0.10, "durability_years": 0.10,
			},
			AttributeRanges: map[string][2]float64{
				"penetration_depth": {0.5, 5.0}, "breathability": {50.0, 95.0},
				"weathering_resistance": {60.0, 98.0}, "compatibility": {50.0, 95.0},
				"cost": {20.0, 500.0}, "durability_years": {3.0, 30.0},
			},
			NonBenefitAttributes: []string{"cost"},
		},
		Alarm: AlarmParams{
			CrackWidthThreshold: 0.5, HardnessDropPercent: 20.0,
			WebSocketBufferSize: 512, BroadcastBufferSize: 512,
			PingIntervalSeconds: 54, PongTimeoutSeconds: 90, WriteTimeoutSeconds: 15,
			OfflineMaxPerClient: 100, OfflineTTLHours: 24,
			CleanupIntervalMinutes: 30, OfflinePersistPath: "data/offline_messages.json",
		},
	}
}
