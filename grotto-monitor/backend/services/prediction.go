package services

import (
	"math"
	"math/rand"
	"sort"

	"grotto-monitor/backend/models"
)

type WeatheringPredictionService struct{}

func getBaseRate(rockType string) float64 {
	switch rockType {
	case "砂岩":
		return 0.8
	case "石灰岩":
		return 1.0
	case "花岗岩":
		return 0.5
	case "砂砾岩":
		return 0.9
	case "砂岩夹泥岩":
		return 0.85
	default:
		return 1.0
	}
}

func getRiskLevel(rate float64) string {
	if rate < 0.3 {
		return "低风险"
	} else if rate < 0.6 {
		return "中等风险"
	} else if rate < 0.8 {
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

func calculateDataStats(data []models.MonitoringData) (tempData, humData []float64, hardnessData []models.MonitoringData) {
	for _, d := range data {
		if d.SensorID%4 == 1 {
			tempData = append(tempData, d.Value)
		} else if d.SensorID%4 == 2 {
			humData = append(humData, d.Value)
		} else if d.SensorID%4 == 3 {
			hardnessData = append(hardnessData, d)
		}
	}
	return
}

func (s *WeatheringPredictionService) Predict(site *models.GrottoSite, historicalData []models.MonitoringData, req models.PredictionRequest) (*models.PredictionResult, error) {
	baseRate := getBaseRate(site.RockType)

	T := req.Temperature
	H := req.Humidity

	tempFactor := 1 + 0.02*math.Pow(T-20, 2)/100
	humidityFactor := 1 + 0.3*math.Pow(H/100, 2)
	interactionFactor := 1 + 0.005*T*H/100

	finalRate := baseRate * tempFactor * humidityFactor * interactionFactor

	riskLevel := getRiskLevel(finalRate)
	recommendation := getRecommendation(riskLevel)

	dataFactor := math.Min(float64(len(historicalData))/8760.0, 1.0)
	confidence := 0.5 + 0.5*dataFactor
	if confidence > 1.0 {
		confidence = 1.0
	}

	return &models.PredictionResult{
		PredictedRate:  math.Round(finalRate*10000) / 10000,
		RiskLevel:      riskLevel,
		Recommendation: recommendation,
		Confidence:     math.Round(confidence*1000) / 1000,
	}, nil
}

func (s *WeatheringPredictionService) PredictBatch(rockType string, requests []models.WeatheringPredictionRequest, dataCount int) (*models.BatchPredictionResponse, error) {
	predictions := make([]models.WeatheringPredictionResponse, 0, len(requests))
	baseRate := getBaseRate(rockType)

	for _, req := range requests {
		T := req.Temperature
		H := req.Humidity

		tempFactor := 1 + 0.02*math.Pow(T-20, 2)/100
		humidityFactor := 1 + 0.3*math.Pow(H/100, 2)
		interactionFactor := 1 + 0.005*T*H/100

		finalRate := baseRate * tempFactor * humidityFactor * interactionFactor

		riskLevel := getRiskLevel(finalRate)
		recommendation := getRecommendation(riskLevel)

		dataFactor := math.Min(float64(dataCount)/8760.0, 1.0)
		confidence := 0.5 + 0.5*dataFactor
		if confidence > 1.0 {
			confidence = 1.0
		}

		predictions = append(predictions, models.WeatheringPredictionResponse{
			PredictedRate:  math.Round(finalRate*10000) / 10000,
			RiskLevel:      riskLevel,
			Recommendation: recommendation,
			Confidence:     math.Round(confidence*1000) / 1000,
		})
	}
	return &models.BatchPredictionResponse{Predictions: predictions}, nil
}

type DecisionTreeNode struct {
	FeatureIndex int
	Threshold    float64
	Left         *DecisionTreeNode
	Right        *DecisionTreeNode
	IsLeaf       bool
	Prediction   float64
}

type RandomForest struct {
	Trees []*DecisionTreeNode
}

type TrainingSample struct {
	Temperature float64
	Humidity    float64
	RockType    float64
	Rate        float64
}

func rockTypeToFloat(rockType string) float64 {
	switch rockType {
	case "砂岩":
		return 0.8
	case "石灰岩":
		return 1.0
	case "花岗岩":
		return 0.5
	case "砂砾岩":
		return 0.9
	default:
		return 0.85
	}
}

func generateTrainingData(rockType string, count int) []TrainingSample {
	samples := make([]TrainingSample, count)
	baseRate := getBaseRate(rockType)
	rockFactor := rockTypeToFloat(rockType)

	for i := 0; i < count; i++ {
		temp := -10 + rand.Float64()*60
		humidity := rand.Float64() * 100

		tempFactor := 1 + 0.02*math.Pow(temp-20, 2)/100
		humidityFactor := 1 + 0.3*math.Pow(humidity/100, 2)
		interactionFactor := 1 + 0.005*temp*humidity/100

		noise := 0.9 + rand.Float64()*0.2
		rate := baseRate * tempFactor * humidityFactor * interactionFactor * noise

		samples[i] = TrainingSample{
			Temperature: temp,
			Humidity:    humidity,
			RockType:    rockFactor,
			Rate:        rate,
		}
	}
	return samples
}

func calculateGiniImpurity(samples []TrainingSample) float64 {
	if len(samples) == 0 {
		return 0
	}

	sum := 0.0
	sumSq := 0.0
	for _, s := range samples {
		sum += s.Rate
		sumSq += s.Rate * s.Rate
	}

	mean := sum / float64(len(samples))
	variance := sumSq/float64(len(samples)) - mean*mean
	return variance
}

func findBestSplit(samples []TrainingSample, featureIndices []int) (int, float64, float64) {
	bestFeature := -1
	bestThreshold := 0.0
	bestGain := -1.0

	parentImpurity := calculateGiniImpurity(samples)

	for _, featureIdx := range featureIndices {
		var values []float64
		for _, s := range samples {
			switch featureIdx {
			case 0:
				values = append(values, s.Temperature)
			case 1:
				values = append(values, s.Humidity)
			case 2:
				values = append(values, s.RockType)
			}
		}

		sort.Float64s(values)
		for i := 0; i < len(values)-1; i++ {
			threshold := (values[i] + values[i+1]) / 2

			var left, right []TrainingSample
			for _, s := range samples {
				var val float64
				switch featureIdx {
				case 0:
					val = s.Temperature
				case 1:
					val = s.Humidity
				case 2:
					val = s.RockType
				}
				if val <= threshold {
					left = append(left, s)
				} else {
					right = append(right, s)
				}
			}

			if len(left) == 0 || len(right) == 0 {
				continue
			}

			leftImpurity := calculateGiniImpurity(left)
			rightImpurity := calculateGiniImpurity(right)
			weightedImpurity := (float64(len(left))*leftImpurity + float64(len(right))*rightImpurity) / float64(len(samples))

			gain := parentImpurity - weightedImpurity
			if gain > bestGain {
				bestGain = gain
				bestFeature = featureIdx
				bestThreshold = threshold
			}
		}
	}

	return bestFeature, bestThreshold, bestGain
}

func buildTree(samples []TrainingSample, maxDepth, minSamplesSplit, currentDepth int, featureIndices []int) *DecisionTreeNode {
	if currentDepth >= maxDepth || len(samples) < minSamplesSplit {
		sum := 0.0
		for _, s := range samples {
			sum += s.Rate
		}
		return &DecisionTreeNode{
			IsLeaf:     true,
			Prediction: sum / float64(len(samples)),
		}
	}

	selectedFeatures := make([]int, len(featureIndices))
	copy(selectedFeatures, featureIndices)
	rand.Shuffle(len(selectedFeatures), func(i, j int) {
		selectedFeatures[i], selectedFeatures[j] = selectedFeatures[j], selectedFeatures[i]
	})
	selectedFeatures = selectedFeatures[:int(math.Sqrt(float64(len(selectedFeatures))))+1]

	bestFeature, bestThreshold, bestGain := findBestSplit(samples, selectedFeatures)
	if bestGain <= 0 {
		sum := 0.0
		for _, s := range samples {
			sum += s.Rate
		}
		return &DecisionTreeNode{
			IsLeaf:     true,
			Prediction: sum / float64(len(samples)),
		}
	}

	var left, right []TrainingSample
	for _, s := range samples {
		var val float64
		switch bestFeature {
		case 0:
			val = s.Temperature
		case 1:
			val = s.Humidity
		case 2:
			val = s.RockType
		}
		if val <= bestThreshold {
			left = append(left, s)
		} else {
			right = append(right, s)
		}
	}

	return &DecisionTreeNode{
		FeatureIndex: bestFeature,
		Threshold:    bestThreshold,
		Left:         buildTree(left, maxDepth, minSamplesSplit, currentDepth+1, featureIndices),
		Right:        buildTree(right, maxDepth, minSamplesSplit, currentDepth+1, featureIndices),
	}
}

func (tree *DecisionTreeNode) predict(sample TrainingSample) float64 {
	if tree.IsLeaf {
		return tree.Prediction
	}

	var val float64
	switch tree.FeatureIndex {
	case 0:
		val = sample.Temperature
	case 1:
		val = sample.Humidity
	case 2:
		val = sample.RockType
	}

	if val <= tree.Threshold {
		return tree.Left.predict(sample)
	}
	return tree.Right.predict(sample)
}

func (rf *RandomForest) Predict(sample TrainingSample) float64 {
	sum := 0.0
	for _, tree := range rf.Trees {
		sum += tree.predict(sample)
	}
	return sum / float64(len(rf.Trees))
}

func trainRandomForest(samples []TrainingSample, numTrees, maxDepth, minSamplesSplit int) *RandomForest {
	rf := &RandomForest{
		Trees: make([]*DecisionTreeNode, numTrees),
	}

	featureIndices := []int{0, 1, 2}

	for i := 0; i < numTrees; i++ {
		bootstrapSample := make([]TrainingSample, len(samples))
		for j := range bootstrapSample {
			bootstrapSample[j] = samples[rand.Intn(len(samples))]
		}

		rf.Trees[i] = buildTree(bootstrapSample, maxDepth, minSamplesSplit, 0, featureIndices)
	}

	return rf
}

func (s *WeatheringPredictionService) PredictWithRandomForest(rockType string, temp, humidity float64) (float64, float64, error) {
	trainingData := generateTrainingData(rockType, 500)
	rf := trainRandomForest(trainingData, 50, 10, 5)

	sample := TrainingSample{
		Temperature: temp,
		Humidity:    humidity,
		RockType:    rockTypeToFloat(rockType),
	}

	predictions := make([]float64, len(rf.Trees))
	for i, tree := range rf.Trees {
		predictions[i] = tree.predict(sample)
	}

	mean := 0.0
	for _, p := range predictions {
		mean += p
	}
	mean /= float64(len(predictions))

	variance := 0.0
	for _, p := range predictions {
		variance += (p - mean) * (p - mean)
	}
	variance /= float64(len(predictions))
	stdDev := math.Sqrt(variance)

	return mean, stdDev, nil
}

func (s *WeatheringPredictionService) PredictEnsemble(site *models.GrottoSite, historicalData []models.MonitoringData, req models.PredictionRequest) (*models.PredictionResult, error) {
	regressionResult, err := s.Predict(site, historicalData, req)
	if err != nil {
		return nil, err
	}

	rfMean, rfStdDev, err := s.PredictWithRandomForest(site.RockType, req.Temperature, req.Humidity)
	if err != nil {
		return regressionResult, nil
	}

	ensembleRate := (regressionResult.PredictedRate + rfMean) / 2

	dataFactor := math.Min(float64(len(historicalData))/8760.0, 1.0)
	rfConfidence := 1.0 - math.Min(rfStdDev/0.5, 0.5)
	ensembleConfidence := (regressionResult.Confidence + rfConfidence*dataFactor) / 2

	riskLevel := getRiskLevel(ensembleRate)
	recommendation := getRecommendation(riskLevel)

	return &models.PredictionResult{
		PredictedRate:  math.Round(ensembleRate*10000) / 10000,
		RiskLevel:      riskLevel,
		Recommendation: recommendation,
		Confidence:     math.Round(ensembleConfidence*1000) / 1000,
	}, nil
}
