package repository

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"

	"grotto-monitor/backend/models"
)

type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

func (c *Config) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode)
}

func Connect(ctx context.Context, cfg Config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}
	return db, nil
}

type Repository struct {
	DB *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{DB: db}
}

func (r *Repository) GetAllSites() ([]models.GrottoSite, error) {
	ctx := context.Background()
	rows, err := r.DB.QueryContext(ctx, "SELECT id, name, location, rock_type, created_at, description, latitude, longitude FROM grotto_sites ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("querying sites: %w", err)
	}
	defer rows.Close()

	var sites []models.GrottoSite
	for rows.Next() {
		var s models.GrottoSite
		if err := rows.Scan(&s.ID, &s.Name, &s.Location, &s.RockType, &s.CreatedAt, &s.Description, &s.Latitude, &s.Longitude); err != nil {
			return nil, fmt.Errorf("scanning site: %w", err)
		}
		sites = append(sites, s)
	}
	return sites, rows.Err()
}

func (r *Repository) GetSiteByID(id int64) (*models.GrottoSite, error) {
	ctx := context.Background()
	var s models.GrottoSite
	err := r.DB.QueryRowContext(ctx,
		"SELECT id, name, location, rock_type, created_at, description, latitude, longitude FROM grotto_sites WHERE id = $1", id).
		Scan(&s.ID, &s.Name, &s.Location, &s.RockType, &s.CreatedAt, &s.Description, &s.Latitude, &s.Longitude)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying site by id: %w", err)
	}
	return &s, nil
}

func (r *Repository) GetSensorsBySiteID(id int64) ([]models.Sensor, error) {
	ctx := context.Background()
	rows, err := r.DB.QueryContext(ctx,
		"SELECT id, site_id, sensor_code, sensor_type, sensor_type, position_x, position_y, position_z, installed_at, status, baseline_value FROM sensors WHERE site_id = $1 ORDER BY id", id)
	if err != nil {
		return nil, fmt.Errorf("querying sensors: %w", err)
	}
	defer rows.Close()

	var sensors []models.Sensor
	for rows.Next() {
		var s models.Sensor
		if err := rows.Scan(&s.ID, &s.SiteID, &s.SensorCode, &s.SensorType, &s.Type, &s.PositionX, &s.PositionY, &s.PositionZ, &s.InstalledAt, &s.Status, &s.BaselineValue); err != nil {
			return nil, fmt.Errorf("scanning sensor: %w", err)
		}
		sensors = append(sensors, s)
	}
	return sensors, rows.Err()
}

func (r *Repository) GetSensorByID(id int64) (*models.Sensor, error) {
	ctx := context.Background()
	var s models.Sensor
	err := r.DB.QueryRowContext(ctx,
		"SELECT id, site_id, sensor_code, sensor_type, sensor_type, position_x, position_y, position_z, installed_at, status, baseline_value FROM sensors WHERE id = $1", id).
		Scan(&s.ID, &s.SiteID, &s.SensorCode, &s.SensorType, &s.Type, &s.PositionX, &s.PositionY, &s.PositionZ, &s.InstalledAt, &s.Status, &s.BaselineValue)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying sensor by id: %w", err)
	}
	return &s, nil
}

func (r *Repository) GetMonitoringDataBySite(id int64, hours int) ([]models.MonitoringData, error) {
	ctx := context.Background()
	rows, err := r.DB.QueryContext(ctx,
		"SELECT time, sensor_id, site_id, value FROM monitoring_data WHERE site_id = $1 AND time > NOW() - INTERVAL '1 hour' * $2 ORDER BY time DESC", id, hours)
	if err != nil {
		return nil, fmt.Errorf("querying latest data: %w", err)
	}
	defer rows.Close()

	var data []models.MonitoringData
	for rows.Next() {
		var d models.MonitoringData
		if err := rows.Scan(&d.Time, &d.SensorID, &d.SiteID, &d.Value); err != nil {
			return nil, fmt.Errorf("scanning monitoring data: %w", err)
		}
		data = append(data, d)
	}
	return data, rows.Err()
}

func (r *Repository) GetHourlyDataBySensor(id int64, days int) ([]models.MonitoringHourly, error) {
	ctx := context.Background()
	rows, err := r.DB.QueryContext(ctx,
		"SELECT bucket, sensor_id, site_id, avg_value, min_value, max_value FROM monitoring_hourly WHERE sensor_id = $1 AND bucket > NOW() - INTERVAL '1 day' * $2 ORDER BY bucket", id, days)
	if err != nil {
		return nil, fmt.Errorf("querying hourly data: %w", err)
	}
	defer rows.Close()

	var hourly []models.MonitoringHourly
	for rows.Next() {
		var h models.MonitoringHourly
		if err := rows.Scan(&h.Bucket, &h.SensorID, &h.SiteID, &h.AvgValue, &h.MinValue, &h.MaxValue); err != nil {
			return nil, fmt.Errorf("scanning hourly data: %w", err)
		}
		hourly = append(hourly, h)
	}
	return hourly, rows.Err()
}

func (r *Repository) InsertMonitoringData(input models.MonitoringDataInsert) (*models.MonitoringData, error) {
	ctx := context.Background()
	_, err := r.DB.ExecContext(ctx,
		"INSERT INTO monitoring_data (time, sensor_id, site_id, value) VALUES ($1, $2, $3, $4)",
		input.Time, input.SensorID, input.SiteID, input.Value)
	if err != nil {
		return nil, fmt.Errorf("inserting monitoring data: %w", err)
	}
	return &models.MonitoringData{
		Time:     input.Time,
		SensorID: input.SensorID,
		SiteID:   input.SiteID,
		Value:    input.Value,
	}, nil
}

func (r *Repository) GetBaselineHardness(sensorID int64) (float64, error) {
	ctx := context.Background()
	var baseline float64
	err := r.DB.QueryRowContext(ctx,
		"SELECT baseline_value FROM sensors WHERE id = $1", sensorID).Scan(&baseline)
	if err != nil {
		return 0, err
	}
	return baseline, nil
}

func (r *Repository) InsertAlert(alert models.Alert) (*models.Alert, error) {
	ctx := context.Background()
	var id int
	err := r.DB.QueryRowContext(ctx,
		"INSERT INTO alerts (site_id, sensor_id, alert_type, severity, message, value, threshold, triggered_at, acknowledged) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id",
		alert.SiteID, alert.SensorID, alert.AlertType, alert.Severity, alert.Message, alert.Value, alert.Threshold, alert.TriggeredAt, alert.Acknowledged).Scan(&id)
	if err != nil {
		return nil, err
	}
	alert.ID = id
	return &alert, nil
}

func (r *Repository) GetAlerts(filter models.AlertFilter) ([]models.Alert, error) {
	ctx := context.Background()
	query := "SELECT id, site_id, sensor_id, alert_type, severity, message, value, threshold, triggered_at, acknowledged FROM alerts WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if filter.SiteID != nil {
		query += fmt.Sprintf(" AND site_id = $%d", argIdx)
		args = append(args, *filter.SiteID)
		argIdx++
	}
	if filter.Acknowledged != nil {
		query += fmt.Sprintf(" AND acknowledged = $%d", argIdx)
		args = append(args, *filter.Acknowledged)
		argIdx++
	}
	query += " ORDER BY triggered_at DESC"

	rows, err := r.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying alerts: %w", err)
	}
	defer rows.Close()

	var alerts []models.Alert
	for rows.Next() {
		var a models.Alert
		if err := rows.Scan(&a.ID, &a.SiteID, &a.SensorID, &a.AlertType, &a.Severity, &a.Message, &a.Value, &a.Threshold, &a.TriggeredAt, &a.Acknowledged); err != nil {
			return nil, fmt.Errorf("scanning alert: %w", err)
		}
		alerts = append(alerts, a)
	}
	return alerts, rows.Err()
}

func (r *Repository) AcknowledgeAlert(id int64) error {
	ctx := context.Background()
	_, err := r.DB.ExecContext(ctx, "UPDATE alerts SET acknowledged = true WHERE id = $1", id)
	return err
}

func (r *Repository) GetAllMaterials() ([]models.ProtectionMaterial, error) {
	ctx := context.Background()
	rows, err := r.DB.QueryContext(ctx,
		"SELECT id, name, category, penetration_depth, breathability, weathering_resistance, compatibility, cost, durability_years, description FROM protection_materials ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("querying materials: %w", err)
	}
	defer rows.Close()

	var materials []models.ProtectionMaterial
	for rows.Next() {
		var m models.ProtectionMaterial
		if err := rows.Scan(&m.ID, &m.Name, &m.Category, &m.PenetrationDepth, &m.Breathability, &m.WeatheringResistance, &m.Compatibility, &m.Cost, &m.DurabilityYears, &m.Description); err != nil {
			return nil, fmt.Errorf("scanning material: %w", err)
		}
		materials = append(materials, m)
	}
	return materials, rows.Err()
}

func (r *Repository) GetDataForPrediction(siteID int64, days int) ([]models.MonitoringData, error) {
	ctx := context.Background()
	rows, err := r.DB.QueryContext(ctx,
		"SELECT time, sensor_id, site_id, value FROM monitoring_data WHERE site_id = $1 AND time > NOW() - INTERVAL '1 day' * $2 ORDER BY time", siteID, days)
	if err != nil {
		return nil, fmt.Errorf("querying data for prediction: %w", err)
	}
	defer rows.Close()

	var data []models.MonitoringData
	for rows.Next() {
		var d models.MonitoringData
		if err := rows.Scan(&d.Time, &d.SensorID, &d.SiteID, &d.Value); err != nil {
			return nil, fmt.Errorf("scanning prediction data: %w", err)
		}
		data = append(data, d)
	}
	return data, rows.Err()
}

func (r *Repository) CalculateWeatheringRates(siteID int64, days int) ([]models.WeatheringRatePoint, error) {
	ctx := context.Background()
	rows, err := r.DB.QueryContext(ctx, `
		SELECT 
			time_bucket('1 day', md.time) as day,
			md_s.value as hardness,
			md_t.value as temperature,
			md_h.value as humidity
		FROM (
			SELECT time_bucket('1 day', time) as day, sensor_id, avg(value) as value
			FROM monitoring_data 
			WHERE site_id = $1 AND time > NOW() - INTERVAL '1 day' * $2
			GROUP BY day, sensor_id
		) md
		LEFT JOIN (
			SELECT time_bucket('1 day', time) as day, avg(value) as value
			FROM monitoring_data 
			WHERE site_id = $1 AND sensor_id IN (SELECT id FROM sensors WHERE sensor_type = '表面硬度')
			AND time > NOW() - INTERVAL '1 day' * $2
			GROUP BY day
		) md_s ON md.day = md_s.day
		LEFT JOIN (
			SELECT time_bucket('1 day', time) as day, avg(value) as value
			FROM monitoring_data 
			WHERE site_id = $1 AND sensor_id IN (SELECT id FROM sensors WHERE sensor_type = '温度')
			AND time > NOW() - INTERVAL '1 day' * $2
			GROUP BY day
		) md_t ON md.day = md_t.day
		LEFT JOIN (
			SELECT time_bucket('1 day', time) as day, avg(value) as value
			FROM monitoring_data 
			WHERE site_id = $1 AND sensor_id IN (SELECT id FROM sensors WHERE sensor_type = '湿度')
			AND time > NOW() - INTERVAL '1 day' * $2
			GROUP BY day
		) md_h ON md.day = md_h.day
		WHERE md_s.value IS NOT NULL
		GROUP BY md.day, md_s.value, md_t.value, md_h.value
		ORDER BY md.day
	`, siteID, days)
	if err != nil {
		return nil, fmt.Errorf("calculating weathering rates: %w", err)
	}
	defer rows.Close()

	var rates []models.WeatheringRatePoint
	var prevHardness float64
	first := true
	for rows.Next() {
		var t time.Time
		var hardness, temp, hum float64
		if err := rows.Scan(&t, &hardness, &temp, &hum); err != nil {
			return nil, fmt.Errorf("scanning rate point: %w", err)
		}
		if !first && prevHardness > 0 {
			rate := (prevHardness - hardness) / prevHardness
			rates = append(rates, models.WeatheringRatePoint{
				Time:        t,
				Rate:        rate,
				Temperature: temp,
				Humidity:    hum,
			})
		}
		prevHardness = hardness
		first = false
	}
	return rates, rows.Err()
}
