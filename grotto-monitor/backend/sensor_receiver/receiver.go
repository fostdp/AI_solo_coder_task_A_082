package sensor_receiver

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"grotto-monitor/backend/config"
	"grotto-monitor/backend/models"
	"grotto-monitor/backend/repository"
)

type AlertEvent struct {
	SiteID      int
	SensorID    int
	AlertType   string
	Severity    string
	Message     string
	Value       float64
	Threshold   float64
	TriggeredAt time.Time
}

type SensorReceiver struct {
	repo        *repository.Repository
	alertCh     chan<- AlertEvent
	alarmConfig config.AlarmParams
}

func NewSensorReceiver(repo *repository.Repository, alertCh chan<- AlertEvent, cfg config.AlarmParams) *SensorReceiver {
	return &SensorReceiver{
		repo:        repo,
		alertCh:     alertCh,
		alarmConfig: cfg,
	}
}

func (sr *SensorReceiver) RegisterRoutes(r *mux.Router) {
	r.Use(corsMiddleware)
	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/sites", sr.GetSites).Methods("GET", "OPTIONS")
	api.HandleFunc("/sites/{id}", sr.GetSite).Methods("GET", "OPTIONS")
	api.HandleFunc("/sites/{id}/sensors", sr.GetSensorsBySite).Methods("GET", "OPTIONS")
	api.HandleFunc("/sites/{id}/data", sr.GetMonitoringData).Methods("GET", "OPTIONS")
	api.HandleFunc("/sensors/{id}/hourly", sr.GetHourlyData).Methods("GET", "OPTIONS")
	api.HandleFunc("/data", sr.InsertMonitoringData).Methods("POST", "OPTIONS")
	api.HandleFunc("/sites/{id}/weathering-rates", sr.GetWeatheringRates).Methods("GET", "OPTIONS")
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

func (sr *SensorReceiver) GetSites(w http.ResponseWriter, r *http.Request) {
	sites, err := sr.repo.GetAllSites()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, sites)
}

func (sr *SensorReceiver) GetSite(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid site id")
		return
	}
	site, err := sr.repo.GetSiteByID(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if site == nil {
		respondError(w, http.StatusNotFound, "site not found")
		return
	}
	respondJSON(w, http.StatusOK, site)
}

func (sr *SensorReceiver) GetSensorsBySite(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid site id")
		return
	}
	sensors, err := sr.repo.GetSensorsBySiteID(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, sensors)
}

func (sr *SensorReceiver) GetMonitoringData(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid site id")
		return
	}
	hoursStr := r.URL.Query().Get("hours")
	hours := 24
	if hoursStr != "" {
		if h, err := strconv.Atoi(hoursStr); err == nil {
			hours = h
		}
	}
	data, err := sr.repo.GetMonitoringDataBySite(id, hours)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	sensors, err := sr.repo.GetSensorsBySiteID(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	grouped := make(map[int][]models.MonitoringData)
	for _, d := range data {
		grouped[d.SensorID] = append(grouped[d.SensorID], d)
	}
	result := make([]map[string]interface{}, 0)
	for _, sensor := range sensors {
		entry := map[string]interface{}{
			"sensor": sensor,
			"data":   grouped[sensor.ID],
		}
		result = append(result, entry)
	}
	respondJSON(w, http.StatusOK, result)
}

func (sr *SensorReceiver) GetHourlyData(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid sensor id")
		return
	}
	daysStr := r.URL.Query().Get("days")
	days := 30
	if daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil {
			days = d
		}
	}
	data, err := sr.repo.GetHourlyDataBySensor(id, days)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, data)
}

func (sr *SensorReceiver) GetWeatheringRates(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid site id")
		return
	}
	daysStr := r.URL.Query().Get("days")
	days := 365
	if daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil {
			days = d
		}
	}
	rates, err := sr.repo.CalculateWeatheringRates(id, days)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, rates)
}

func (sr *SensorReceiver) InsertMonitoringData(w http.ResponseWriter, r *http.Request) {
	var input models.MonitoringDataInsert
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	data, err := sr.repo.InsertMonitoringData(input)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	sensor, err := sr.repo.GetSensorByID(int64(input.SensorID))
	if err == nil && sensor != nil {
		if sensor.Type == "裂隙宽度" && input.Value > sr.alarmConfig.CrackWidthThreshold {
			sr.sendAlert(AlertEvent{
				SiteID:      sensor.SiteID,
				SensorID:    sensor.ID,
				AlertType:   "裂隙超限",
				Severity:    "warning",
				Message:     "裂隙宽度超过阈值: " + strconv.FormatFloat(input.Value, 'f', 2, 64) + "mm",
				Value:       input.Value,
				Threshold:   sr.alarmConfig.CrackWidthThreshold,
				TriggeredAt: time.Now(),
			})
		}

		if sensor.Type == "表面硬度" {
			baseline, err := sr.repo.GetBaselineHardness(int64(input.SensorID))
			if err == nil && baseline > 0 {
				drop := (baseline - input.Value) / baseline
				dropPercent := sr.alarmConfig.HardnessDropPercent / 100.0
				if drop > dropPercent {
					threshold := baseline * (1.0 - dropPercent)
					sr.sendAlert(AlertEvent{
						SiteID:      sensor.SiteID,
						SensorID:    sensor.ID,
						AlertType:   "硬度下降",
						Severity:    "warning",
						Message:     "表面硬度下降超过" + strconv.FormatFloat(sr.alarmConfig.HardnessDropPercent, 'f', 0, 64) + "%: 当前值 " + strconv.FormatFloat(input.Value, 'f', 2, 64) + ", 基准值 " + strconv.FormatFloat(baseline, 'f', 2, 64),
						Value:       input.Value,
						Threshold:   threshold,
						TriggeredAt: time.Now(),
					})
				}
			}
		}
	}

	respondJSON(w, http.StatusCreated, data)
}

func (sr *SensorReceiver) sendAlert(event AlertEvent) {
	select {
	case sr.alertCh <- event:
	default:
		log.Println("alert channel full, dropping alert event")
	}
}
