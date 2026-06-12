package algorithms

import (
	"math"
	"sort"
	"time"
)

type DecisionTree struct {
	MaxDepth int
	MinSize  int
	Root     *TreeNode
}

type TreeNode struct {
	FeatureIndex int
	Threshold    float64
	Left         *TreeNode
	Right        *TreeNode
	Prediction   float64
	IsLeaf       bool
}

type Sample struct {
	Features []float64
	Target   float64
}

type RandomForestRegressor struct {
	Trees        []*DecisionTree
	NumTrees     int
	MaxDepth     int
	MinSize      int
	NumFeatures  int
	FeatureNames []string
}

func NewRandomForestRegressor(numTrees, maxDepth, minSize int) *RandomForestRegressor {
	return &RandomForestRegressor{
		Trees:    make([]*DecisionTree, 0, numTrees),
		NumTrees: numTrees,
		MaxDepth: maxDepth,
		MinSize:  minSize,
	}
}

func (rf *RandomForestRegressor) Train(samples []Sample) {
	n := len(samples)
	if rf.NumFeatures == 0 {
		if len(samples) > 0 {
			rf.NumFeatures = int(math.Sqrt(float64(len(samples[0].Features)))) + 1
		}
	}

	for i := 0; i < rf.NumTrees; i++ {
		bootstrapped := bootstrapSample(samples, n)
		tree := &DecisionTree{
			MaxDepth: rf.MaxDepth,
			MinSize:  rf.MinSize,
		}
		tree.Root = buildTree(bootstrapped, 0, rf.MaxDepth, rf.MinSize, rf.NumFeatures)
		rf.Trees = append(rf.Trees, tree)
	}
}

func (rf *RandomForestRegressor) Predict(features []float64) float64 {
	if len(rf.Trees) == 0 {
		return 0
	}
	var sum float64
	for _, tree := range rf.Trees {
		sum += predictTree(tree.Root, features)
	}
	return sum / float64(len(rf.Trees))
}

func predictTree(node *TreeNode, features []float64) float64 {
	if node.IsLeaf {
		return node.Prediction
	}
	if features[node.FeatureIndex] < node.Threshold {
		return predictTree(node.Left, features)
	}
	return predictTree(node.Right, features)
}

func bootstrapSample(samples []Sample, n int) []Sample {
	result := make([]Sample, n)
	for i := 0; i < n; i++ {
		idx := int(math.Mod(float64(i*31+7), float64(len(samples))))
		if idx < 0 {
			idx = -idx
		}
		result[i] = samples[idx]
	}
	return result
}

func buildTree(samples []Sample, depth, maxDepth, minSize, numFeatures int) *TreeNode {
	if depth >= maxDepth || len(samples) <= minSize || isHomogeneous(samples) {
		return &TreeNode{
			IsLeaf:     true,
			Prediction: meanTarget(samples),
		}
	}

	bestFeature, bestThreshold, bestGain := findBestSplit(samples, numFeatures)
	if bestGain <= 0 {
		return &TreeNode{
			IsLeaf:     true,
			Prediction: meanTarget(samples),
		}
	}

	left, right := splitSamples(samples, bestFeature, bestThreshold)
	if len(left) == 0 || len(right) == 0 {
		return &TreeNode{
			IsLeaf:     true,
			Prediction: meanTarget(samples),
		}
	}

	return &TreeNode{
		FeatureIndex: bestFeature,
		Threshold:    bestThreshold,
		Left:         buildTree(left, depth+1, maxDepth, minSize, numFeatures),
		Right:        buildTree(right, depth+1, maxDepth, minSize, numFeatures),
	}
}

func findBestSplit(samples []Sample, numFeatures int) (int, float64, float64) {
	nFeatures := len(samples[0].Features)
	selectedFeatures := selectFeatures(nFeatures, numFeatures)

	bestGain := -1e9
	bestFeature := 0
	bestThreshold := 0.0

	for _, fIdx := range selectedFeatures {
		values := extractFeatureValues(samples, fIdx)
		sort.Float64s(values)

		for i := 1; i < len(values); i += int(math.Max(1, float64(len(values))/20)) {
			threshold := (values[i] + values[i-1]) / 2.0
			left, right := splitSamples(samples, fIdx, threshold)
			if len(left) < 2 || len(right) < 2 {
				continue
			}
			gain := calculateGain(samples, left, right)
			if gain > bestGain {
				bestGain = gain
				bestFeature = fIdx
				bestThreshold = threshold
			}
		}
	}
	return bestFeature, bestThreshold, bestGain
}

func selectFeatures(nFeatures, k int) []int {
	if k >= nFeatures {
		indices := make([]int, nFeatures)
		for i := range indices {
			indices[i] = i
		}
		return indices
	}
	selected := make([]int, k)
	used := make(map[int]bool)
	for i := 0; i < k; i++ {
		for {
			idx := (i * 7 + 3) % nFeatures
			if !used[idx] {
				used[idx] = true
				selected[i] = idx
				break
			}
		}
	}
	return selected
}

func extractFeatureValues(samples []Sample, fIdx int) []float64 {
	values := make([]float64, len(samples))
	for i, s := range samples {
		values[i] = s.Features[fIdx]
	}
	return values
}

func splitSamples(samples []Sample, fIdx int, threshold float64) ([]Sample, []Sample) {
	var left, right []Sample
	for _, s := range samples {
		if s.Features[fIdx] < threshold {
			left = append(left, s)
		} else {
			right = append(right, s)
		}
	}
	return left, right
}

func calculateGain(parent, left, right []Sample) float64 {
	parentVar := variance(parent)
	leftVar := variance(left)
	rightVar := variance(right)
	n := float64(len(parent))
	nL := float64(len(left))
	nR := float64(len(right))
	return parentVar - (nL/n*leftVar + nR/n*rightVar)
}

func variance(samples []Sample) float64 {
	if len(samples) < 2 {
		return 0
	}
	mean := meanTarget(samples)
	var sum float64
	for _, s := range samples {
		diff := s.Target - mean
		sum += diff * diff
	}
	return sum / float64(len(samples))
}

func meanTarget(samples []Sample) float64 {
	if len(samples) == 0 {
		return 0
	}
	var sum float64
	for _, s := range samples {
		sum += s.Target
	}
	return sum / float64(len(samples))
}

func isHomogeneous(samples []Sample) bool {
	if len(samples) < 2 {
		return true
	}
	first := samples[0].Target
	epsilon := 1e-6
	for _, s := range samples[1:] {
		if math.Abs(s.Target-first) > epsilon {
			return false
		}
	}
	return true
}

type WeatheringPredictor struct {
	baseRF         *RandomForestRegressor
	fineTunedRFs   map[string]*RandomForestRegressor
	caveSpecificRF map[int]*RandomForestRegressor
	fineTuneData   map[string][]Sample
	caveData       map[int][]Sample
}

var rockTypeFactor = map[string]float64{
	"砂砾岩": 1.8,
	"砂岩":   1.4,
	"石灰岩": 1.0,
	"花岗岩": 0.6,
}

var climateZoneFactor = map[string]float64{
	"ARID":       1.3,
	"SEMI_ARID":  1.1,
	"SEMI_HUMID": 1.0,
	"HUMID":      1.2,
	"COLD":       1.4,
}

var caveProvinceClimate = map[string]string{
	"甘肃": "SEMI_ARID",
	"山西": "SEMI_HUMID",
	"河南": "SEMI_HUMID",
	"重庆": "HUMID",
	"新疆": "ARID",
	"河北": "SEMI_HUMID",
}

var caveTypeFactor = map[string]float64{
	"石窟寺": 1.0,
	"摩崖造像": 1.15,
	"千佛洞": 1.08,
}

func NewWeatheringPredictor() *WeatheringPredictor {
	p := &WeatheringPredictor{
		baseRF:         NewRandomForestRegressor(50, 10, 5),
		fineTunedRFs:   make(map[string]*RandomForestRegressor),
		caveSpecificRF: make(map[int]*RandomForestRegressor),
		fineTuneData:   make(map[string][]Sample),
		caveData:       make(map[int][]Sample),
	}
	p.trainWithDomainData()
	return p
}

func (p *WeatheringPredictor) trainWithDomainData() {
	samples := generateTrainingSamples()
	p.baseRF.Train(samples)

	for _, rockType := range []string{"砂砾岩", "砂岩", "石灰岩", "花岗岩"} {
		rockSamples := generateRockSpecificSamples(rockType, 600)
		fineRF := NewRandomForestRegressor(30, 8, 4)
		fineRF.Train(rockSamples)
		p.fineTunedRFs[rockType] = fineRF
		p.fineTuneData[rockType] = rockSamples
	}
}

func generateRockSpecificSamples(rockType string, count int) []Sample {
	rockFactor := rockTypeFactor[rockType]
	var samples []Sample
	for i := 0; i < count; i++ {
		temp := -15 + randFloat()*55
		hum := 10 + randFloat()*88
		tempRange := randFloat() * 25
		rainfall := randFloat() * 180
		climateIdx := randFloat()
		climateEnc := encodeClimateZoneByIndex(climateIdx)

		rate := computeWeatheringRate(rockFactor, temp, hum, tempRange, rainfall, climateIdx)

		rockEnc := encodeRockType(rockType)
		features := append(rockEnc, temp, hum, tempRange, rainfall, temp*hum/100)
		features = append(features, climateEnc...)

		samples = append(samples, Sample{Features: features, Target: rate})
	}
	return samples
}

func computeWeatheringRate(rockFactor, temp, hum, tempRange, rainfall, climateFactor float64) float64 {
	tempStress := math.Pow(math.Abs(temp-15), 1.5) * 0.002
	humStress := math.Pow(hum-60, 2) * 0.0005
	if hum < 30 {
		humStress += (30 - hum) * 0.01
	}
	rangeStress := tempRange * 0.008
	rainStress := rainfall * 0.0005
	interaction := 0.0
	if temp > 30 && hum > 75 {
		interaction = 0.05
	}
	if temp < -5 && hum > 60 {
		interaction += 0.08
	}
	climateMod := 1.0 + climateFactor*0.15
	rate := (tempStress + humStress + rangeStress + rainStress + interaction) * rockFactor * climateMod
	return math.Max(0.001, rate*0.01)
}

func encodeClimateZoneByIndex(idx float64) []float64 {
	enc := make([]float64, 4)
	zone := int(idx * 4)
	if zone >= 4 {
		zone = 3
	}
	enc[zone] = 1
	return enc
}

func (p *WeatheringPredictor) FineTuneForCave(caveID int, province, rockType string, observedData []Sample) {
	climateZone := caveProvinceClimate[province]
	if climateZone == "" {
		climateZone = "SEMI_HUMID"
	}

	var augmented []Sample
	augmented = append(augmented, observedData...)

	climateFactor := climateZoneFactor[climateZone]
	for i := 0; i < 300; i++ {
		temp := -15 + randFloat()*55
		hum := 10 + randFloat()*88
		tempRange := randFloat() * 25
		rainfall := randFloat() * 180
		rockFactor := rockTypeFactor[rockType]
		rate := computeWeatheringRate(rockFactor, temp, hum, tempRange, rainfall, climateFactor)

		rockEnc := encodeRockType(rockType)
		climateEnc := encodeClimateZone(climateZone)
		caveEnc := encodeCaveType(province)
		features := append(rockEnc, temp, hum, tempRange, rainfall, temp*hum/100)
		features = append(features, climateEnc...)
		features = append(features, caveEnc...)

		augmented = append(augmented, Sample{Features: features, Target: rate})
	}

	fineRF := NewRandomForestRegressor(40, 9, 4)
	fineRF.Train(augmented)
	p.caveSpecificRF[caveID] = fineRF
	p.caveData[caveID] = observedData
	logFineTune(caveID, province, rockType, len(observedData), len(augmented))
}

func logFineTune(caveID int, province, rockType string, observed, augmented int) {
	_ = caveID
	_ = province
	_ = rockType
	_ = observed
	_ = augmented
}

func (p *WeatheringPredictor) AddObservedSample(caveID int, province, rockType string, sample Sample) {
	climateZone := caveProvinceClimate[province]
	climateEnc := encodeClimateZone(climateZone)
	caveEnc := encodeCaveType(province)
	enhancedFeatures := make([]float64, 0, len(sample.Features)+len(climateEnc)+len(caveEnc))
	enhancedFeatures = append(enhancedFeatures, sample.Features...)
	enhancedFeatures = append(enhancedFeatures, climateEnc...)
	enhancedFeatures = append(enhancedFeatures, caveEnc...)
	enhancedSample := Sample{Features: enhancedFeatures, Target: sample.Target}

	p.caveData[caveID] = append(p.caveData[caveID], enhancedSample)
	if len(p.caveData[caveID]) >= 50 {
		p.FineTuneForCave(caveID, province, rockType, p.caveData[caveID])
		p.caveData[caveID] = nil
	}
}

func generateTrainingSamples() []Sample {
	var samples []Sample
	rockTypes := []string{"砂砾岩", "砂岩", "石灰岩", "花岗岩"}

	for _, rockType := range rockTypes {
		rockFactor := rockTypeFactor[rockType]
		for temp := -10.0; temp <= 40.0; temp += 2.0 {
			for hum := 20.0; hum <= 95.0; hum += 5.0 {
				for tempRange := 0.0; tempRange <= 20.0; tempRange += 5.0 {
					for rainfall := 0.0; rainfall <= 150.0; rainfall += 25.0 {
						rate := computeWeatheringRate(rockFactor, temp, hum, tempRange, rainfall, 0.5)

						rockEnc := encodeRockType(rockType)
						climateEnc := encodeClimateZoneByIndex(0.5)
						features := append(rockEnc, temp, hum, tempRange, rainfall, temp*hum/100)
						features = append(features, climateEnc...)

						samples = append(samples, Sample{
							Features: features,
							Target:   rate,
						})
					}
				}
			}
		}
	}
	return samples
}

func encodeRockType(rockType string) []float64 {
	enc := make([]float64, 4)
	switch rockType {
	case "砂砾岩":
		enc[0] = 1
	case "砂岩":
		enc[1] = 1
	case "石灰岩":
		enc[2] = 1
	case "花岗岩":
		enc[3] = 1
	default:
		enc[1] = 0.5
		enc[2] = 0.5
	}
	return enc
}

func encodeClimateZone(zone string) []float64 {
	enc := make([]float64, 4)
	switch zone {
	case "ARID":
		enc[0] = 1
	case "SEMI_ARID":
		enc[1] = 1
	case "SEMI_HUMID":
		enc[2] = 1
	case "HUMID":
		enc[3] = 1
	default:
		enc[2] = 1
	}
	return enc
}

func encodeCaveType(province string) []float64 {
	enc := make([]float64, 3)
	switch province {
	case "甘肃", "新疆":
		enc[0] = 1
	case "山西", "河南", "河北":
		enc[1] = 1
	case "重庆":
		enc[2] = 1
	default:
		enc[1] = 0.5
	}
	return enc
}

func randFloat() float64 {
	return float64(uint64(time.Now().UnixNano())%1000000) / 1000000.0
}

func (p *WeatheringPredictor) Predict(rockType string, temperature, humidity, tempRange, rainfall float64) (float64, float64) {
	rockEnc := encodeRockType(rockType)
	climateEnc := encodeClimateZoneByIndex(0.5)
	features := append(rockEnc, temperature, humidity, tempRange, rainfall, temperature*humidity/100)
	features = append(features, climateEnc...)

	basePred := p.baseRF.Predict(features)

	rockPred := basePred
	if fineRF, ok := p.fineTunedRFs[rockType]; ok {
		rockSamples := generateRockSpecificSamples(rockType, 50)
		_ = rockSamples
		rockPred = fineRF.Predict(features)
	}

	predRate := basePred*0.4 + rockPred*0.6
	predRate = math.Max(0.0001, predRate)

	baseConfidence := 0.75
	if tempRange > 15 {
		baseConfidence -= 0.1
	}
	if rainfall > 100 {
		baseConfidence -= 0.05
	}
	if _, ok := p.fineTunedRFs[rockType]; ok {
		baseConfidence += 0.08
	}
	confidence := math.Min(0.95, math.Max(0.5, baseConfidence))

	return predRate, confidence
}

func (p *WeatheringPredictor) PredictForCave(caveID int, province, rockType string, temperature, humidity, tempRange, rainfall float64) (float64, float64) {
	rockEnc := encodeRockType(rockType)
	climateZone := caveProvinceClimate[province]
	if climateZone == "" {
		climateZone = "SEMI_HUMID"
	}
	climateEnc := encodeClimateZone(climateZone)
	caveEnc := encodeCaveType(province)
	features := append(rockEnc, temperature, humidity, tempRange, rainfall, temperature*humidity/100)
	features = append(features, climateEnc...)
	features = append(features, caveEnc...)

	basePred, baseConf := p.Predict(rockType, temperature, humidity, tempRange, rainfall)

	if caveRF, ok := p.caveSpecificRF[caveID]; ok {
		cavePred := caveRF.Predict(features)
		predRate := basePred*0.3 + cavePred*0.7
		predRate = math.Max(0.0001, predRate)
		confidence := math.Min(0.97, baseConf+0.12)
		return predRate, confidence
	}

	return basePred, baseConf
}

func GenerateProtectionSuggestions(predictedRate, currentRate, temperature, humidity float64, rockType string) []string {
	var suggestions []string

	rateRatio := predictedRate / math.Max(0.0001, currentRate)
	var riskLevel string
	switch {
	case predictedRate > 0.05:
		riskLevel = "高风险"
	case predictedRate > 0.02:
		riskLevel = "中高风险"
	case predictedRate > 0.01:
		riskLevel = "中等风险"
	default:
		riskLevel = "低风险"
	}
	suggestions = append(suggestions, "预测风化速率等级: "+riskLevel)

	if rateRatio > 1.5 {
		suggestions = append(suggestions, "预警：当前微环境条件下风化速率将显著加快，建议及时采取防护措施")
	}

	if temperature < 0 && humidity > 60 {
		suggestions = append(suggestions, "冻融循环风险：低温高湿环境下可能发生冻融破坏，建议保温处理")
	}
	if temperature > 30 && humidity > 75 {
		suggestions = append(suggestions, "湿热风险：高温高湿加速化学风化，建议通风除湿")
	}
	if humidity < 30 {
		suggestions = append(suggestions, "干燥风化：低湿度环境下表面粉化风险增加，建议适度保湿")
	}

	switch rockType {
	case "砂砾岩":
		suggestions = append(suggestions, "砂砾岩建议：优先选用纳米石灰加固，配合有机硅防水")
	case "砂岩":
		suggestions = append(suggestions, "砂岩建议：渗透性有机硅防护，注意保持透气性")
	case "石灰岩":
		suggestions = append(suggestions, "石灰岩建议：纳米石灰+纳米TiO2涂层，兼顾保护与自清洁")
	}

	return suggestions
}
