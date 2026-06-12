package algorithms

import (
	"math"
	"sort"
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
	rf *RandomForestRegressor
}

var rockTypeFactor = map[string]float64{
	"砂砾岩": 1.8,
	"砂岩":   1.4,
	"石灰岩": 1.0,
	"花岗岩": 0.6,
}

func NewWeatheringPredictor() *WeatheringPredictor {
	p := &WeatheringPredictor{
		rf: NewRandomForestRegressor(50, 10, 5),
	}
	p.trainWithDomainData()
	return p
}

func (p *WeatheringPredictor) trainWithDomainData() {
	samples := generateTrainingSamples()
	p.rf.Train(samples)
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

						rate := (tempStress + humStress + rangeStress + rainStress + interaction) * rockFactor
						rate = math.Max(0.001, rate*0.01)

						rockEnc := encodeRockType(rockType)
						features := append(rockEnc, temp, hum, tempRange, rainfall, temp*hum/100)

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

func (p *WeatheringPredictor) Predict(rockType string, temperature, humidity, tempRange, rainfall float64) (float64, float64) {
	rockEnc := encodeRockType(rockType)
	features := append(rockEnc, temperature, humidity, tempRange, rainfall, temperature*humidity/100)

	predRate := p.rf.Predict(features)
	predRate = math.Max(0.0001, predRate)

	baseConfidence := 0.75
	if tempRange > 15 {
		baseConfidence -= 0.1
	}
	if rainfall > 100 {
		baseConfidence -= 0.05
	}
	confidence := math.Min(0.95, math.Max(0.5, baseConfidence))

	return predRate, confidence
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
