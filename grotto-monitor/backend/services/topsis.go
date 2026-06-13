package services

import (
	"errors"
	"math"
	"sort"

	"grotto-monitor/backend/models"
)

type TOPSISService struct{}

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

var nonBenefitAttributes = map[string]bool{
	"cost": true,
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

func (s *TOPSISService) Optimize(materials []models.ProtectionMaterial, req models.TOPSISRequest) (*models.TOPSISResponse, error) {
	if len(materials) == 0 {
		return nil, errors.New("no materials provided")
	}

	effectiveWeights := make(map[string]float64)
	for k, v := range defaultWeights {
		effectiveWeights[k] = v
	}
	for k, v := range req.Priorities {
		effectiveWeights[k] = v
	}

	weightSlice := make([]float64, len(attributeOrder))
	for i, attr := range attributeOrder {
		weightSlice[i] = effectiveWeights[attr]
	}

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

	results := make([]models.TOPSISResult, n)
	for rank, r := range ranked {
		results[rank] = models.TOPSISResult{
			MaterialID:   materials[r.idx].ID,
			MaterialName: materials[r.idx].Name,
			TOPSISScore:  r.score,
			Rank:         rank + 1,
		}
	}

	return &models.TOPSISResponse{
		Results:          results,
		DecisionMatrix:   decisionMatrix,
		NormalizedMatrix: normalized,
		WeightedMatrix:   weighted,
	}, nil
}
