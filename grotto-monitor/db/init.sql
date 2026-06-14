CREATE EXTENSION IF NOT EXISTS timescaledb;

CREATE TABLE grotto_sites (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    location VARCHAR(200),
    rock_type VARCHAR(50),
    created_at TIMESTAMP,
    description TEXT,
    latitude DOUBLE PRECISION,
    longitude DOUBLE PRECISION
);

INSERT INTO grotto_sites (name, location, rock_type, created_at, description, latitude, longitude) VALUES
('敦煌莫高窟', '甘肃省敦煌市', '砂岩', '2024-01-01 00:00:00', '世界文化遗产，以精美壁画和塑像闻名于世，始建于前秦时期', 40.0422, 94.8094),
('云冈石窟', '山西省大同市', '砂岩', '2024-01-01 00:00:00', '世界文化遗产，北魏时期开凿的大型石窟群，造像气势宏伟', 40.1054, 113.1384),
('龙门石窟', '河南省洛阳市', '石灰岩', '2024-01-01 00:00:00', '世界文化遗产，北魏至唐代开凿，石刻艺术精湛', 34.5589, 112.4714),
('麦积山石窟', '甘肃省天水市', '砂岩', '2024-01-01 00:00:00', '世界文化遗产，以泥塑艺术著称，被誉为东方雕塑馆', 34.3833, 105.9833),
('大足石刻', '重庆市大足区', '砂岩', '2024-01-01 00:00:00', '世界文化遗产，唐宋时期石刻艺术代表作，题材丰富', 29.7175, 105.7175),
('响堂山石窟', '河北省邯郸市', '石灰岩', '2024-01-01 00:00:00', '北齐时期皇家石窟，造像风格独特，刻经丰富', 36.4500, 114.1000),
('巩义石窟', '河南省巩义市', '砂岩', '2024-01-01 00:00:00', '北魏至宋代石窟，帝后礼佛图雕刻精美', 34.7500, 112.9500),
('炳灵寺石窟', '甘肃省永靖县', '砂岩', '2024-01-01 00:00:00', '西秦至明代石窟群，保存有最早纪年题记', 35.7500, 103.0500),
('克孜尔石窟', '新疆维吾尔自治区拜城县', '砂岩', '2024-01-01 00:00:00', '中国开凿最早的大型石窟群，龟兹艺术代表', 41.7500, 82.5000),
('须弥山石窟', '宁夏回族自治区固原市', '砂岩', '2024-01-01 00:00:00', '北魏至唐代石窟群，丝绸之路重要遗存', 36.1667, 106.2500);

CREATE TABLE sensors (
    id SERIAL PRIMARY KEY,
    site_id INT REFERENCES grotto_sites(id),
    sensor_code VARCHAR(50) UNIQUE NOT NULL,
    sensor_type VARCHAR(30),
    position_x FLOAT,
    position_y FLOAT,
    position_z FLOAT,
    installed_at TIMESTAMP,
    status VARCHAR(20) DEFAULT 'active',
    baseline_value FLOAT DEFAULT 0
);

INSERT INTO sensors (site_id, sensor_code, sensor_type, position_x, position_y, position_z, installed_at, status, baseline_value) VALUES
(1, 'DH-MGK-T-01', '温度', 12.5, 3.2, 8.1, '2024-01-15 08:00:00', 'active', 0),
(1, 'DH-MGK-H-01', '湿度', 12.5, 3.2, 8.1, '2024-01-15 08:00:00', 'active', 0),
(1, 'DH-MGK-S-01', '表面硬度', 15.0, 5.8, 6.3, '2024-01-15 08:00:00', 'active', 55.0),
(1, 'DH-MGK-C-01', '裂隙宽度', 18.2, 4.5, 7.7, '2024-01-15 08:00:00', 'active', 0),
(2, 'YG-SK-T-01', '温度', 20.0, 6.0, 10.5, '2024-01-16 09:00:00', 'active', 0),
(2, 'YG-SK-H-01', '湿度', 20.0, 6.0, 10.5, '2024-01-16 09:00:00', 'active', 0),
(2, 'YG-SK-S-01', '表面硬度', 22.3, 8.4, 9.0, '2024-01-16 09:00:00', 'active', 58.0),
(2, 'YG-SK-C-01', '裂隙宽度', 25.1, 7.2, 11.3, '2024-01-16 09:00:00', 'active', 0),
(3, 'LM-SK-T-01', '温度', 30.0, 9.0, 15.0, '2024-01-17 08:30:00', 'active', 0),
(3, 'LM-SK-H-01', '湿度', 30.0, 9.0, 15.0, '2024-01-17 08:30:00', 'active', 0),
(3, 'LM-SK-S-01', '表面硬度', 33.5, 11.2, 13.8, '2024-01-17 08:30:00', 'active', 62.0),
(3, 'LM-SK-C-01', '裂隙宽度', 35.8, 10.5, 16.2, '2024-01-17 08:30:00', 'active', 0),
(4, 'MJS-SK-T-01', '温度', 8.0, 2.5, 5.0, '2024-01-18 10:00:00', 'active', 0),
(4, 'MJS-SK-H-01', '湿度', 8.0, 2.5, 5.0, '2024-01-18 10:00:00', 'active', 0),
(4, 'MJS-SK-S-01', '表面硬度', 10.5, 4.3, 3.8, '2024-01-18 10:00:00', 'active', 52.0),
(4, 'MJS-SK-C-01', '裂隙宽度', 12.0, 3.6, 6.1, '2024-01-18 10:00:00', 'active', 0),
(5, 'DZ-SK-T-01', '温度', 28.0, 7.5, 12.0, '2024-01-19 09:30:00', 'active', 0),
(5, 'DZ-SK-H-01', '湿度', 28.0, 7.5, 12.0, '2024-01-19 09:30:00', 'active', 0),
(5, 'DZ-SK-S-01', '表面硬度', 30.5, 9.8, 10.5, '2024-01-19 09:30:00', 'active', 56.0),
(5, 'DZ-SK-C-01', '裂隙宽度', 32.3, 8.6, 13.7, '2024-01-19 09:30:00', 'active', 0),
(6, 'XTS-SK-T-01', '温度', 16.0, 4.8, 9.0, '2024-01-20 08:00:00', 'active', 0),
(6, 'XTS-SK-H-01', '湿度', 16.0, 4.8, 9.0, '2024-01-20 08:00:00', 'active', 0),
(6, 'XTS-SK-S-01', '表面硬度', 18.5, 6.9, 7.5, '2024-01-20 08:00:00', 'active', 60.0),
(6, 'XTS-SK-C-01', '裂隙宽度', 20.2, 5.7, 10.0, '2024-01-20 08:00:00', 'active', 0),
(7, 'GY-SK-T-01', '温度', 24.0, 5.5, 11.0, '2024-01-21 09:00:00', 'active', 0),
(7, 'GY-SK-H-01', '湿度', 24.0, 5.5, 11.0, '2024-01-21 09:00:00', 'active', 0),
(7, 'GY-SK-S-01', '表面硬度', 26.8, 7.6, 9.2, '2024-01-21 09:00:00', 'active', 54.0),
(7, 'GY-SK-C-01', '裂隙宽度', 28.5, 6.3, 12.4, '2024-01-21 09:00:00', 'active', 0),
(8, 'BLS-SK-T-01', '温度', 6.0, 2.0, 4.5, '2024-01-22 10:00:00', 'active', 0),
(8, 'BLS-SK-H-01', '湿度', 6.0, 2.0, 4.5, '2024-01-22 10:00:00', 'active', 0),
(8, 'BLS-SK-S-01', '表面硬度', 8.5, 3.8, 3.2, '2024-01-22 10:00:00', 'active', 53.0),
(8, 'BLS-SK-C-01', '裂隙宽度', 10.0, 3.1, 5.8, '2024-01-22 10:00:00', 'active', 0),
(9, 'KZE-SK-T-01', '温度', 14.0, 3.5, 7.5, '2024-01-23 08:00:00', 'active', 0),
(9, 'KZE-SK-H-01', '湿度', 14.0, 3.5, 7.5, '2024-01-23 08:00:00', 'active', 0),
(9, 'KZE-SK-S-01', '表面硬度', 16.5, 5.6, 6.0, '2024-01-23 08:00:00', 'active', 57.0),
(9, 'KZE-SK-C-01', '裂隙宽度', 18.8, 4.9, 8.3, '2024-01-23 08:00:00', 'active', 0),
(10, 'XMS-SK-T-01', '温度', 22.0, 4.2, 8.0, '2024-01-24 09:00:00', 'active', 0),
(10, 'XMS-SK-H-01', '湿度', 22.0, 4.2, 8.0, '2024-01-24 09:00:00', 'active', 0),
(10, 'XMS-SK-S-01', '表面硬度', 24.5, 6.3, 6.8, '2024-01-24 09:00:00', 'active', 59.0),
(10, 'XMS-SK-C-01', '裂隙宽度', 26.2, 5.1, 9.5, '2024-01-24 09:00:00', 'active', 0);

CREATE TABLE monitoring_data (
    time TIMESTAMPTZ NOT NULL,
    sensor_id INT NOT NULL REFERENCES sensors(id),
    site_id INT NOT NULL,
    value DOUBLE PRECISION NOT NULL
);

SELECT create_hypertable('monitoring_data', 'time', chunk_time_interval => INTERVAL '1 day');

CREATE TABLE protection_materials (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100),
    category VARCHAR(50),
    penetration_depth DOUBLE PRECISION,
    breathability DOUBLE PRECISION,
    weathering_resistance DOUBLE PRECISION,
    compatibility DOUBLE PRECISION,
    cost DOUBLE PRECISION,
    durability_years DOUBLE PRECISION,
    description TEXT
);

INSERT INTO protection_materials (name, category, penetration_depth, breathability, weathering_resistance, compatibility, cost, durability_years, description) VALUES
('有机硅防水剂', '防水', 2.0, 0.85, 0.70, 0.90, 120, 15, '有机硅类防水材料，渗透性好，对砂岩和石灰岩均有良好兼容性，能有效防止水分侵入'),
('纳米石灰加固剂', '加固', 3.5, 0.90, 0.75, 0.95, 280, 20, '纳米级氢氧化钙分散液，与石灰岩基底高度兼容，可深层渗透加固风化层'),
('环氧树脂', '加固', 1.5, 0.30, 0.85, 0.50, 200, 25, '高强度加固材料，粘结力强但透气性差，适用于结构性加固修复'),
('硅丙烯酸树脂', '防护', 2.5, 0.75, 0.80, 0.85, 150, 12, '硅改性丙烯酸树脂，兼具有机硅的疏水性和丙烯酸的耐候性'),
('氟碳涂料', '防护', 0.5, 0.60, 0.95, 0.70, 500, 30, '氟碳树脂涂层，耐候性极佳，适用于表面防护，成本较高'),
('纳米TiO2光催化涂层', '防护', 1.0, 0.70, 0.90, 0.80, 350, 10, '纳米二氧化钛光催化材料，可降解表面污染物，自清洁效果好'),
('改性硅酸乙酯', '加固', 4.0, 0.95, 0.65, 0.85, 180, 8, '硅酸乙酯改性加固剂，透气性优异，能深度渗透但耐候性一般'),
('聚氨酯防护剂', '防护', 2.0, 0.50, 0.80, 0.60, 160, 18, '聚氨酯类防护涂层，柔韧性好，耐磨损，但透气性中等');

CREATE TABLE alerts (
    id SERIAL PRIMARY KEY,
    site_id INT REFERENCES grotto_sites(id),
    sensor_id INT REFERENCES sensors(id),
    alert_type VARCHAR(50),
    severity VARCHAR(20),
    message TEXT,
    value DOUBLE PRECISION,
    threshold DOUBLE PRECISION,
    triggered_at TIMESTAMPTZ NOT NULL,
    acknowledged BOOLEAN DEFAULT FALSE
);

CREATE TABLE alert_thresholds (
    id SERIAL PRIMARY KEY,
    sensor_type VARCHAR(50) UNIQUE,
    threshold_value DOUBLE PRECISION,
    threshold_percent DOUBLE PRECISION,
    description TEXT
);

INSERT INTO alert_thresholds (sensor_type, threshold_value, threshold_percent, description) VALUES
('裂隙宽度', 0.5, NULL, '裂隙宽度超过0.5mm触发告警'),
('表面硬度', NULL, 20, '表面硬度下降超过20%触发告警');

CREATE INDEX idx_monitoring_data_time ON monitoring_data (time DESC);
CREATE INDEX idx_monitoring_data_sensor_id ON monitoring_data (sensor_id);
CREATE INDEX idx_monitoring_data_site_id ON monitoring_data (site_id);

CREATE MATERIALIZED VIEW monitoring_hourly
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('1 hour', time) AS bucket,
    sensor_id,
    site_id,
    avg(value) AS avg_value,
    min(value) AS min_value,
    max(value) AS max_value
FROM monitoring_data
GROUP BY bucket, sensor_id, site_id
WITH NO DATA;

SELECT add_continuous_aggregate_policy('monitoring_hourly',
    start_offset => INTERVAL '3 hours',
    end_offset => INTERVAL '1 hour',
    schedule_interval => INTERVAL '1 hour');

ALTER TABLE monitoring_data SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'site_id, sensor_id',
    timescaledb.compress_orderby = 'time DESC'
);

SELECT add_compression_policy('monitoring_data',
    compress_after => INTERVAL '7 days',
    schedule_interval => INTERVAL '1 day',
    if_not_exists => TRUE
);

ALTER TABLE monitoring_data SET (
    timescaledb.compress_chunk_time_interval = '1 day'
);

CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

ALTER SYSTEM SET shared_preload_libraries = 'timescaledb,pg_stat_statements';
ALTER SYSTEM SET work_mem = '64MB';
ALTER SYSTEM SET maintenance_work_mem = '512MB';
ALTER SYSTEM SET effective_cache_size = '4GB';
ALTER SYSTEM SET max_wal_size = '4GB';
ALTER SYSTEM SET checkpoint_completion_target = 0.9;

CREATE INDEX IF NOT EXISTS idx_monitoring_data_time_sensor ON monitoring_data (time DESC, sensor_id);
CREATE INDEX IF NOT EXISTS idx_monitoring_data_time_site ON monitoring_data (time DESC, site_id);

CREATE OR REPLACE VIEW monitoring_summary AS
SELECT
    gs.id AS site_id,
    gs.name AS site_name,
    COUNT(DISTINCT s.id) AS sensor_count,
    COUNT(md.*) AS total_records,
    MIN(md.time) AS earliest_record,
    MAX(md.time) AS latest_record
FROM grotto_sites gs
LEFT JOIN sensors s ON s.site_id = gs.id
LEFT JOIN monitoring_data md ON md.site_id = gs.id
GROUP BY gs.id, gs.name
ORDER BY gs.id;

CREATE OR REPLACE FUNCTION get_site_latest_data(p_site_id INT)
RETURNS TABLE (
    sensor_id INT,
    sensor_code VARCHAR,
    sensor_type VARCHAR,
    latest_time TIMESTAMPTZ,
    latest_value DOUBLE PRECISION
) AS $$
BEGIN
    RETURN QUERY
    SELECT DISTINCT ON (s.id)
        s.id,
        s.sensor_code,
        s.sensor_type,
        md.time,
        md.value
    FROM sensors s
    LEFT JOIN monitoring_data md ON md.sensor_id = s.id
    WHERE s.site_id = p_site_id
    ORDER BY s.id, md.time DESC;
END;
$$ LANGUAGE plpgsql;
