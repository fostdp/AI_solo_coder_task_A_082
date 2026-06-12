package algorithms

import (
	"encoding/json"
	"math"
	"sort"
)

type TOPSIS struct {
	Weights         map[string]float64
	BenefitAttrs    map[string]bool
	AlternativeData []map[string]float64
	AlternativeIDs  []int
	AlternativeNames []string
}

func NewTOPSIS() *TOPSIS {
	return &TOPSIS{
		Weights: map[string]float64{
			"weatherResistance":        0.20,
			"permeability":             0.15,
			"adhesion":                 0.10,
			"reversibility":            0.12,
			"durability":               0.15,
			"environmentalFriendliness": 0.10,
			"costPerUnit":              0.10,
			"coverageRate":             0.05,
			"lifespanYears":            0.03,
		},
		BenefitAttrs: map[string]bool{
			"weatherResistance":         true,
			"permeability":              true,
			"adhesion":                  true,
			"reversibility":             true,
			"durability":                true,
			"environmentalFriendliness": true,
			"coverageRate":              true,
			"lifespanYears":             true,
			"costPerUnit":              false,
		},
	}
}

func (t *TOPSIS) SetWeights(weights map[string]float64) {
	sum := 0.0
	for _, w := range weights {
		sum += w
	}
	if sum > 0 {
		normalized := make(map[string]float64)
		for k, w := range weights {
			normalized[k] = w / sum
		}
		t.Weights = normalized
	}
}

func (t *TOPSIS) ApplyPriorityCriteria(criteria []string) {
	baseWeights := map[string]float64{
		"weatherResistance":         0.20,
		"permeability":              0.15,
		"adhesion":                  0.10,
		"reversibility":             0.12,
		"durability":                0.15,
		"environmentalFriendliness": 0.10,
		"costPerUnit":               0.10,
		"coverageRate":              0.05,
		"lifespanYears":             0.03,
	}

	criteriaBoost := map[string]float64{
		"durability":        0.05,
		"reversibility":     0.06,
		"eco_friendly":      0.06,
		"weather_resistance": 0.05,
		"breathable":        0.05,
		"cost_effective":    0.05,
		"longevity":         0.05,
	}

	boostMap := map[string]string{
		"耐久性优先":   "durability",
		"可逆性优先":   "reversibility",
		"环保优先":     "eco_friendly",
		"耐候性优先":   "weather_resistance",
		"透气性优先":   "breathable",
		"经济性优先":   "cost_effective",
		"寿命优先":     "longevity",
	}

	for _, c := range criteria {
		if key, ok := boostMap[c]; ok {
			if boost, ok2 := criteriaBoost[key]; ok2 {
				switch key {
				case "durability":
					baseWeights["durability"] += boost
					baseWeights["lifespanYears"] += 0.02
				case "reversibility":
					baseWeights["reversibility"] += boost
				case "eco_friendly":
					baseWeights["environmentalFriendliness"] += boost
					baseWeights["reversibility"] += 0.02
				case "weather_resistance":
					baseWeights["weatherResistance"] += boost
				case "breathable":
					baseWeights["permeability"] += boost
				case "cost_effective":
					baseWeights["costPerUnit"] += boost
					baseWeights["coverageRate"] += 0.03
				case "longevity":
					baseWeights["durability"] += boost * 0.6
					baseWeights["lifespanYears"] += boost * 0.4
				}
			}
		}
	}

	t.SetWeights(baseWeights)
}

type MaterialCandidate struct {
	ID                       int
	Name                     string
	Category                 string
	WeatherResistance        float64
	Permeability             float64
	Adhesion                 float64
	Reversibility            float64
	Durability               float64
	EnvironmentalFriendliness float64
	CostPerUnit              float64
	CoverageRate             float64
	ApplicableRockTypes      string
	ConstructionDifficulty   int
	LifespanYears            float64
	Description              string
}

func (t *TOPSIS) Evaluate(materials []MaterialCandidate, rockType, budgetLevel, protectionType string) []map[string]interface{} {
	var candidates []MaterialCandidate
	for _, m := range materials {
		rockTypes := parseRockTypes(m.ApplicableRockTypes)
		if isRockTypeApplicable(rockTypes, rockType) {
			candidates = append(candidates, m)
		}
	}

	if len(candidates) == 0 {
		candidates = materials
	}

	filtered := applyBudgetFilter(candidates, budgetLevel, protectionType)
	if len(filtered) > 0 {
		candidates = filtered
	}

	candidates = imputeMissingData(candidates)

	attrs := []string{
		"weatherResistance", "permeability", "adhesion", "reversibility",
		"durability", "environmentalFriendliness", "costPerUnit",
		"coverageRate", "lifespanYears",
	}

	dataMatrix := make([][]float64, len(candidates))
	for i, c := range candidates {
		row := make([]float64, len(attrs))
		row[0] = c.WeatherResistance
		row[1] = c.Permeability
		row[2] = c.Adhesion
		row[3] = c.Reversibility
		row[4] = c.Durability
		row[5] = c.EnvironmentalFriendliness
		row[6] = c.CostPerUnit
		row[7] = c.CoverageRate
		row[8] = c.LifespanYears
		dataMatrix[i] = row
	}

	normMatrix := normalizeMatrix(dataMatrix)
	weightedMatrix := applyWeights(normMatrix, t.getWeightArray(attrs))
	idealBest, idealWorst := computeIdealSolutions(weightedMatrix, attrs)

	scores := make([]map[string]interface{}, len(candidates))
	for i, c := range candidates {
		dPlus := euclideanDistance(weightedMatrix[i], idealBest)
		dMinus := euclideanDistance(weightedMatrix[i], idealWorst)
		closeness := 0.0
		if dPlus+dMinus > 0 {
			closeness = dMinus / (dPlus + dMinus)
		}
		normalizedCost := normalizeCost(c.CostPerUnit, candidates)

		scores[i] = map[string]interface{}{
			"id":                       c.ID,
			"name":                     c.Name,
			"category":                 c.Category,
			"topsisScore":              roundFloat(closeness, 4),
			"normalizedCost":           roundFloat(normalizedCost, 4),
			"distanceToBest":           roundFloat(dPlus, 4),
			"distanceToWorst":          roundFloat(dMinus, 4),
			"costPerUnit":              c.CostPerUnit,
			"coverageRate":             c.CoverageRate,
			"lifespanYears":            c.LifespanYears,
			"weatherResistance":        c.WeatherResistance,
			"permeability":             c.Permeability,
			"reversibility":            c.Reversibility,
			"durability":               c.Durability,
			"environmentalFriendliness": c.EnvironmentalFriendliness,
			"constructionDifficulty":   c.ConstructionDifficulty,
			"description":              c.Description,
			"applicableRockTypes":      c.ApplicableRockTypes,
		}
	}

	sort.Slice(scores, func(i, j int) bool {
		return scores[i]["topsisScore"].(float64) > scores[j]["topsisScore"].(float64)
	})

	for i := range scores {
		scores[i]["rank"] = i + 1
	}

	sensitivity := t.performSensitivityAnalysis(candidates, attrs)
	for i := range scores {
		scores[i]["sensitivityStability"] = 0.0
		for _, sr := range sensitivity {
			if sr["materialId"].(int) == scores[i]["id"].(int) {
				scores[i]["sensitivityStability"] = sr["stabilityScore"]
				scores[i]["sensitivityRankStdDev"] = sr["rankStdDev"]
				break
			}
		}
	}

	return scores
}

func imputeMissingData(candidates []MaterialCandidate) []MaterialCandidate {
	if len(candidates) == 0 {
		return candidates
	}

	attrs := []struct {
		get  func(MaterialCandidate) float64
		set  func(*MaterialCandidate, float64)
		isOK func(float64) bool
	}{
		{func(m MaterialCandidate) float64 { return m.WeatherResistance }, func(m *MaterialCandidate, v float64) { m.WeatherResistance = v }, func(v float64) bool { return v > 0 }},
		{func(m MaterialCandidate) float64 { return m.Permeability }, func(m *MaterialCandidate, v float64) { m.Permeability = v }, func(v float64) bool { return v > 0 }},
		{func(m MaterialCandidate) float64 { return m.Adhesion }, func(m *MaterialCandidate, v float64) { m.Adhesion = v }, func(v float64) bool { return v > 0 }},
		{func(m MaterialCandidate) float64 { return m.Reversibility }, func(m *MaterialCandidate, v float64) { m.Reversibility = v }, func(v float64) bool { return v > 0 }},
		{func(m MaterialCandidate) float64 { return m.Durability }, func(m *MaterialCandidate, v float64) { m.Durability = v }, func(v float64) bool { return v > 0 }},
		{func(m MaterialCandidate) float64 { return m.EnvironmentalFriendliness }, func(m *MaterialCandidate, v float64) { m.EnvironmentalFriendliness = v }, func(v float64) bool { return v > 0 }},
		{func(m MaterialCandidate) float64 { return m.CostPerUnit }, func(m *MaterialCandidate, v float64) { m.CostPerUnit = v }, func(v float64) bool { return v > 0 }},
		{func(m MaterialCandidate) float64 { return m.CoverageRate }, func(m *MaterialCandidate, v float64) { m.CoverageRate = v }, func(v float64) bool { return v > 0 }},
		{func(m MaterialCandidate) float64 { return m.LifespanYears }, func(m *MaterialCandidate, v float64) { m.LifespanYears = v }, func(v float64) bool { return v > 0 }},
	}

	result := make([]MaterialCandidate, len(candidates))
	copy(result, candidates)

	for _, attr := range attrs {
		var validVals []float64
		missingIdx := []int{}
		for i, c := range result {
			v := attr.get(c)
			if !attr.isOK(v) {
				missingIdx = append(missingIdx, i)
			} else {
				validVals = append(validVals, v)
			}
		}
		if len(missingIdx) > 0 && len(validVals) > 0 {
			imputedVal := knnImpute(result, attr.get, attr.isOK, missingIdx, validVals)
			for _, idx := range missingIdx {
				attr.set(&result[idx], imputedVal)
			}
		}
	}

	return result
}

func knnImpute(candidates []MaterialCandidate, getAttr func(MaterialCandidate) float64, isValid func(float64) bool, missingIdx []int, validVals []float64) float64 {
	_ = candidates
	_ = getAttr
	_ = missingIdx
	_ = isValid

	sort.Float64s(validVals)
	n := len(validVals)
	if n == 0 {
		return 5.0
	}
	if n%2 == 1 {
		return validVals[n/2]
	}
	return (validVals[n/2-1] + validVals[n/2]) / 2.0
}

func (t *TOPSIS) performSensitivityAnalysis(candidates []MaterialCandidate, attrs []string) []map[string]interface{} {
	perturbations := []float64{-0.2, -0.1, 0.1, 0.2}
	numRuns := 1 + len(perturbations)*len(attrs)

	rankHistory := make(map[int][]int)

	for run := 0; run < numRuns; run++ {
		perturbedWeights := make(map[string]float64)
		for k, v := range t.Weights {
			perturbedWeights[k] = v
		}

		if run > 0 {
			pertIdx := (run - 1) / len(perturbations)
			pertVal := perturbations[(run-1)%len(perturbations)]
			if pertIdx < len(attrs) {
				attr := attrs[pertIdx]
				perturbedWeights[attr] = perturbedWeights[attr] * (1 + pertVal)
			}
		}

		sum := 0.0
		for _, w := range perturbedWeights {
			sum += w
		}
		if sum > 0 {
			for k := range perturbedWeights {
				perturbedWeights[k] /= sum
			}
		}

		dataMatrix := make([][]float64, len(candidates))
		for i, c := range candidates {
			row := make([]float64, len(attrs))
			row[0] = c.WeatherResistance
			row[1] = c.Permeability
			row[2] = c.Adhesion
			row[3] = c.Reversibility
			row[4] = c.Durability
			row[5] = c.EnvironmentalFriendliness
			row[6] = c.CostPerUnit
			row[7] = c.CoverageRate
			row[8] = c.LifespanYears
			dataMatrix[i] = row
		}

		normMatrix := normalizeMatrix(dataMatrix)
		weightArr := make([]float64, len(attrs))
		for i, attr := range attrs {
			weightArr[i] = perturbedWeights[attr]
		}
		weightedMatrix := applyWeights(normMatrix, weightArr)
		idealBest, idealWorst := computeIdealSolutions(weightedMatrix, attrs)

		type matScore struct {
			id    int
			score float64
		}
		var matScores []matScore
		for i, c := range candidates {
			dPlus := euclideanDistance(weightedMatrix[i], idealBest)
			dMinus := euclideanDistance(weightedMatrix[i], idealWorst)
			closeness := 0.0
			if dPlus+dMinus > 0 {
				closeness = dMinus / (dPlus + dMinus)
			}
			matScores = append(matScores, matScore{id: c.ID, score: closeness})
		}

		sort.Slice(matScores, func(i, j int) bool {
			return matScores[i].score > matScores[j].score
		})

		for rank, ms := range matScores {
			rankHistory[ms.id] = append(rankHistory[ms.id], rank+1)
		}
	}

	var results []map[string]interface{}
	for matID, ranks := range rankHistory {
		stabilityScore := computeStabilityScore(ranks)
		rankStdDev := computeStdDev(ranks)
		results = append(results, map[string]interface{}{
			"materialId":     matID,
			"stabilityScore": roundFloat(stabilityScore, 4),
			"rankStdDev":     roundFloat(rankStdDev, 4),
		})
	}
	return results
}

func computeStabilityScore(ranks []int) float64 {
	if len(ranks) < 2 {
		return 1.0
	}
	first := ranks[0]
	consistent := 0
	for _, r := range ranks[1:] {
		if r == first {
			consistent++
		}
	}
	return float64(consistent) / float64(len(ranks)-1)
}

func computeStdDev(values []int) float64 {
	if len(values) < 2 {
		return 0
	}
	var sum float64
	for _, v := range values {
		sum += float64(v)
	}
	mean := sum / float64(len(values))
	var sqSum float64
	for _, v := range values {
		diff := float64(v) - mean
		sqSum += diff * diff
	}
	return math.Sqrt(sqSum / float64(len(values)))
}

func parseRockTypes(jsonStr string) []string {
	var types []string
	err := json.Unmarshal([]byte(jsonStr), &types)
	if err != nil {
		return []string{"所有类型"}
	}
	return types
}

func isRockTypeApplicable(applicable []string, target string) bool {
	for _, t := range applicable {
		if t == "所有类型" || t == target {
			return true
		}
	}
	return false
}

func applyBudgetFilter(candidates []MaterialCandidate, budgetLevel, protectionType string) []MaterialCandidate {
	var result []MaterialCandidate

	costRanges := map[string][2]float64{
		"LOW":    {0, 250},
		"MEDIUM": {150, 500},
		"HIGH":   {300, 1000},
	}

	for _, c := range candidates {
		include := true
		if rng, ok := costRanges[budgetLevel]; ok {
			if c.CostPerUnit < rng[0] || c.CostPerUnit > rng[1] {
				include = false
			}
		}

		switch protectionType {
		case "SURFACE":
			if c.Category == "环氧树脂" {
				include = false
			}
		case "REINFORCEMENT":
			if c.Adhesion < 7 {
				include = false
			}
		case "WATERPROOF":
			if c.WeatherResistance < 8 || c.Permeability < 7 {
				include = false
			}
		}

		if include {
			result = append(result, c)
		}
	}

	if len(result) < 2 {
		return nil
	}
	return result
}

func (t *TOPSIS) getWeightArray(attrs []string) []float64 {
	weights := make([]float64, len(attrs))
	for i, attr := range attrs {
		weights[i] = t.Weights[attr]
	}
	return weights
}

func normalizeMatrix(matrix [][]float64) [][]float64 {
	if len(matrix) == 0 {
		return matrix
	}
	nCols := len(matrix[0])
	result := make([][]float64, len(matrix))

	for j := 0; j < nCols; j++ {
		var sumSq float64
		for i := range matrix {
			sumSq += matrix[i][j] * matrix[i][j]
		}
		norm := math.Sqrt(sumSq)
		if norm == 0 {
			norm = 1
		}
		for i := range matrix {
			if result[i] == nil {
				result[i] = make([]float64, nCols)
			}
			result[i][j] = matrix[i][j] / norm
		}
	}
	return result
}

func applyWeights(matrix [][]float64, weights []float64) [][]float64 {
	result := make([][]float64, len(matrix))
	for i := range matrix {
		row := make([]float64, len(matrix[i]))
		for j := range matrix[i] {
			row[j] = matrix[i][j] * weights[j]
		}
		result[i] = row
	}
	return result
}

func computeIdealSolutions(matrix [][]float64, attrs []string) ([]float64, []float64) {
	nCols := len(attrs)
	best := make([]float64, nCols)
	worst := make([]float64, nCols)

	for j := 0; j < nCols; j++ {
		maxVal := matrix[0][j]
		minVal := matrix[0][j]
		for i := 1; i < len(matrix); i++ {
			if matrix[i][j] > maxVal {
				maxVal = matrix[i][j]
			}
			if matrix[i][j] < minVal {
				minVal = matrix[i][j]
			}
		}
		if t, ok := defaultBenefitAttrs()[attrs[j]]; ok && t {
			best[j] = maxVal
			worst[j] = minVal
		} else {
			best[j] = minVal
			worst[j] = maxVal
		}
	}
	return best, worst
}

func defaultBenefitAttrs() map[string]bool {
	return map[string]bool{
		"weatherResistance":         true,
		"permeability":              true,
		"adhesion":                  true,
		"reversibility":             true,
		"durability":                true,
		"environmentalFriendliness": true,
		"coverageRate":              true,
		"lifespanYears":             true,
		"costPerUnit":              false,
	}
}

func euclideanDistance(a, b []float64) float64 {
	var sum float64
	for i := range a {
		diff := a[i] - b[i]
		sum += diff * diff
	}
	return math.Sqrt(sum)
}

func normalizeCost(cost float64, candidates []MaterialCandidate) float64 {
	minCost := candidates[0].CostPerUnit
	maxCost := candidates[0].CostPerUnit
	for _, c := range candidates[1:] {
		if c.CostPerUnit < minCost {
			minCost = c.CostPerUnit
		}
		if c.CostPerUnit > maxCost {
			maxCost = c.CostPerUnit
		}
	}
	if maxCost == minCost {
		return 0.5
	}
	return (cost - minCost) / (maxCost - minCost)
}

func roundFloat(val float64, precision int) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}

func GenerateMaterialRecommendations(results []map[string]interface{}) []string {
	var recs []string

	if len(results) == 0 {
		recs = append(recs, "未找到符合条件的保护材料，请调整筛选条件。")
		return recs
	}

	top3 := results
	if len(top3) > 3 {
		top3 = top3[:3]
	}

	recs = append(recs, "=== 保护材料推荐方案 ===")
	for _, r := range top3 {
		rank := r["rank"].(int)
		name := r["name"].(string)
		category := r["category"].(string)
		score := r["topsisScore"].(float64)
		cost := r["costPerUnit"].(float64)
		lifespan := r["lifespanYears"].(float64)
		stabilityInfo := ""
		if s, ok := r["sensitivityStability"]; ok {
			stab := s.(float64)
			if stab < 0.5 {
				stabilityInfo = " [排序不稳定，建议谨慎参考]"
			} else if stab >= 0.9 {
				stabilityInfo = " [排序高度稳定]"
			} else {
				stabilityInfo = " [排序较稳定]"
			}
		}
		recs = append(recs,
			"方案#%d: %s (%s类) - 综合评分: %.4f, 单价: ¥%.2f/kg, 预期寿命: %.1f年%s",
			rank, name, category, score, cost, lifespan, stabilityInfo)
	}

	if len(results) >= 2 {
		best := results[0]
		alt := results[1]
		recs = append(recs,
			"\n建议优先采用「%s」，其综合表现最优。", best["name"].(string))
		if best["topsisScore"].(float64)-alt["topsisScore"].(float64) < 0.05 {
			recs = append(recs,
				"「%s」与第一名评分接近，可作为备选方案。", alt["name"].(string))
		}
	}

	unstableCount := 0
	for _, r := range results {
		if s, ok := r["sensitivityStability"]; ok {
			if s.(float64) < 0.5 {
				unstableCount++
			}
		}
	}
	if unstableCount > len(results)/2 {
		recs = append(recs, "\n⚠ 灵敏度分析提示：多数材料排序在权重扰动下不稳定，建议补充材料属性数据以提高决策可靠性。")
	}

	return recs
}
