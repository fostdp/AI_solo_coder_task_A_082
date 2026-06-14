package weathering_predictor

import (
	"encoding/json"
	"math"
	"math/rand"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/gorilla/mux"

	"grotto-monitor/backend/config"
	"grotto-monitor/backend/models"
	"grotto-monitor/backend/repository"
)

type DecisionTreeNode struct {
	FeatureIndex int
	Threshold    float64
	Left         *DecisionTreeNode
	Right        *DecisionTreeNode
	IsLeaf       bool
	Prediction   float64
}

type RandomForest struct {
	Trees        []*DecisionTreeNode
	FeatureCount int
}

type RichTrainingSample struct {
	Temperature    float64
	Humidity       float64
	TempSq         float64
	HumSq          float64
	TempHum        float64
	RockEncoded    float64
	Latitude       float64
	Longitude      float64
	ClimateZone    float64
	Elevation      float64
	AridityIndex   float64
	AnnualAvgTemp  float64
	AnnualAvgHum   float64
	TemperatureVar float64
	HumidityVar    float64
	SeasonFactor   float64
	ExtremeIndex   float64
	Rate           float64
	DomainLabel    int
}

type DomainAdapter struct {
	ProjectionMatrix []float64
	SourceMeans      []float64
	TargetMeans      []float64
	SourceStds       []float64
	TargetStds       []float64
	Trained          bool
	NumFeatures      int
}

func NewDomainAdapter() *DomainAdapter {
	return &DomainAdapter{
		NumFeatures: 17,
	}
}

func (da *DomainAdapter) computeStats(samples []RichTrainingSample) ([]float64, []float64) {
	n := len(samples)
	if n == 0 {
		return make([]float64, da.NumFeatures), make([]float64, da.NumFeatures)
	}

	means := make([]float64, da.NumFeatures)
	for _, sample := range samples {
		feats := sampleToSliceStatic(sample)
		for f := 0; f < da.NumFeatures; f++ {
			means[f] += feats[f]
		}
	}
	for f := 0; f < da.NumFeatures; f++ {
		means[f] /= float64(n)
	}

	stds := make([]float64, da.NumFeatures)
	for _, sample := range samples {
		feats := sampleToSliceStatic(sample)
		for f := 0; f < da.NumFeatures; f++ {
			diff := feats[f] - means[f]
			stds[f] += diff * diff
		}
	}
	for f := 0; f < da.NumFeatures; f++ {
		stds[f] = math.Sqrt(stds[f] / float64(n))
		if stds[f] < 1e-8 {
			stds[f] = 1.0
		}
	}

	return means, stds
}

func (da *DomainAdapter) Fit(source, target []RichTrainingSample) {
	da.SourceMeans, da.SourceStds = da.computeStats(source)
	da.TargetMeans, da.TargetStds = da.computeStats(target)
	da.ProjectionMatrix = make([]float64, da.NumFeatures)
	for f := 0; f < da.NumFeatures; f++ {
		ratio := 1.0
		if da.SourceStds[f] > 1e-8 {
			ratio = da.TargetStds[f] / da.SourceStds[f]
		}
		da.ProjectionMatrix[f] = 0.3 + 0.7*ratio
	}
	da.Trained = true
}

func (da *DomainAdapter) Transform(features []float64) []float64 {
	transformed := make([]float64, len(features))
	if !da.Trained {
		copy(transformed, features)
		return transformed
	}

	for f := 0; f < len(features) && f < da.NumFeatures; f++ {
		normalized := (features[f] - da.SourceMeans[f]) / da.SourceStds[f]
		transformed[f] = normalized*da.TargetStds[f]*da.ProjectionMatrix[f] + da.TargetMeans[f]
	}
	return transformed
}

func (da *DomainAdapter) computeImportanceWeights(
	sourceSamples []RichTrainingSample, targetSample RichTrainingSample,
) []float64 {
	weights := make([]float64, len(sourceSamples))
	targetFeats := sampleToSliceStatic(targetSample)

	for i, source := range sourceSamples {
		sourceFeats := sampleToSliceStatic(source)
		dist := 0.0
		for f := 0; f < da.NumFeatures; f++ {
			diff := (sourceFeats[f] - targetFeats[f]) / (da.SourceStds[f] + 1e-8)
			dist += diff * diff
		}
		dist = math.Sqrt(dist)
		weights[i] = math.Exp(-dist / 50.0)
	}

	sum := 0.0
	for _, w := range weights {
		sum += w
	}
	if sum > 0 {
		for i := range weights {
			weights[i] /= sum
		}
	}

	return weights
}

type WeatheringPredictor struct {
	repo                *repository.Repository
	params              config.PredictionParams
	sourceDomainForest  *RandomForest
	domainAdapter       *DomainAdapter
	featureCache        map[int]*models.GrottoFeatureVector
	cacheMu             sync.RWMutex
}

func NewWeatheringPredictor(repo *repository.Repository, params config.PredictionParams) *WeatheringPredictor {
	wp := &WeatheringPredictor{
		repo:          repo,
		params:        params,
		featureCache:  make(map[int]*models.GrottoFeatureVector),
		domainAdapter: NewDomainAdapter(),
	}
	wp.sourceDomainForest = wp.pretrainSourceDomain()
	return wp
}

func (wp *WeatheringPredictor) RegisterRoutes(r *mux.Router) {
	r.Use(corsMiddleware)
	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/predict", wp.PredictWeatheringRate).Methods("POST", "OPTIONS")
	api.HandleFunc("/predict/batch", wp.PredictBatchRates).Methods("POST", "OPTIONS")
}

var rockTypes = []string{"砂岩", "石灰岩", "花岗岩", "砂砾岩", "砂岩夹泥岩", "火山岩", "变质岩"}

var allRockTypes = []string{"砂岩", "石灰岩", "花岗岩", "砂砾岩", "砂岩夹泥岩", "火山岩", "变质岩"}

var climateZones = []float64{0.3, 0.45, 0.55, 0.65, 0.75, 0.85, 0.95}

func rockTypeOneHot(rockType string) []float64 {
	vec := make([]float64, len(rockTypes))
	for i, rt := range rockTypes {
		if rt == rockType {
			vec[i] = 1.0
			return vec
		}
	}
	vec[0] = 1.0
	return vec
}

func estimateClimateZone(latitude, longitude float64) float64 {
	if latitude >= 35 {
		if longitude < 100 {
			return 0.85
		}
		return 0.65
	} else if latitude >= 30 {
		if longitude < 105 {
			return 0.75
		}
		return 0.55
	}
	return 0.45
}

func estimateElevation(latitude, longitude float64) float64 {
	baseElevation := 500.0
	if latitude > 40 && longitude < 95 {
		baseElevation = 1200.0
	} else if latitude > 35 && longitude < 100 {
		baseElevation = 1500.0
	} else if latitude < 30 && longitude > 105 {
		baseElevation = 800.0
	} else if latitude > 35 && longitude > 110 {
		baseElevation = 1000.0
	}
	return baseElevation
}

func estimateAridity(latitude, longitude float64) float64 {
	if latitude > 38 && longitude < 100 {
		return 0.9
	}
	if latitude > 35 && longitude < 105 {
		return 0.7
	}
	if latitude < 30 && longitude > 105 {
		return 0.3
	}
	return 0.5
}

func estimateAnnualTemp(latitude float64) float64 {
	return 15.0 - (latitude-30.0)*0.6
}

func estimateAnnualHum(latitude, longitude float64) float64 {
	baseHum := 55.0
	if latitude > 40 && longitude < 95 {
		baseHum = 30.0
	} else if longitude < 100 {
		baseHum = 40.0
	} else if latitude < 30 && longitude > 105 {
		baseHum = 75.0
	}
	return baseHum
}

func (wp *WeatheringPredictor) extractGrottoFeatures(site *models.GrottoSite) *models.GrottoFeatureVector {
	wp.cacheMu.RLock()
	if cached, ok := wp.featureCache[site.ID]; ok {
		wp.cacheMu.RUnlock()
		return cached
	}
	wp.cacheMu.RUnlock()

	fv := &models.GrottoFeatureVector{
		Latitude:       site.Latitude,
		Longitude:      site.Longitude,
		RockTypeOneHot: rockTypeOneHot(site.RockType),
		ClimateZone:    estimateClimateZone(site.Latitude, site.Longitude),
		Elevation:      estimateElevation(site.Latitude, site.Longitude),
		AnnualAvgTemp:  estimateAnnualTemp(site.Latitude),
		AnnualAvgHum:   estimateAnnualHum(site.Latitude, site.Longitude),
		AridityIndex:   estimateAridity(site.Latitude, site.Longitude),
		TemperatureVar: 15.0 + math.Abs(site.Latitude-35.0)*0.5,
		HumidityVar:    30.0 + (1.0-estimateAridity(site.Latitude, site.Longitude))*10,
	}

	wp.cacheMu.Lock()
	wp.featureCache[site.ID] = fv
	wp.cacheMu.Unlock()

	return fv
}

func (wp *WeatheringPredictor) sampleToSlice(sample RichTrainingSample) []float64 {
	return []float64{
		sample.Temperature,
		sample.Humidity,
		sample.TempSq,
		sample.HumSq,
		sample.TempHum,
		sample.RockEncoded,
		sample.Latitude,
		sample.Longitude,
		sample.ClimateZone,
		sample.Elevation,
		sample.AridityIndex,
		sample.AnnualAvgTemp,
		sample.AnnualAvgHum,
		sample.TemperatureVar,
		sample.HumidityVar,
		sample.SeasonFactor,
		sample.ExtremeIndex,
	}
}

func sampleToSliceStatic(s RichTrainingSample) []float64 {
	return []float64{
		s.Temperature, s.Humidity, s.TempSq, s.HumSq, s.TempHum,
		s.RockEncoded, s.Latitude, s.Longitude, s.ClimateZone,
		s.Elevation, s.AridityIndex, s.AnnualAvgTemp, s.AnnualAvgHum,
		s.TemperatureVar, s.HumidityVar, s.SeasonFactor, s.ExtremeIndex,
	}
}

func (wp *WeatheringPredictor) pretrainSourceDomain() *RandomForest {
	var sourceSamples []RichTrainingSample

	for rockIdx, rockType := range allRockTypes {
		for ci, cz := range climateZones {
			for lat := 25.0; lat <= 45.0; lat += 2.0 {
				for lon := 80.0; lon <= 120.0; lon += 3.0 {
					for season := 0; season < 4; season++ {
						seasonFactor := math.Sin(2 * math.Pi * float64(season) / 4.0)
						baseHum := estimateAnnualHum(lat, lon)
						baseTemp := estimateAnnualTemp(lat)
						aridity := estimateAridity(lat, lon)
						elevation := estimateElevation(lat, lon)

						for k := 0; k < 3; k++ {
							temp := baseTemp + (rand.Float64()-0.5)*30.0 + seasonFactor*10.0
							hum := baseHum + (rand.Float64()-0.5)*40.0
							rockFactor := wp.rockTypeToFloat(rockType)

							weatheringRate := wp.computePhysicalWeathering(
								temp, hum, rockFactor, cz, elevation,
								aridity, baseTemp, baseHum, seasonFactor,
							)

							extremeIndex := 0.0
							if temp > 35 || temp < -5 {
								extremeIndex += 0.5
							}
							if hum > 85 || hum < 20 {
								extremeIndex += 0.5
							}

							sourceSamples = append(sourceSamples, RichTrainingSample{
								Temperature:    temp,
								Humidity:       hum,
								TempSq:         temp * temp,
								HumSq:          hum * hum,
								TempHum:        temp * hum,
								RockEncoded:    rockFactor,
								Latitude:       lat,
								Longitude:      lon,
								ClimateZone:    cz,
								Elevation:      elevation,
								AridityIndex:   aridity,
								AnnualAvgTemp:  baseTemp,
								AnnualAvgHum:   baseHum,
								TemperatureVar: 15.0 + math.Abs(lat-35.0)*0.5,
								HumidityVar:    30.0 + (1.0-aridity)*10,
								SeasonFactor:   seasonFactor,
								ExtremeIndex:   extremeIndex,
								Rate:           weatheringRate,
								DomainLabel:    rockIdx*10 + ci,
							})
						}
					}
				}
			}
		}
	}

	numFeatures := 17
	featureIndices := make([]int, numFeatures)
	for i := range featureIndices {
		featureIndices[i] = i
	}

	rf := &RandomForest{
		Trees:        make([]*DecisionTreeNode, wp.params.SourceDomainTrees),
		FeatureCount: numFeatures,
	}

	for i := 0; i < wp.params.SourceDomainTrees; i++ {
		bootstrap := make([]RichTrainingSample, len(sourceSamples))
		for j := range bootstrap {
			bootstrap[j] = sourceSamples[rand.Intn(len(sourceSamples))]
		}
		rf.Trees[i] = wp.buildTreeRich(bootstrap, wp.params.SourceMaxDepth, wp.params.SourceMinSplit, 0, featureIndices)
	}

	return rf
}

func (wp *WeatheringPredictor) computePhysicalWeathering(
	temp, hum, rockFactor, climateZone, elevation, aridity, avgTemp, avgHum, seasonFactor float64,
) float64 {
	pm := wp.params.PhysicalModel
	tempDiff := temp - pm.TempCenter
	tempAmplitude := math.Abs(temp - avgTemp)
	humAnomaly := math.Abs(hum - avgHum)

	tempFactor := 1.0 + pm.TempQuadraticCoeff*tempDiff*tempDiff/100.0 + pm.TempAmplitudeCoeff*tempAmplitude
	humFactor := 1.0 + pm.HumidityQuadraticCoeff*hum*hum/10000.0 + pm.HumidityAnomalyCoeff*humAnomaly
	interaction := 1.0 + pm.InteractionCoeff*temp*hum/100.0

	elevationFactor := 1.0 + (elevation-pm.ElevationBase)*pm.ElevationCoeff
	aridityFactor := 1.0 + pm.AridityCoeff*aridity*aridity

	seasonalStress := 1.0 + pm.SeasonalStressCoeff*math.Abs(seasonFactor)

	freezeThaw := 0.0
	if temp < 0 && hum > 50 {
		freezeThaw = pm.FreezeThawCoeff * (1.0 - temp/pm.FreezeThawTempRef) * (hum / 100.0)
	}

	saltCrystallization := 0.0
	if hum > pm.SaltCrystHumLow && hum < pm.SaltCrystHumHigh && temp > pm.SaltCrystTempLow {
		saltCrystallization = pm.SaltCrystCoeff * ((hum - pm.SaltCrystHumLow) / (pm.SaltCrystHumHigh - pm.SaltCrystHumLow)) * ((temp - pm.SaltCrystTempLow) / 20.0)
	}

	climateModifier := pm.ClimateModifierLow + pm.ClimateModifierRange*climateZone

	rate := rockFactor * tempFactor * humFactor * interaction
	rate *= elevationFactor * aridityFactor * seasonalStress
	rate *= climateModifier
	rate += freezeThaw + saltCrystallization

	noise := pm.NoiseLow + rand.Float64()*pm.NoiseRange
	rate *= noise

	return math.Max(wp.params.MinRate, math.Min(wp.params.MaxRate, rate))
}

func (wp *WeatheringPredictor) buildTreeRich(
	samples []RichTrainingSample, maxDepth, minSamplesSplit, currentDepth int, featureIndices []int,
) *DecisionTreeNode {
	if currentDepth >= maxDepth || len(samples) < minSamplesSplit {
		sum := 0.0
		for _, sample := range samples {
			sum += sample.Rate
		}
		return &DecisionTreeNode{
			IsLeaf:     true,
			Prediction: sum / float64(len(samples)),
		}
	}

	numFeatures := len(featureIndices)
	selectedCount := int(math.Sqrt(float64(numFeatures))) + 2
	if selectedCount > numFeatures {
		selectedCount = numFeatures
	}

	selectedFeatures := make([]int, numFeatures)
	copy(selectedFeatures, featureIndices)
	rand.Shuffle(numFeatures, func(i, j int) {
		selectedFeatures[i], selectedFeatures[j] = selectedFeatures[j], selectedFeatures[i]
	})
	selectedFeatures = selectedFeatures[:selectedCount]

	bestFeature, bestThreshold, bestGain := wp.findBestSplitRich(samples, selectedFeatures)
	if bestGain <= 0 {
		sum := 0.0
		for _, sample := range samples {
			sum += sample.Rate
		}
		return &DecisionTreeNode{
			IsLeaf:     true,
			Prediction: sum / float64(len(samples)),
		}
	}

	var left, right []RichTrainingSample
	for _, sample := range samples {
		features := wp.sampleToSlice(sample)
		if features[bestFeature] <= bestThreshold {
			left = append(left, sample)
		} else {
			right = append(right, sample)
		}
	}

	return &DecisionTreeNode{
		FeatureIndex: bestFeature,
		Threshold:    bestThreshold,
		Left:         wp.buildTreeRich(left, maxDepth, minSamplesSplit, currentDepth+1, featureIndices),
		Right:        wp.buildTreeRich(right, maxDepth, minSamplesSplit, currentDepth+1, featureIndices),
	}
}

func (wp *WeatheringPredictor) findBestSplitRich(
	samples []RichTrainingSample, featureIndices []int,
) (int, float64, float64) {
	bestFeature := -1
	bestThreshold := 0.0
	bestGain := -1.0

	parentVar := wp.varianceRich(samples)

	for _, fi := range featureIndices {
		values := make([]float64, len(samples))
		for i, sample := range samples {
			features := wp.sampleToSlice(sample)
			values[i] = features[fi]
		}

		sort.Float64s(values)

		for i := 0; i < len(values)-1; i++ {
			if values[i] == values[i+1] {
				continue
			}
			threshold := (values[i] + values[i+1]) / 2

			var leftSamples, rightSamples []RichTrainingSample
			for _, sample := range samples {
				features := wp.sampleToSlice(sample)
				if features[fi] <= threshold {
					leftSamples = append(leftSamples, sample)
				} else {
					rightSamples = append(rightSamples, sample)
				}
			}

			if len(leftSamples) < 2 || len(rightSamples) < 2 {
				continue
			}

			leftVar := wp.varianceRich(leftSamples)
			rightVar := wp.varianceRich(rightSamples)
			weightedVar := (float64(len(leftSamples))*leftVar + float64(len(rightSamples))*rightVar) / float64(len(samples))

			gain := parentVar - weightedVar
			if gain > bestGain {
				bestGain = gain
				bestFeature = fi
				bestThreshold = threshold
			}
		}
	}

	return bestFeature, bestThreshold, bestGain
}

func (wp *WeatheringPredictor) varianceRich(samples []RichTrainingSample) float64 {
	if len(samples) < 2 {
		return 0
	}
	sum := 0.0
	sumSq := 0.0
	for _, sample := range samples {
		sum += sample.Rate
		sumSq += sample.Rate * sample.Rate
	}
	n := float64(len(samples))
	mean := sum / n
	return sumSq/n - mean*mean
}

func (wp *WeatheringPredictor) fineTuneWithTargetData(
	siteFeatures *models.GrottoFeatureVector,
	historicalData []models.MonitoringData,
	rockType string,
) *RandomForest {
	var targetSamples []RichTrainingSample
	rockFactor := wp.rockTypeToFloat(rockType)

	tempData, humData, hardnessData := extractSensorReadings(historicalData)

	if len(tempData) < 10 {
		tempData = generateSyntheticTemps(siteFeatures)
		humData = generateSyntheticHums(siteFeatures)
	}

	seasonData := []float64{-1.0, -0.5, 0.0, 0.5, 1.0, 0.5, 0.0, -0.5}

	for seasonIdx := 0; seasonIdx < 8; seasonIdx++ {
		for dayIdx := 0; dayIdx < 15; dayIdx++ {
			var temp, hum float64
			if len(tempData) > 0 {
				temp = tempData[(seasonIdx*15+dayIdx)%len(tempData)]
			} else {
				temp = siteFeatures.AnnualAvgTemp + (rand.Float64()-0.5)*20
			}
			if len(humData) > 0 {
				hum = humData[(seasonIdx*15+dayIdx)%len(humData)]
			} else {
				hum = siteFeatures.AnnualAvgHum + (rand.Float64()-0.5)*30
			}

			seasonFactor := seasonData[seasonIdx]
			rate := wp.computePhysicalWeathering(
				temp, hum, rockFactor,
				siteFeatures.ClimateZone,
				siteFeatures.Elevation,
				siteFeatures.AridityIndex,
				siteFeatures.AnnualAvgTemp,
				siteFeatures.AnnualAvgHum,
				seasonFactor,
			)

			if len(hardnessData) > 2 {
				hIdx := (seasonIdx*15 + dayIdx) % len(hardnessData)
				rate = adjustRateFromHardness(rate, hardnessData, hIdx)
			}

			extremeIndex := 0.0
			if temp > 35 || temp < -5 {
				extremeIndex += 0.5
			}
			if hum > 85 || hum < 20 {
				extremeIndex += 0.5
			}

			targetSamples = append(targetSamples, RichTrainingSample{
				Temperature:    temp,
				Humidity:       hum,
				TempSq:         temp * temp,
				HumSq:          hum * hum,
				TempHum:        temp * hum,
				RockEncoded:    rockFactor,
				Latitude:       siteFeatures.Latitude,
				Longitude:      siteFeatures.Longitude,
				ClimateZone:    siteFeatures.ClimateZone,
				Elevation:      siteFeatures.Elevation,
				AridityIndex:   siteFeatures.AridityIndex,
				AnnualAvgTemp:  siteFeatures.AnnualAvgTemp,
				AnnualAvgHum:   siteFeatures.AnnualAvgHum,
				TemperatureVar: siteFeatures.TemperatureVar,
				HumidityVar:    siteFeatures.HumidityVar,
				SeasonFactor:   seasonFactor,
				ExtremeIndex:   extremeIndex,
				Rate:           rate,
			})
		}
	}

	numFeatures := 17
	featureIndices := make([]int, numFeatures)
	for i := range featureIndices {
		featureIndices[i] = i
	}

	rf := &RandomForest{
		Trees:        make([]*DecisionTreeNode, wp.params.TargetDomainTrees),
		FeatureCount: numFeatures,
	}

	for i := 0; i < wp.params.TargetDomainTrees; i++ {
		bootstrap := make([]RichTrainingSample, len(targetSamples))
		for j := range bootstrap {
			bootstrap[j] = targetSamples[rand.Intn(len(targetSamples))]
		}
		rf.Trees[i] = wp.buildTreeRich(bootstrap, wp.params.TargetMaxDepth, wp.params.TargetMinSplit, 0, featureIndices)
	}

	return rf
}

func extractSensorReadings(data []models.MonitoringData) ([]float64, []float64, []models.MonitoringData) {
	var tempData, humData []float64
	var hardnessData []models.MonitoringData

	for _, d := range data {
		sensorType := classifySensor(d.SensorID)
		switch sensorType {
		case "温度":
			tempData = append(tempData, d.Value)
		case "湿度":
			humData = append(humData, d.Value)
		case "表面硬度":
			hardnessData = append(hardnessData, d)
		}
	}

	return tempData, humData, hardnessData
}

func classifySensor(sensorID int) string {
	mod := sensorID % 4
	switch mod {
	case 1:
		return "温度"
	case 2:
		return "湿度"
	case 3:
		return "表面硬度"
	case 0:
		return "裂隙宽度"
	}
	return "未知"
}

func generateSyntheticTemps(fv *models.GrottoFeatureVector) []float64 {
	result := make([]float64, 120)
	for i := 0; i < 120; i++ {
		result[i] = fv.AnnualAvgTemp + (rand.Float64()-0.5)*25
	}
	return result
}

func generateSyntheticHums(fv *models.GrottoFeatureVector) []float64 {
	result := make([]float64, 120)
	for i := 0; i < 120; i++ {
		result[i] = math.Max(10, math.Min(95, fv.AnnualAvgHum+(rand.Float64()-0.5)*35))
	}
	return result
}

func adjustRateFromHardness(baseRate float64, hardnessData []models.MonitoringData, idx int) float64 {
	if idx >= len(hardnessData) {
		return baseRate
	}
	if idx == 0 {
		return baseRate
	}

	prev := hardnessData[idx-1].Value
	curr := hardnessData[idx].Value
	if prev > 0 {
		actualRate := (prev - curr) / prev * 100
		if actualRate > 0 {
			return baseRate*0.6 + actualRate*0.4
		}
	}
	return baseRate
}

func (wp *WeatheringPredictor) rockTypeToFloat(rockType string) float64 {
	if rate, ok := wp.params.RockTypeRates[rockType]; ok {
		return rate
	}
	defaultRate, hasDefault := wp.params.RockTypeRates["砂岩夹泥岩"]
	if hasDefault {
		return defaultRate
	}
	return 0.85
}

func (wp *WeatheringPredictor) getRiskLevel(rate float64) string {
	thresholds := wp.params.RiskThresholds
	lowVal, hasLow := thresholds["低风险"]
	if !hasLow {
		lowVal = 0.3
	}
	midVal, hasMid := thresholds["中等风险"]
	if !hasMid {
		midVal = 0.6
	}
	highVal, hasHigh := thresholds["高风险"]
	if !hasHigh {
		highVal = 0.8
	}

	if rate < lowVal {
		return "低风险"
	} else if rate < midVal {
		return "中等风险"
	} else if rate < highVal {
		return "高风险"
	}
	return "极高风险"
}

func getRecommendation(riskLevel string) string {
	switch riskLevel {
	case "低风险":
		return "岩石风化速率较低，建议维持常规监测频率，每年进行一次详细检查，记录环境参数变化趋势，保持现有保护措施即可。"
	case "中等风险":
		return "岩石风化速率中等，建议加强监测频率至每季度一次，考虑采取表面防护措施如防水涂层，控制洞窟内温湿度波动，减少人为干扰。"
	case "高风险":
		return "岩石风化速率较高，建议立即加强保护措施，采用渗透加固材料进行深层加固，安装环境控制系统维持稳定温湿度，每月进行一次详细检测评估。"
	case "极高风险":
		return "岩石风化速率极高，需紧急采取综合保护措施！建议立即实施深层渗透加固和表面封护联合方案，部署全自动环境监控系统，限制游客进入，组织专家论证长期保护方案。"
	default:
		return ""
	}
}

func (wp *WeatheringPredictor) Predict(
	site *models.GrottoSite,
	historicalData []models.MonitoringData,
	req models.PredictionRequest,
) (*models.PredictionResult, error) {
	siteFeatures := wp.extractGrottoFeatures(site)
	rockFactor := wp.rockTypeToFloat(site.RockType)

	seasonFactor := computeSeasonFactor()
	tempAmplitude := math.Abs(req.Temperature - siteFeatures.AnnualAvgTemp)
	humAnomaly := math.Abs(req.Humidity - siteFeatures.AnnualAvgHum)

	extremeIndex := 0.0
	if req.Temperature > 35 || req.Temperature < -5 {
		extremeIndex += 0.5
	}
	if req.Humidity > 85 || req.Humidity < 20 {
		extremeIndex += 0.5
	}

	pm := wp.params.PhysicalModel
	baseRate := wp.rockTypeToFloat(site.RockType)
	tempFactor := 1 + pm.TempQuadraticCoeff*math.Pow(req.Temperature-pm.TempCenter, 2)/100 + pm.TempAmplitudeCoeff*tempAmplitude
	humidityFactor := 1 + pm.HumidityQuadraticCoeff*math.Pow(req.Humidity/100, 2) + pm.HumidityAnomalyCoeff*humAnomaly
	interactionFactor := 1 + pm.InteractionCoeff*req.Temperature*req.Humidity/100

	elevationFactor := 1.0 + (siteFeatures.Elevation-pm.ElevationBase)*pm.ElevationCoeff
	aridityFactor := 1.0 + pm.AridityCoeff*siteFeatures.AridityIndex*siteFeatures.AridityIndex
	seasonalStress := 1.0 + pm.SeasonalStressCoeff*math.Abs(seasonFactor)
	climateModifier := pm.ClimateModifierLow + pm.ClimateModifierRange*siteFeatures.ClimateZone

	freezeThaw := 0.0
	if req.Temperature < 0 && req.Humidity > 50 {
		freezeThaw = pm.FreezeThawCoeff * (1.0 - req.Temperature/pm.FreezeThawTempRef) * (req.Humidity / 100.0)
	}

	saltCrystallization := 0.0
	if req.Humidity > pm.SaltCrystHumLow && req.Humidity < pm.SaltCrystHumHigh && req.Temperature > pm.SaltCrystTempLow {
		saltCrystallization = pm.SaltCrystCoeff * ((req.Humidity - pm.SaltCrystHumLow) / (pm.SaltCrystHumHigh - pm.SaltCrystHumLow)) * ((req.Temperature - pm.SaltCrystTempLow) / 20.0)
	}

	finalRate := baseRate * tempFactor * humidityFactor * interactionFactor
	finalRate *= elevationFactor * aridityFactor * seasonalStress * climateModifier
	finalRate += freezeThaw + saltCrystallization
	finalRate = math.Max(wp.params.MinRate, math.Min(wp.params.MaxRate, finalRate))

	riskLevel := wp.getRiskLevel(finalRate)
	recommendation := getRecommendation(riskLevel)

	dataFactor := math.Min(float64(len(historicalData))/8760.0, 1.0)
	confidence := wp.params.ConfidenceBase + wp.params.ConfidenceDataFactor*dataFactor + wp.params.ConfidenceExtremeFactor*(1.0-extremeIndex*0.5)
	confidence = math.Max(wp.params.MinConfidence, math.Min(wp.params.MaxConfidence, confidence))

	return &models.PredictionResult{
		PredictedRate:  math.Round(finalRate*10000) / 10000,
		RiskLevel:      riskLevel,
		Recommendation: recommendation,
		Confidence:     math.Round(confidence*1000) / 1000,
	}, nil
}

func (wp *WeatheringPredictor) PredictEnsemble(
	site *models.GrottoSite,
	historicalData []models.MonitoringData,
	req models.PredictionRequest,
) (*models.PredictionResult, error) {
	regressionResult, err := wp.Predict(site, historicalData, req)
	if err != nil {
		return nil, err
	}

	siteFeatures := wp.extractGrottoFeatures(site)
	targetForest := wp.fineTuneWithTargetData(siteFeatures, historicalData, site.RockType)

	seasonFactor := computeSeasonFactor()
	extremeIndex := 0.0
	if req.Temperature > 35 || req.Temperature < -5 {
		extremeIndex += 0.5
	}
	if req.Humidity > 85 || req.Humidity < 20 {
		extremeIndex += 0.5
	}

	rockFactor := wp.rockTypeToFloat(site.RockType)
	richSample := RichTrainingSample{
		Temperature:    req.Temperature,
		Humidity:       req.Humidity,
		TempSq:         req.Temperature * req.Temperature,
		HumSq:          req.Humidity * req.Humidity,
		TempHum:        req.Temperature * req.Humidity,
		RockEncoded:    rockFactor,
		Latitude:       siteFeatures.Latitude,
		Longitude:      siteFeatures.Longitude,
		ClimateZone:    siteFeatures.ClimateZone,
		Elevation:      siteFeatures.Elevation,
		AridityIndex:   siteFeatures.AridityIndex,
		AnnualAvgTemp:  siteFeatures.AnnualAvgTemp,
		AnnualAvgHum:   siteFeatures.AnnualAvgHum,
		TemperatureVar: siteFeatures.TemperatureVar,
		HumidityVar:    siteFeatures.HumidityVar,
		SeasonFactor:   seasonFactor,
		ExtremeIndex:   extremeIndex,
	}

	richFeatures := wp.sampleToSlice(richSample)

	sourcePredictions := make([]float64, len(wp.sourceDomainForest.Trees))
	for i, tree := range wp.sourceDomainForest.Trees {
		sourcePredictions[i] = tree.predictRich(richFeatures)
	}

	targetPredictions := make([]float64, len(targetForest.Trees))
	for i, tree := range targetForest.Trees {
		targetPredictions[i] = tree.predictRich(richFeatures)
	}

	dataAdequacy := math.Min(float64(len(historicalData))/4380.0, 1.0)
	transferWeight := wp.params.BaseTargetWeight + wp.params.DataAdequacyTargetBoost*dataAdequacy
	sourceWeight := 1.0 - transferWeight

	sourceMean := 0.0
	for _, p := range sourcePredictions {
		sourceMean += p
	}
	sourceMean /= float64(len(sourcePredictions))

	targetMean := 0.0
	for _, p := range targetPredictions {
		targetMean += p
	}
	targetMean /= float64(len(targetPredictions))

	rfPrediction := sourceWeight*sourceMean + transferWeight*targetMean
	rfPrediction = math.Max(wp.params.MinRate, math.Min(wp.params.MaxRate, rfPrediction))

	sourceVar := 0.0
	for _, p := range sourcePredictions {
		sourceVar += (p - sourceMean) * (p - sourceMean)
	}
	sourceVar /= float64(len(sourcePredictions))

	targetVar := 0.0
	for _, p := range targetPredictions {
		targetVar += (p - targetMean) * (p - targetMean)
	}
	targetVar /= float64(len(targetPredictions))

	rfStdDev := math.Sqrt(sourceWeight*sourceWeight*sourceVar + transferWeight*transferWeight*targetVar)

	ensembleRate := regressionResult.PredictedRate*wp.params.RegressionWeight + rfPrediction*wp.params.ForestWeight
	ensembleRate = math.Max(wp.params.MinRate, math.Min(wp.params.MaxRate, ensembleRate))

	rfConfidence := 1.0 - math.Min(rfStdDev/0.6, 0.6)
	dataFactor := math.Min(float64(len(historicalData))/8760.0, 1.0)
	ensembleConfidence := regressionResult.Confidence*0.3 + rfConfidence*0.4 + 0.3*dataFactor
	ensembleConfidence = math.Max(wp.params.MinConfidence, math.Min(wp.params.MaxConfidence, ensembleConfidence))

	riskLevel := wp.getRiskLevel(ensembleRate)
	recommendation := getRecommendation(riskLevel)

	return &models.PredictionResult{
		PredictedRate:  math.Round(ensembleRate*10000) / 10000,
		RiskLevel:      riskLevel,
		Recommendation: recommendation,
		Confidence:     math.Round(ensembleConfidence*1000) / 1000,
	}, nil
}

func (wp *WeatheringPredictor) PredictBatch(
	rockType string,
	requests []models.WeatheringPredictionRequest,
	dataCount int,
) (*models.BatchPredictionResponse, error) {
	predictions := make([]models.WeatheringPredictionResponse, 0, len(requests))
	baseRate := wp.rockTypeToFloat(rockType)
	rockFactor := wp.rockTypeToFloat(rockType)
	pm := wp.params.PhysicalModel

	for _, req := range requests {
		T := req.Temperature
		H := req.Humidity

		tempFactor := 1 + pm.TempQuadraticCoeff*math.Pow(T-pm.TempCenter, 2)/100
		humidityFactor := 1 + pm.HumidityQuadraticCoeff*math.Pow(H/100, 2)
		interactionFactor := 1 + pm.InteractionCoeff*T*H/100

		finalRate := baseRate * tempFactor * humidityFactor * interactionFactor * rockFactor

		riskLevel := wp.getRiskLevel(finalRate)
		recommendation := getRecommendation(riskLevel)

		dataFactor := math.Min(float64(dataCount)/8760.0, 1.0)
		confidence := 0.5 + 0.5*dataFactor
		confidence = math.Max(wp.params.MinConfidence, math.Min(wp.params.MaxConfidence, confidence))

		predictions = append(predictions, models.WeatheringPredictionResponse{
			PredictedRate:  math.Round(finalRate*10000) / 10000,
			RiskLevel:      riskLevel,
			Recommendation: recommendation,
			Confidence:     math.Round(confidence*1000) / 1000,
		})
	}
	return &models.BatchPredictionResponse{Predictions: predictions}, nil
}

func computeSeasonFactor() float64 {
	month := float64(timeNowMonth())
	return math.Sin(2 * math.Pi * (month - 3.0) / 12.0)
}

func timeNowMonth() int {
	return int(time.Now().Month())
}

func (tree *DecisionTreeNode) predictRich(features []float64) float64 {
	if tree.IsLeaf {
		return tree.Prediction
	}
	if features[tree.FeatureIndex] <= tree.Threshold {
		return tree.Left.predictRich(features)
	}
	return tree.Right.predictRich(features)
}

func (wp *WeatheringPredictor) PredictWeatheringRate(w http.ResponseWriter, r *http.Request) {
	var input models.PredictionRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	site, err := wp.repo.GetSiteByID(int64(input.SiteID))
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if site == nil {
		respondError(w, http.StatusNotFound, "site not found")
		return
	}
	historicalData, err := wp.repo.GetMonitoringDataBySite(int64(input.SiteID), 365*24)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get historical data")
		return
	}
	result, err := wp.PredictEnsemble(site, historicalData, input)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, result)
}

func (wp *WeatheringPredictor) PredictBatchRates(w http.ResponseWriter, r *http.Request) {
	var input models.BatchPredictionRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	site, err := wp.repo.GetSiteByID(int64(input.SiteID))
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if site == nil {
		respondError(w, http.StatusNotFound, "site not found")
		return
	}
	result, err := wp.PredictBatch(site.RockType, input.Combinations, len(input.Combinations))
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, result)
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

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": status < 400,
		"data":    data,
	})
}

func respondError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"error":   message,
	})
}
