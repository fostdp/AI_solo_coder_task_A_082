package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"
)

type SensorConfig struct {
	ID            int
	SiteID        int
	SensorCode    string
	SensorType    string
	BaselineValue float64
	MinValue      float64
	MaxValue      float64
	CurrentValue  float64
}

type MonitoringData struct {
	Time     time.Time `json:"time"`
	SensorID int       `json:"sensor_id"`
	SiteID   int       `json:"site_id"`
	Value    float64   `json:"value"`
}

var sensors = []SensorConfig{
	{ID: 1, SiteID: 1, SensorCode: "DH-MGK-T-01", SensorType: "温度", BaselineValue: 0, MinValue: -10, MaxValue: 45, CurrentValue: 15},
	{ID: 2, SiteID: 1, SensorCode: "DH-MGK-H-01", SensorType: "湿度", BaselineValue: 0, MinValue: 10, MaxValue: 80, CurrentValue: 30},
	{ID: 3, SiteID: 1, SensorCode: "DH-MGK-S-01", SensorType: "表面硬度", BaselineValue: 55.0, MinValue: 20, MaxValue: 60, CurrentValue: 55},
	{ID: 4, SiteID: 1, SensorCode: "DH-MGK-C-01", SensorType: "裂隙宽度", BaselineValue: 0, MinValue: 0, MaxValue: 2.0, CurrentValue: 0.1},
	{ID: 5, SiteID: 2, SensorCode: "YG-SK-T-01", SensorType: "温度", BaselineValue: 0, MinValue: -15, MaxValue: 40, CurrentValue: 12},
	{ID: 6, SiteID: 2, SensorCode: "YG-SK-H-01", SensorType: "湿度", BaselineValue: 0, MinValue: 15, MaxValue: 85, CurrentValue: 40},
	{ID: 7, SiteID: 2, SensorCode: "YG-SK-S-01", SensorType: "表面硬度", BaselineValue: 58.0, MinValue: 25, MaxValue: 62, CurrentValue: 58},
	{ID: 8, SiteID: 2, SensorCode: "YG-SK-C-01", SensorType: "裂隙宽度", BaselineValue: 0, MinValue: 0, MaxValue: 2.5, CurrentValue: 0.15},
	{ID: 9, SiteID: 3, SensorCode: "LM-SK-T-01", SensorType: "温度", BaselineValue: 0, MinValue: -5, MaxValue: 42, CurrentValue: 18},
	{ID: 10, SiteID: 3, SensorCode: "LM-SK-H-01", SensorType: "湿度", BaselineValue: 0, MinValue: 20, MaxValue: 90, CurrentValue: 55},
	{ID: 11, SiteID: 3, SensorCode: "LM-SK-S-01", SensorType: "表面硬度", BaselineValue: 62.0, MinValue: 30, MaxValue: 65, CurrentValue: 62},
	{ID: 12, SiteID: 3, SensorCode: "LM-SK-C-01", SensorType: "裂隙宽度", BaselineValue: 0, MinValue: 0, MaxValue: 1.8, CurrentValue: 0.08},
	{ID: 13, SiteID: 4, SensorCode: "MJS-SK-T-01", SensorType: "温度", BaselineValue: 0, MinValue: -8, MaxValue: 38, CurrentValue: 14},
	{ID: 14, SiteID: 4, SensorCode: "MJS-SK-H-01", SensorType: "湿度", BaselineValue: 0, MinValue: 25, MaxValue: 88, CurrentValue: 60},
	{ID: 15, SiteID: 4, SensorCode: "MJS-SK-S-01", SensorType: "表面硬度", BaselineValue: 52.0, MinValue: 22, MaxValue: 55, CurrentValue: 52},
	{ID: 16, SiteID: 4, SensorCode: "MJS-SK-C-01", SensorType: "裂隙宽度", BaselineValue: 0, MinValue: 0, MaxValue: 2.2, CurrentValue: 0.12},
	{ID: 17, SiteID: 5, SensorCode: "DZ-SK-T-01", SensorType: "温度", BaselineValue: 0, MinValue: 0, MaxValue: 45, CurrentValue: 22},
	{ID: 18, SiteID: 5, SensorCode: "DZ-SK-H-01", SensorType: "湿度", BaselineValue: 0, MinValue: 30, MaxValue: 95, CurrentValue: 70},
	{ID: 19, SiteID: 5, SensorCode: "DZ-SK-S-01", SensorType: "表面硬度", BaselineValue: 56.0, MinValue: 28, MaxValue: 60, CurrentValue: 56},
	{ID: 20, SiteID: 5, SensorCode: "DZ-SK-C-01", SensorType: "裂隙宽度", BaselineValue: 0, MinValue: 0, MaxValue: 1.9, CurrentValue: 0.1},
	{ID: 21, SiteID: 6, SensorCode: "XTS-SK-T-01", SensorType: "温度", BaselineValue: 0, MinValue: -12, MaxValue: 40, CurrentValue: 13},
	{ID: 22, SiteID: 6, SensorCode: "XTS-SK-H-01", SensorType: "湿度", BaselineValue: 0, MinValue: 18, MaxValue: 82, CurrentValue: 45},
	{ID: 23, SiteID: 6, SensorCode: "XTS-SK-S-01", SensorType: "表面硬度", BaselineValue: 60.0, MinValue: 30, MaxValue: 63, CurrentValue: 60},
	{ID: 24, SiteID: 6, SensorCode: "XTS-SK-C-01", SensorType: "裂隙宽度", BaselineValue: 0, MinValue: 0, MaxValue: 2.1, CurrentValue: 0.09},
	{ID: 25, SiteID: 7, SensorCode: "GY-SK-T-01", SensorType: "温度", BaselineValue: 0, MinValue: -6, MaxValue: 41, CurrentValue: 16},
	{ID: 26, SiteID: 7, SensorCode: "GY-SK-H-01", SensorType: "湿度", BaselineValue: 0, MinValue: 22, MaxValue: 86, CurrentValue: 52},
	{ID: 27, SiteID: 7, SensorCode: "GY-SK-S-01", SensorType: "表面硬度", BaselineValue: 54.0, MinValue: 24, MaxValue: 57, CurrentValue: 54},
	{ID: 28, SiteID: 7, SensorCode: "GY-SK-C-01", SensorType: "裂隙宽度", BaselineValue: 0, MinValue: 0, MaxValue: 2.0, CurrentValue: 0.11},
	{ID: 29, SiteID: 8, SensorCode: "BLS-SK-T-01", SensorType: "温度", BaselineValue: 0, MinValue: -10, MaxValue: 39, CurrentValue: 12},
	{ID: 30, SiteID: 8, SensorCode: "BLS-SK-H-01", SensorType: "湿度", BaselineValue: 0, MinValue: 20, MaxValue: 80, CurrentValue: 42},
	{ID: 31, SiteID: 8, SensorCode: "BLS-SK-S-01", SensorType: "表面硬度", BaselineValue: 53.0, MinValue: 23, MaxValue: 56, CurrentValue: 53},
	{ID: 32, SiteID: 8, SensorCode: "BLS-SK-C-01", SensorType: "裂隙宽度", BaselineValue: 0, MinValue: 0, MaxValue: 1.7, CurrentValue: 0.08},
	{ID: 33, SiteID: 9, SensorCode: "KZE-SK-T-01", SensorType: "温度", BaselineValue: 0, MinValue: -20, MaxValue: 40, CurrentValue: 10},
	{ID: 34, SiteID: 9, SensorCode: "KZE-SK-H-01", SensorType: "湿度", BaselineValue: 0, MinValue: 10, MaxValue: 70, CurrentValue: 25},
	{ID: 35, SiteID: 9, SensorCode: "KZE-SK-S-01", SensorType: "表面硬度", BaselineValue: 57.0, MinValue: 26, MaxValue: 60, CurrentValue: 57},
	{ID: 36, SiteID: 9, SensorCode: "KZE-SK-C-01", SensorType: "裂隙宽度", BaselineValue: 0, MinValue: 0, MaxValue: 1.5, CurrentValue: 0.07},
	{ID: 37, SiteID: 10, SensorCode: "XMS-SK-T-01", SensorType: "温度", BaselineValue: 0, MinValue: -15, MaxValue: 38, CurrentValue: 11},
	{ID: 38, SiteID: 10, SensorCode: "XMS-SK-H-01", SensorType: "湿度", BaselineValue: 0, MinValue: 15, MaxValue: 75, CurrentValue: 38},
	{ID: 39, SiteID: 10, SensorCode: "XMS-SK-S-01", SensorType: "表面硬度", BaselineValue: 59.0, MinValue: 28, MaxValue: 62, CurrentValue: 59},
	{ID: 40, SiteID: 10, SensorCode: "XMS-SK-C-01", SensorType: "裂隙宽度", BaselineValue: 0, MinValue: 0, MaxValue: 1.8, CurrentValue: 0.09},
}

var hourlyWeatherData = []struct {
	Hour      int
	TempDelta float64
	HumDelta  float64
}{
	{0, -2.0, 3.0},
	{1, -2.5, 3.5},
	{2, -3.0, 4.0},
	{3, -3.2, 4.2},
	{4, -3.0, 4.0},
	{5, -2.0, 3.0},
	{6, -1.0, 1.5},
	{7, 0.5, -0.5},
	{8, 2.0, -2.0},
	{9, 4.0, -4.0},
	{10, 5.5, -5.0},
	{11, 6.5, -5.5},
	{12, 7.0, -6.0},
	{13, 7.2, -6.2},
	{14, 6.8, -5.8},
	{15, 6.0, -5.0},
	{16, 4.5, -3.5},
	{17, 2.5, -1.5},
	{18, 0.5, 0.5},
	{19, -1.0, 2.0},
	{20, -1.8, 2.8},
	{21, -2.2, 3.2},
	{22, -2.4, 3.4},
	{23, -2.5, 3.5},
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return intValue
}

func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return boolValue
}

func generateSensorValue(sensor *SensorConfig, hour int) float64 {
	seasonalFactor := 1.0 + 0.3*math.Sin(2*math.Pi*float64(time.Now().YearDay())/365.25)

	switch sensor.SensorType {
	case "温度":
		weather := hourlyWeatherData[hour]
		baseTemp := (sensor.MinValue + sensor.MaxValue) / 2
		newValue := baseTemp + weather.TempDelta*seasonalFactor
		newValue += (rand.Float64() - 0.5) * 1.5
		newValue = math.Max(sensor.MinValue, math.Min(sensor.MaxValue, newValue))
		return math.Round(newValue*10) / 10

	case "湿度":
		weather := hourlyWeatherData[hour]
		baseHum := (sensor.MinValue + sensor.MaxValue) / 2
		newValue := baseHum + weather.HumDelta*seasonalFactor
		newValue += (rand.Float64() - 0.5) * 3
		newValue = math.Max(sensor.MinValue, math.Min(sensor.MaxValue, newValue))
		return math.Round(newValue*10) / 10

	case "表面硬度":
		weatheringRate := 0.0001 + rand.Float64()*0.0002
		newValue := sensor.CurrentValue - weatheringRate
		newValue += (rand.Float64() - 0.5) * 0.2
		newValue = math.Max(sensor.MinValue, math.Min(sensor.MaxValue, newValue))
		return math.Round(newValue*100) / 100

	case "裂隙宽度":
		expansionRate := 0.00005 + rand.Float64()*0.0001
		temp := getTemperatureForSite(sensor.SiteID, hour)
		tempExpansion := (temp - 15) * 0.0001
		newValue := sensor.CurrentValue + expansionRate + tempExpansion
		newValue += (rand.Float64() - 0.5) * 0.01
		newValue = math.Max(sensor.MinValue, math.Min(sensor.MaxValue, newValue))
		return math.Round(newValue*1000) / 1000

	default:
		return sensor.CurrentValue
	}
}

func getTemperatureForSite(siteID, hour int) float64 {
	for _, s := range sensors {
		if s.SiteID == siteID && s.SensorType == "温度" {
			weather := hourlyWeatherData[hour]
			baseTemp := (s.MinValue + s.MaxValue) / 2
			return baseTemp + weather.TempDelta
		}
	}
	return 20
}

func sendData(apiURL string, data MonitoringData) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal data: %w", err)
	}

	resp, err := http.Post(apiURL+"/api/data", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("post data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func backfillHistoricalData(apiURL string, days int) {
	log.Printf("开始回灌 %d 天历史数据...", days)
	now := time.Now()
	totalHours := days * 24
	processed := 0

	for i := totalHours; i > 0; i-- {
		dataTime := now.Add(time.Duration(-i) * time.Hour)
		hour := dataTime.Hour()

		for j := range sensors {
			sensor := &sensors[j]
			value := generateSensorValue(sensor, hour)
			sensor.CurrentValue = value

			data := MonitoringData{
				Time:     dataTime,
				SensorID: sensor.ID,
				SiteID:   sensor.SiteID,
				Value:    value,
			}

			if err := sendData(apiURL, data); err != nil {
				log.Printf("回灌数据失败 [%s]: %v", sensor.SensorCode, err)
			}
		}

		processed++
		if processed%100 == 0 {
			log.Printf("已回灌 %d/%d 小时数据", processed, totalHours)
		}
	}

	log.Println("历史数据回灌完成")
}

func main() {
	rand.Seed(time.Now().UnixNano())

	apiHost := getEnv("API_HOST", "localhost")
	apiPort := getEnvInt("API_PORT", 8080)
	apiURL := fmt.Sprintf("http://%s:%d", apiHost, apiPort)
	intervalHours := getEnvInt("INTERVAL_HOURS", 1)
	backfillDays := getEnvInt("BACKFILL_DAYS", 30)
	shouldBackfill := getEnvBool("BACKFILL", true)

	log.Println("========================================")
	log.Println("石窟监测传感器模拟器启动")
	log.Println("========================================")
	log.Printf("API 地址: %s", apiURL)
	log.Printf("上报间隔: %d 小时", intervalHours)
	log.Printf("回灌天数: %d 天", backfillDays)
	log.Printf("传感器数量: %d", len(sensors))
	log.Println("========================================")

	if shouldBackfill {
		backfillHistoricalData(apiURL, backfillDays)
	}

	log.Println("开始实时数据上报...")

	ticker := time.NewTicker(time.Duration(intervalHours) * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		hour := now.Hour()

		log.Printf("[%s] 开始上报数据...", now.Format("2006-01-02 15:04:05"))

		for j := range sensors {
			sensor := &sensors[j]
			value := generateSensorValue(sensor, hour)
			sensor.CurrentValue = value

			data := MonitoringData{
				Time:     now,
				SensorID: sensor.ID,
				SiteID:   sensor.SiteID,
				Value:    value,
			}

			if err := sendData(apiURL, data); err != nil {
				log.Printf("上报失败 [%s]: %v", sensor.SensorCode, err)
			} else {
				log.Printf("  ✓ %-20s = %8.3f", sensor.SensorCode, value)
			}
		}

		log.Printf("上报完成，等待 %d 小时后下次上报...", intervalHours)
		log.Println("----------------------------------------")
	}
}
