package services

import (
	"context"
	"time"

	"grotto-monitor/internal/database"
	"grotto-monitor/internal/models"

	"github.com/jackc/pgx/v5"
)

type CaveService struct{}

func NewCaveService() *CaveService {
	return &CaveService{}
}

func (s *CaveService) GetAll(ctx context.Context) ([]models.Cave, error) {
	rows, err := database.DB.Query(ctx, `
		SELECT id, name, location, province, latitude, longitude, rock_type, dynasty, description, created_at, updated_at
		FROM caves ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var caves []models.Cave
	for rows.Next() {
		var c models.Cave
		err := rows.Scan(&c.ID, &c.Name, &c.Location, &c.Province, &c.Latitude, &c.Longitude,
			&c.RockType, &c.Dynasty, &c.Description, &c.CreatedAt, &c.UpdatedAt)
		if err != nil {
			return nil, err
		}
		caves = append(caves, c)
	}
	return caves, nil
}

func (s *CaveService) GetByID(ctx context.Context, id int) (*models.Cave, error) {
	var c models.Cave
	err := database.DB.QueryRow(ctx, `
		SELECT id, name, location, province, latitude, longitude, rock_type, dynasty, description, created_at, updated_at
		FROM caves WHERE id = $1
	`, id).Scan(&c.ID, &c.Name, &c.Location, &c.Province, &c.Latitude, &c.Longitude,
		&c.RockType, &c.Dynasty, &c.Description, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *CaveService) GetStatistics(ctx context.Context) ([]models.CaveStatistics, error) {
	rows, err := database.DB.Query(ctx, `
		SELECT cave_id, cave_name, province, rock_type, total_points, active_alerts
		FROM v_cave_statistics ORDER BY cave_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []models.CaveStatistics
	for rows.Next() {
		var s1 models.CaveStatistics
		err := rows.Scan(&s1.CaveID, &s1.CaveName, &s1.Province, &s1.RockType, &s1.TotalPoints, &s1.ActiveAlerts)
		if err != nil {
			return nil, err
		}
		stats = append(stats, s1)
	}
	return stats, nil
}

type MonitoringPointService struct{}

func NewMonitoringPointService() *MonitoringPointService {
	return &MonitoringPointService{}
}

func (s *MonitoringPointService) GetByCaveID(ctx context.Context, caveID int) ([]models.MonitoringPoint, error) {
	rows, err := database.DB.Query(ctx, `
		SELECT id, cave_id, name, code, position_x, position_y, position_z, rock_type,
		       initial_hardness, initial_crack_width, area_description, status, created_at, updated_at
		FROM monitoring_points WHERE cave_id = $1 ORDER BY id
	`, caveID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []models.MonitoringPoint
	for rows.Next() {
		var p models.MonitoringPoint
		err := rows.Scan(&p.ID, &p.CaveID, &p.Name, &p.Code, &p.PositionX, &p.PositionY, &p.PositionZ,
			&p.RockType, &p.InitialHardness, &p.InitialCrackWidth, &p.AreaDescription, &p.Status, &p.CreatedAt, &p.UpdatedAt)
		if err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, nil
}

func (s *MonitoringPointService) GetByID(ctx context.Context, id int) (*models.MonitoringPoint, error) {
	var p models.MonitoringPoint
	err := database.DB.QueryRow(ctx, `
		SELECT id, cave_id, name, code, position_x, position_y, position_z, rock_type,
		       initial_hardness, initial_crack_width, area_description, status, created_at, updated_at
		FROM monitoring_points WHERE id = $1
	`, id).Scan(&p.ID, &p.CaveID, &p.Name, &p.Code, &p.PositionX, &p.PositionY, &p.PositionZ,
		&p.RockType, &p.InitialHardness, &p.InitialCrackWidth, &p.AreaDescription, &p.Status, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

type SensorDataService struct{}

func NewSensorDataService() *SensorDataService {
	return &SensorDataService{}
}

func (s *SensorDataService) Insert(ctx context.Context, data *models.SensorData) error {
	_, err := database.DB.Exec(ctx, `
		INSERT INTO sensor_data (time, point_id, temperature, humidity, surface_hardness, crack_width,
		                          wind_speed, rainfall, solar_radiation, co2_concentration, vibration)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, data.Time, data.PointID, data.Temperature, data.Humidity, data.SurfaceHardness, data.CrackWidth,
		data.WindSpeed, data.Rainfall, data.SolarRadiation, data.CO2Concentration, data.Vibration)
	return err
}

func (s *SensorDataService) GetLatestByPoint(ctx context.Context, pointID int) (*models.LatestSensorData, error) {
	var d models.LatestSensorData
	err := database.DB.QueryRow(ctx, `
		SELECT point_id, time, temperature, humidity, surface_hardness, crack_width
		FROM sensor_data WHERE point_id = $1 ORDER BY time DESC LIMIT 1
	`, pointID).Scan(&d.PointID, &d.Time, &d.Temperature, &d.Humidity, &d.SurfaceHardness, &d.CrackWidth)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (s *SensorDataService) GetHistoryByPoint(ctx context.Context, pointID int, start, end time.Time) ([]models.SensorData, error) {
	rows, err := database.DB.Query(ctx, `
		SELECT time, point_id, temperature, humidity, surface_hardness, crack_width,
		       wind_speed, rainfall, solar_radiation, co2_concentration, vibration
		FROM sensor_data
		WHERE point_id = $1 AND time >= $2 AND time <= $3
		ORDER BY time ASC
	`, pointID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var data []models.SensorData
	for rows.Next() {
		var d models.SensorData
		err := rows.Scan(&d.Time, &d.PointID, &d.Temperature, &d.Humidity, &d.SurfaceHardness, &d.CrackWidth,
			&d.WindSpeed, &d.Rainfall, &d.SolarRadiation, &d.CO2Concentration, &d.Vibration)
		if err != nil {
			return nil, err
		}
		data = append(data, d)
	}
	return data, nil
}

func (s *SensorDataService) GetAllLatest(ctx context.Context) ([]models.LatestSensorData, error) {
	rows, err := database.DB.Query(ctx, `
		SELECT point_id, time, temperature, humidity, surface_hardness, crack_width
		FROM v_latest_sensor_data
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var data []models.LatestSensorData
	for rows.Next() {
		var d models.LatestSensorData
		err := rows.Scan(&d.PointID, &d.Time, &d.Temperature, &d.Humidity, &d.SurfaceHardness, &d.CrackWidth)
		if err != nil {
			return nil, err
		}
		data = append(data, d)
	}
	return data, nil
}

type AlertService struct{}

func NewAlertService() *AlertService {
	return &AlertService{}
}

func (s *AlertService) Insert(ctx context.Context, alert *models.Alert) error {
	_, err := database.DB.Exec(ctx, `
		INSERT INTO alerts (point_id, alert_type, severity, title, message, current_value, threshold_value, sensor_data_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, alert.PointID, alert.AlertType, alert.Severity, alert.Title, alert.Message,
		alert.CurrentValue, alert.ThresholdValue, alert.SensorDataTime)
	return err
}

func (s *AlertService) GetRecent(ctx context.Context, limit int, acknowledged *bool) ([]models.Alert, error) {
	var rows pgx.Rows
	var err error

	query := `
		SELECT a.id, a.point_id, a.alert_type, a.severity, a.title, a.message,
		       a.current_value, a.threshold_value, a.sensor_data_time,
		       a.is_acknowledged, a.acknowledged_at, a.acknowledged_by, a.created_at
		FROM alerts a
	`

	if acknowledged != nil {
		query += ` WHERE a.is_acknowledged = $1 ORDER BY a.created_at DESC LIMIT $2`
		rows, err = database.DB.Query(ctx, query, *acknowledged, limit)
	} else {
		query += ` ORDER BY a.created_at DESC LIMIT $1`
		rows, err = database.DB.Query(ctx, query, limit)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []models.Alert
	for rows.Next() {
		var a models.Alert
		var pointName string
		err := rows.Scan(&a.ID, &a.PointID, &a.AlertType, &a.Severity, &a.Title, &a.Message,
			&a.CurrentValue, &a.ThresholdValue, &a.SensorDataTime,
			&a.IsAcknowledged, &a.AcknowledgedAt, &a.AcknowledgedBy, &a.CreatedAt, &pointName)
		if err != nil {
			return nil, err
		}
		alerts = append(alerts, a)
	}
	return alerts, nil
}

func (s *AlertService) Acknowledge(ctx context.Context, id int, user string) error {
	_, err := database.DB.Exec(ctx, `
		UPDATE alerts SET is_acknowledged = true, acknowledged_at = CURRENT_TIMESTAMP, acknowledged_by = $1
		WHERE id = $2
	`, user, id)
	return err
}

func (s *AlertService) CheckForExcessiveAlert(ctx context.Context, pointID int, alertType string, withinHours int) (bool, error) {
	var count int
	err := database.DB.QueryRow(ctx, `
		SELECT COUNT(*) FROM alerts
		WHERE point_id = $1 AND alert_type = $2 AND created_at > NOW() - ($3 || ' hours')::INTERVAL
	`, pointID, alertType, withinHours).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
