package services

import (
	"errors"
	"math"
	"sort"
	"sync"

	"grotto-monitor/backend/models"
)

type TOPSISService struct {
	imputationCache map[string][]float64
	cacheMu         sync.RWMutex
}

func NewTOPSISService() *TOPSISService {
	return &TOPSISService{
		imputationCache: make(map[string][]float64),
	}
}

var defaultWeights = map[string]float64{
	"penetration_depth":     0.15,
	"breathability":         0.20,
	"weathering_resistance": 0.25,
	"compatibility":         0.20,
	"cost":                  0.10,
	"durability_years":      0.10,
}

var attributeOrder = []string{
	"penetration_depth",
	"breathability",
	"weathering_resistance",
	"compatibility",
	"cost",
	"durability_years",
}

var attributeDisplayName = map[string]string{
	"penetration_depth":     "渗透深度",
	"breathability":         "透气性",
	"weathering_resistance": "耐候性",
	"compatibility":         "兼容性",
	"cost":                  "成本",
	"durability_years":      "耐用年限",
}

var nonBenefitAttributes = map[string]bool{
	"cost": true,
}

var attributeRanges = map[string][2]float64{
	"penetration_depth":     {0.5, 5.0},
	"breathability":         {50.0, 95.0},
	"weathering_resistance": {60.0, 98.0},
	"compatibility":         {50.0, 95.0},
	"cost":                  {20.0, 500.0},
	"durability_years":      {3.0, 30.0},
}

type materialWithFeatures struct {
	material models.ProtectionMaterial
	features []float64
	missing  []bool
}

func getAttributeValue(m models.ProtectionMaterial, attr string) float64 {
	switch attr {
	case "penetration_depth":
		return m.PenetrationDepth
	case "breathability":
		return m.Breathability
	case "weathering_resistance":
		return m.WeatheringResistance
	case "compatibility":
		return m.Compatibility
	case "cost":
		return m.Cost
	case "durability_years":
		return m.DurabilityYears
	default:
		return 0
	}
}

func setAttributeValue(m *models.ProtectionMaterial, attr string, value float64) {
	switch attr {
	case "penetration_depth":
		m.PenetrationDepth = value
	case "breathability":
		m.Breathability = value
	case "weathering_resistance":
		m.WeatheringResistance = value
	case "compatibility":
		m.Compatibility = value
	case "cost":
		m.Cost = value
	case "durability_years":
		m.DurabilityYears = value
	}
}

func isValidValue(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0) && v >= 0
}

func extractFeatures(m models.ProtectionMaterial) ([]float64, []bool) {
	features := make([]float64, len(attributeOrder))
	missing := make([]bool, len(attributeOrder))

	for j, attr := range attributeOrder {
		val := getAttributeValue(m, attr)
		if isValidValue(val) && val > 0 {
			features[j] = val
			missing[j] = false
		} else {
			features[j] = 0
			missing[j] = true
		}
	}
	return features, missing
}

func euclideanDistanceNaN(a, b []float64, aMissing, bMissing []bool) (float64, int) {
	var sum float64
	validCount := 0

	for j := 0; j < len(a); j++ {
		if aMissing[j] || bMissing[j] {
			continue
		}
		diff := a[j] - b[j]
		sum += diff * diff
		validCount++
	}

	if validCount == 0 {
		return math.Inf(1), 0
	}
	adjusted := sum * float64(len(a)) / float64(validCount)
	return math.Sqrt(adjusted), validCount
}

func (svc *TOPSISService) knnImpute(materials []models.ProtectionMaterial, k int) []models.ProtectionMaterial {
	imputed := make([]models.ProtectionMaterial, len(materials))
	dataWithFeatures := make([]materialWithFeatures, len(materials))

	for i, m := range materials {
		imputed[i] = m
		features, missing := extractFeatures(m)
		dataWithFeatures[i] = materialWithFeatures{
			material: m,
			features: features,
			missing:  missing,
		}
	}

	for i := range dataWithFeatures {
		hasMissing := false
		for _, m := range dataWithFeatures[i].missing {
			if m {
				hasMissing = true
				break
			}
		}
		if !hasMissing {
			continue
		}

		type neighborInfo struct {
			idx    int
			dist   float64
			shared int
		}
		var neighbors []neighborInfo

		for j := range dataWithFeatures {
			if i == j {
				continue
			}
			dist, shared := euclideanDistanceNaN(
				dataWithFeatures[i].features,
				dataWithFeatures[j].features,
				dataWithFeatures[i].missing,
				dataWithFeatures[j].missing,
			)
			if shared > 0 {
				neighbors = append(neighbors, neighborInfo{idx: j, dist: dist, shared: shared})
			}
		}

		sort.Slice(neighbors, func(a, b int) bool {
			if neighbors[a].shared != neighbors[b].shared {
				return neighbors[a].shared > neighbors[b].shared
			}
			return neighbors[a].dist < neighbors[b].dist
		})

		actualK := k
		if len(neighbors) < actualK {
			actualK = len(neighbors)
		}
		if actualK == 0 {
			continue
		}
		selectedNeighbors := neighbors[:actualK]

		for attrIdx, isMissing := range dataWithFeatures[i].missing {
			if !isMissing {
				continue
			}

			var weightedSum float64
			var totalWeight float64

			for _, nb := range selectedNeighbors {
				if dataWithFeatures[nb.idx].missing[attrIdx] {
					continue
				}

				weight := 1.0 / (nb.dist + 1e-8)
				weight *= float64(nb.shared) / float64(len(attributeOrder))

				weightedSum += dataWithFeatures[nb.idx].features[attrIdx] * weight
				totalWeight += weight
			}

			if totalWeight > 0 {
				imputedVal := weightedSum / totalWeight
				attr := attributeOrder[attrIdx]
				rng := attributeRanges[attr]
				imputedVal = math.Max(rng[0], math.Min(rng[1], imputedVal))
				setAttributeValue(&imputed[i], attr, math.Round(imputedVal*100)/100)
			} else {
				attr := attributeOrder[attrIdx]
				rng := attributeRanges[attr]
				fallback := (rng[0] + rng[1]) / 2
				setAttributeValue(&imputed[i], attr, math.Round(fallback*100)/100)
			}
		}
	}

	return imputed
}

func buildDecisionMatrix(materials []models.ProtectionMaterial) [][]float64 {
	matrix := make([][]float64, len(materials))
	for i, m := range materials {
		row := make([]float64, len(attributeOrder))
		for j, attr := range attributeOrder {
			row[j] = getAttributeValue(m, attr)
		}
		matrix[i] = row
	}
	return matrix
}

func normalizeMatrix(matrix [][]float64) [][]float64 {
	n := len(matrix)
	if n == 0 {
		return matrix
	}
	numCols := len(matrix[0])
	colSumsSq := make([]float64, numCols)

	for i := 0; i < n; i++ {
		for j := 0; j < numCols; j++ {
			colSumsSq[j] += matrix[i][j] * matrix[i][j]
		}
	}

	norm := make([][]float64, n)
	for i := 0; i < n; i++ {
		norm[i] = make([]float64, numCols)
		for j := 0; j < numCols; j++ {
			sqrtSum := math.Sqrt(colSumsSq[j])
			if sqrtSum != 0 {
				norm[i][j] = matrix[i][j] / sqrtSum
			}
		}
	}
	return norm
}

func applyWeights(norm [][]float64, weights []float64) [][]float64 {
	n := len(norm)
	if n == 0 {
		return norm
	}
	numCols := len(norm[0])
	weighted := make([][]float64, n)
	for i := 0; i < n; i++ {
		weighted[i] = make([]float64, numCols)
		for j := 0; j < numCols; j++ {
			weighted[i][j] = norm[i][j] * weights[j]
		}
	}
	return weighted
}

func determineIdealBest(weighted [][]float64) []float64 {
	n := len(weighted)
	if n == 0 {
		return nil
	}
	numCols := len(weighted[0])
	ideal := make([]float64, numCols)

	for j := 0; j < numCols; j++ {
		attr := attributeOrder[j]
		ideal[j] = weighted[0][j]
		for i := 1; i < n; i++ {
			if nonBenefitAttributes[attr] {
				if weighted[i][j] < ideal[j] {
					ideal[j] = weighted[i][j]
				}
			} else {
				if weighted[i][j] > ideal[j] {
					ideal[j] = weighted[i][j]
				}
			}
		}
	}
	return ideal
}

func determineIdealWorst(weighted [][]float64) []float64 {
	n := len(weighted)
	if n == 0 {
		return nil
	}
	numCols := len(weighted[0])
	ideal := make([]float64, numCols)

	for j := 0; j < numCols; j++ {
		attr := attributeOrder[j]
		ideal[j] = weighted[0][j]
		for i := 1; i < n; i++ {
			if nonBenefitAttributes[attr] {
				if weighted[i][j] > ideal[j] {
					ideal[j] = weighted[i][j]
				}
			} else {
				if weighted[i][j] < ideal[j] {
					ideal[j] = weighted[i][j]
				}
			}
		}
	}
	return ideal
}

func euclideanDistance(row []float64, ideal []float64) float64 {
	var sum float64
	for j := 0; j < len(row); j++ {
		diff := row[j] - ideal[j]
		sum += diff * diff
	}
	return math.Sqrt(sum)
}

type topsisRunResult struct {
	scores []float64
	ranks  []int
}

func runTOPSISOnce(materials []models.ProtectionMaterial, weightSlice []float64) topsisRunResult {
	decisionMatrix := buildDecisionMatrix(materials)
	normalized := normalizeMatrix(decisionMatrix)
	weighted := applyWeights(normalized, weightSlice)

	idealBest := determineIdealBest(weighted)
	idealWorst := determineIdealWorst(weighted)

	n := len(materials)
	scores := make([]float64, n)

	for i := 0; i < n; i++ {
		dBest := euclideanDistance(weighted[i], idealBest)
		dWorst := euclideanDistance(weighted[i], idealWorst)
		sum := dBest + dWorst
		if sum != 0 {
			scores[i] = dWorst / sum
		}
		scores[i] = math.Round(scores[i]*10000) / 10000
	}

	type indexed struct {
		idx   int
		score float64
	}
	ranked := make([]indexed, n)
	for i, s := range scores {
		ranked[i] = indexed{idx: i, score: s}
	}
	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].score > ranked[j].score
	})

	ranks := make([]int, n)
	for rank, r := range ranked {
		ranks[r.idx] = rank + 1
	}

	return topsisRunResult{scores: scores, ranks: ranks}
}

func (svc *TOPSISService) SensitivityAnalysis(
	materials []models.ProtectionMaterial,
	baseWeights map[string]float64,
	baseResult topsisRunResult,
) []models.SensitivityAnalysisResult {
	results := make([]models.SensitivityAnalysisResult, 0, len(attributeOrder)*2)

	perturbationAmounts := []float64{0.20, -0.20}

	for attrIdx, attr := range attributeOrder {
		baseWeight := baseWeights[attr]

		for _, perturb := range perturbationAmounts {
			perturbedWeights := make(map[string]float64)
			for k, v := range baseWeights {
				perturbedWeights[k] = v
			}

			delta := baseWeight * perturb
			perturbedWeights[attr] = baseWeight + delta

			remainingDelta := -delta
			otherCount := 0
			for k := range perturbedWeights {
				if k != attr {
					otherCount++
				}
			}
			if otherCount > 0 {
				perAttrDelta := remainingDelta / float64(otherCount)
				for k := range perturbedWeights {
					if k != attr {
						perturbedWeights[k] += perAttrDelta
						if perturbedWeights[k] < 0 {
							perturbedWeights[k] = 0.001
						}
					}
				}
			}

			total := 0.0
			for _, v := range perturbedWeights {
				total += v
			}
			for k := range perturbedWeights {
				perturbedWeights[k] /= total
			}

			weightSlice := make([]float64, len(attributeOrder))
			for i, a := range attributeOrder {
				weightSlice[i] = perturbedWeights[a]
			}

			perturbedResult := runTOPSISOnce(materials, weightSlice)

			scoreChangeSum := 0.0
			rankChangeSum := 0
			for i := range baseResult.scores {
				scoreChangeSum += math.Abs(perturbedResult.scores[i] - baseResult.scores[i])
				rankChangeSum += int(math.Abs(float64(perturbedResult.ranks[i] - baseResult.ranks[i])))
			}

			avgScoreChange := scoreChangeSum / float64(len(baseResult.scores))
			avgRankChange := float64(rankChangeSum) / float64(len(baseResult.scores))
			sensitivity := avgScoreChange*10 + avgRankChange*0.05

			results = append(results, models.SensitivityAnalysisResult{
				AttributeName:   attributeDisplayName[attr] + (map[float64]string{0.2: " (+)", -0.2: " (-)"}[perturb]),
				OriginalWeight:  math.Round(baseWeight*1000) / 1000,
				PerturbedWeight: math.Round(perturbedWeights[attr]*1000) / 1000,
				ScoreChange:     math.Round(avgScoreChange*10000) / 10000,
				RankChange:      rankChangeSum,
				Sensitivity:     math.Round(sensitivity*1000) / 1000,
			})
			_ = attrIdx
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Sensitivity > results[j].Sensitivity
	})

	return results
}

func (svc *TOPSISService) Optimize(materials []models.ProtectionMaterial, req models.TOPSISRequest) (*models.TOPSISResponse, error) {
	if len(materials) == 0 {
		return nil, errors.New("no materials provided")
	}

	imputedMaterials := svc.knnImpute(materials, 3)

	missingInfo := make([]map[string]interface{}, 0)
	for i, orig := range materials {
		_, origMissing := extractFeatures(orig)
		hasMissing := false
		missingAttrs := []string{}
		for j, m := range origMissing {
			if m {
				hasMissing = true
				missingAttrs = append(missingAttrs, attributeDisplayName[attributeOrder[j]])
			}
		}
		if hasMissing {
			imputedVals := make(map[string]float64)
			for j, m := range origMissing {
				if m {
					imputedVals[attributeDisplayName[attributeOrder[j]]] = getAttributeValue(imputedMaterials[i], attributeOrder[j])
				}
			}
			missingInfo = append(missingInfo, map[string]interface{}{
				"material_name": orig.Name,
				"missing_attrs": missingAttrs,
				"imputed_values": imputedVals,
			})
		}
	}

	effectiveWeights := make(map[string]float64)
	for k, v := range defaultWeights {
		effectiveWeights[k] = v
	}
	for k, v := range req.Priorities {
		effectiveWeights[k] = v
	}

	weightSum := 0.0
	for _, v := range effectiveWeights {
		weightSum += v
	}
	for k := range effectiveWeights {
		effectiveWeights[k] /= weightSum
	}

	weightSlice := make([]float64, len(attributeOrder))
	for i, attr := range attributeOrder {
		weightSlice[i] = effectiveWeights[attr]
	}

	decisionMatrix := buildDecisionMatrix(imputedMaterials)
	normalized := normalizeMatrix(decisionMatrix)
	weighted := applyWeights(normalized, weightSlice)

	idealBest := determineIdealBest(weighted)
	idealWorst := determineIdealWorst(weighted)

	n := len(imputedMaterials)
	scores := make([]float64, n)

	for i := 0; i < n; i++ {
		dBest := euclideanDistance(weighted[i], idealBest)
		dWorst := euclideanDistance(weighted[i], idealWorst)
		sum := dBest + dWorst
		if sum != 0 {
			scores[i] = dWorst / sum
		}
		scores[i] = math.Round(scores[i]*10000) / 10000
	}

	type indexed struct {
		idx   int
		score float64
	}
	ranked := make([]indexed, n)
	for i, s := range scores {
		ranked[i] = indexed{idx: i, score: s}
	}
	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].score > ranked[j].score
	})

	results := make([]models.TOPSISResult, n)
	for rank, r := range ranked {
		results[rank] = models.TOPSISResult{
			MaterialID:   imputedMaterials[r.idx].ID,
			MaterialName: imputedMaterials[r.idx].Name,
			TOPSISScore:  r.score,
			Rank:         rank + 1,
		}
	}

	baseRun := topsisRunResult{scores: scores}
	baseRun.ranks = make([]int, n)
	for rank, r := range ranked {
		baseRun.ranks[r.idx] = rank + 1
	}

	sensitivityAnalysis := svc.SensitivityAnalysis(imputedMaterials, effectiveWeights, baseRun)

	stabilityScore := 1.0
	if len(sensitivityAnalysis) > 0 {
		avgSensitivity := 0.0
		for _, sa := range sensitivityAnalysis {
			avgSensitivity += sa.Sensitivity
		}
		avgSensitivity /= float64(len(sensitivityAnalysis))
		stabilityScore = math.Max(0, 1.0-avgSensitivity*2.0)
		stabilityScore = math.Round(stabilityScore*1000) / 1000
	}

	return &models.TOPSISResponse{
		Results:             results,
		DecisionMatrix:      decisionMatrix,
		NormalizedMatrix:    normalized,
		WeightedMatrix:      weighted,
		MissingInfo:         missingInfo,
		SensitivityAnalysis: sensitivityAnalysis,
		StabilityScore:      stabilityScore,
	}, nil
}
