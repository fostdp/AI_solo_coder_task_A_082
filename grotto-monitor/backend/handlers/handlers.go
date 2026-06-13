package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	"grotto-monitor/backend/models"
	"grotto-monitor/backend/repository"
	"grotto-monitor/backend/services"
	"grotto-monitor/backend/websocket"
)

type Handler struct {
	Repo          *repository.Repository
	PredictionSvc *services.WeatheringPredictionService
	TopsisSvc     *services.TOPSISService
	Hub           *websocket.Hub
}

func NewHandler(repo *repository.Repository, predictionSvc *services.WeatheringPredictionService, topsisSvc *services.TOPSISService, hub *websocket.Hub) *Handler {
	return &Handler{
		Repo:          repo,
		PredictionSvc: predictionSvc,
		TopsisSvc:     topsisSvc,
		Hub:           hub,
	}
}

func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.Use(corsMiddleware)
	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/sites", h.GetSites).Methods("GET", "OPTIONS")
	api.HandleFunc("/sites/{id}", h.GetSite).Methods("GET", "OPTIONS")
	api.HandleFunc("/sites/{id}/sensors", h.GetSensorsBySite).Methods("GET", "OPTIONS")
	api.HandleFunc("/sites/{id}/data", h.GetMonitoringData).Methods("GET", "OPTIONS")
	api.HandleFunc("/sensors/{id}/hourly", h.GetHourlyData).Methods("GET", "OPTIONS")
	api.HandleFunc("/data", h.InsertMonitoringData).Methods("POST", "OPTIONS")
	api.HandleFunc("/alerts", h.GetAlerts).Methods("GET", "OPTIONS")
	api.HandleFunc("/alerts/{id}/acknowledge", h.AcknowledgeAlert).Methods("PUT", "OPTIONS")
	api.HandleFunc("/materials", h.GetMaterials).Methods("GET", "OPTIONS")
	api.HandleFunc("/predict", h.PredictWeatheringRate).Methods("POST", "OPTIONS")
	api.HandleFunc("/predict/batch", h.PredictBatchRates).Methods("POST", "OPTIONS")
	api.HandleFunc("/topsis", h.OptimizeMaterials).Methods("POST", "OPTIONS")
	api.HandleFunc("/sites/{id}/weathering-rates", h.GetWeatheringRates).Methods("GET", "OPTIONS")
	r.HandleFunc("/ws/alerts", h.ServeWs).Methods("GET")
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

func (h *Handler) GetSites(w http.ResponseWriter, r *http.Request) {
	sites, err := h.Repo.GetAllSites()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, sites)
}

func (h *Handler) GetSite(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid site id")
		return
	}
	site, err := h.Repo.GetSiteByID(id)
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

func (h *Handler) GetSensorsBySite(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid site id")
		return
	}
	sensors, err := h.Repo.GetSensorsBySiteID(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, sensors)
}

func (h *Handler) GetMonitoringData(w http.ResponseWriter, r *http.Request) {
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
	data, err := h.Repo.GetMonitoringDataBySite(id, hours)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	sensors, err := h.Repo.GetSensorsBySiteID(id)
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

func (h *Handler) GetHourlyData(w http.ResponseWriter, r *http.Request) {
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
	data, err := h.Repo.GetHourlyDataBySensor(id, days)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, data)
}

func (h *Handler) GetWeatheringRates(w http.ResponseWriter, r *http.Request) {
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
	rates, err := h.Repo.CalculateWeatheringRates(id, days)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, rates)
}

func (h *Handler) InsertMonitoringData(w http.ResponseWriter, r *http.Request) {
	var input models.MonitoringDataInsert
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	data, err := h.Repo.InsertMonitoringData(input)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	sensor, err := h.Repo.GetSensorByID(int64(input.SensorID))
	if err == nil && sensor != nil {
		alertTriggered := false
		var alertMsg string
		var alertType string
		var threshold float64

		if sensor.Type == "裂隙宽度" && input.Value > 0.5 {
			alertTriggered = true
			alertType = "裂隙超限"
			threshold = 0.5
			alertMsg = "裂隙宽度超过阈值: " + strconv.FormatFloat(input.Value, 'f', 2, 64) + "mm"
		}

		if sensor.Type == "表面硬度" {
			baseline, err := h.Repo.GetBaselineHardness(int64(input.SensorID))
			if err == nil && baseline > 0 {
				drop := (baseline - input.Value) / baseline
				if drop > 0.2 {
					alertTriggered = true
					alertType = "硬度下降"
					threshold = baseline * 0.8
					alertMsg = "表面硬度下降超过20%: 当前值 " + strconv.FormatFloat(input.Value, 'f', 2, 64) + ", 基准值 " + strconv.FormatFloat(baseline, 'f', 2, 64)
				}
			}
		}

		if alertTriggered {
			alert := models.Alert{
				SiteID:       sensor.SiteID,
				SensorID:     sensor.ID,
				AlertType:    alertType,
				Message:      alertMsg,
				Severity:     "warning",
				Value:        input.Value,
				Threshold:    threshold,
				TriggeredAt:  time.Now(),
				Acknowledged: false,
			}
			insertedAlert, err := h.Repo.InsertAlert(alert)
			if err == nil {
				msg, _ := json.Marshal(map[string]interface{}{
					"type":  "alert",
					"alert": insertedAlert,
				})
				h.Hub.Broadcast <- msg
			}
		}
	}

	respondJSON(w, http.StatusCreated, data)
}

func (h *Handler) GetAlerts(w http.ResponseWriter, r *http.Request) {
	filter := models.AlertFilter{}
	if siteIDStr := r.URL.Query().Get("site_id"); siteIDStr != "" {
		if siteID, err := strconv.ParseInt(siteIDStr, 10, 64); err == nil {
			siteIDInt := int(siteID)
			filter.SiteID = &siteIDInt
		}
	}
	if ackStr := r.URL.Query().Get("acknowledged"); ackStr != "" {
		ack, err := strconv.ParseBool(ackStr)
		if err == nil {
			filter.Acknowledged = &ack
		}
	}
	alerts, err := h.Repo.GetAlerts(filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, alerts)
}

func (h *Handler) AcknowledgeAlert(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid alert id")
		return
	}
	if err := h.Repo.AcknowledgeAlert(id); err != nil {
		respondError(w, http.StatusNotFound, "alert not found")
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "acknowledged"})
}

func (h *Handler) GetMaterials(w http.ResponseWriter, r *http.Request) {
	materials, err := h.Repo.GetAllMaterials()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, materials)
}

func (h *Handler) PredictWeatheringRate(w http.ResponseWriter, r *http.Request) {
	var input models.PredictionRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	site, err := h.Repo.GetSiteByID(int64(input.SiteID))
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if site == nil {
		respondError(w, http.StatusNotFound, "site not found")
		return
	}
	historicalData, err := h.Repo.GetMonitoringDataBySite(int64(input.SiteID), 365*24)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get historical data")
		return
	}
	result, err := h.PredictionSvc.PredictEnsemble(site, historicalData, input)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, result)
}

func (h *Handler) PredictBatchRates(w http.ResponseWriter, r *http.Request) {
	var input models.BatchPredictionRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	site, err := h.Repo.GetSiteByID(int64(input.SiteID))
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if site == nil {
		respondError(w, http.StatusNotFound, "site not found")
		return
	}
	historicalData, err := h.Repo.GetMonitoringDataBySite(int64(input.SiteID), 365*24)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get historical data")
		return
	}
	result, err := h.PredictionSvc.PredictBatch(site.RockType, input.Combinations, len(historicalData))
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, result)
}

func (h *Handler) OptimizeMaterials(w http.ResponseWriter, r *http.Request) {
	var input models.TOPSISRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	materials, err := h.Repo.GetAllMaterials()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	result, err := h.TopsisSvc.Optimize(materials, input)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, result)
}

func (h *Handler) ServeWs(w http.ResponseWriter, r *http.Request) {
	websocket.ServeWs(h.Hub, w, r)
}
