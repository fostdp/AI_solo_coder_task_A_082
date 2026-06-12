-- ============================================================
-- 古代石窟风化速率监测系统 - TimescaleDB初始化脚本
-- ============================================================

-- 创建数据库（如果不存在则需要手动创建）
-- CREATE DATABASE grotto_monitor;
-- \c grotto_monitor;

-- 启用TimescaleDB扩展
CREATE EXTENSION IF NOT EXISTS timescaledb;
CREATE EXTENSION IF NOT EXISTS postgis;

-- ============================================================
-- 1. 石窟群表
-- ============================================================
CREATE TABLE IF NOT EXISTS caves (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    location VARCHAR(200) NOT NULL,
    province VARCHAR(50) NOT NULL,
    latitude DECIMAL(10, 7) NOT NULL,
    longitude DECIMAL(10, 7) NOT NULL,
    rock_type VARCHAR(50) NOT NULL,
    dynasty VARCHAR(100),
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- 2. 监测点表
-- ============================================================
CREATE TABLE IF NOT EXISTS monitoring_points (
    id SERIAL PRIMARY KEY,
    cave_id INTEGER NOT NULL REFERENCES caves(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    code VARCHAR(50) NOT NULL UNIQUE,
    position_x DECIMAL(10, 4) NOT NULL DEFAULT 0,
    position_y DECIMAL(10, 4) NOT NULL DEFAULT 0,
    position_z DECIMAL(10, 4) NOT NULL DEFAULT 0,
    rock_type VARCHAR(50) NOT NULL,
    initial_hardness DECIMAL(8, 2) NOT NULL,
    initial_crack_width DECIMAL(8, 4) DEFAULT 0,
    area_description TEXT,
    status SMALLINT DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_monitoring_points_cave_id ON monitoring_points(cave_id);

-- ============================================================
-- 3. 传感器数据表（时序Hypertable）
-- ============================================================
CREATE TABLE IF NOT EXISTS sensor_data (
    time TIMESTAMPTZ NOT NULL,
    point_id INTEGER NOT NULL REFERENCES monitoring_points(id) ON DELETE CASCADE,
    temperature DECIMAL(6, 2),
    humidity DECIMAL(6, 2),
    surface_hardness DECIMAL(8, 2),
    crack_width DECIMAL(8, 4),
    wind_speed DECIMAL(6, 2),
    rainfall DECIMAL(8, 2),
    solar_radiation DECIMAL(8, 2),
    co2_concentration DECIMAL(8, 2),
    vibration DECIMAL(8, 4)
);

-- 转换为TimescaleDB Hypertable
SELECT create_hypertable('sensor_data', 'time', if_not_exists => TRUE);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_sensor_data_point_id_time ON sensor_data(point_id, time DESC);

-- 压缩策略（可选，保存空间）
-- SELECT add_compression_policy('sensor_data', INTERVAL '7 days');

-- 保留策略（保留3年数据）
-- SELECT add_retention_policy('sensor_data', INTERVAL '3 years');

-- ============================================================
-- 4. 风化速率记录表
-- ============================================================
CREATE TABLE IF NOT EXISTS weathering_rates (
    id SERIAL PRIMARY KEY,
    point_id INTEGER NOT NULL REFERENCES monitoring_points(id) ON DELETE CASCADE,
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,
    hardness_loss_rate DECIMAL(10, 6) NOT NULL,
    crack_growth_rate DECIMAL(10, 6) NOT NULL,
    avg_temperature DECIMAL(6, 2),
    avg_humidity DECIMAL(6, 2),
    temperature_fluctuation DECIMAL(6, 2),
    rainfall_total DECIMAL(10, 2),
    overall_rate DECIMAL(10, 6) NOT NULL,
    risk_level VARCHAR(20) DEFAULT 'LOW',
    calculation_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_weathering_rates_point_id ON weathering_rates(point_id);
CREATE INDEX IF NOT EXISTS idx_weathering_rates_period ON weathering_rates(period_start, period_end);

-- ============================================================
-- 5. 保护材料数据库
-- ============================================================
CREATE TABLE IF NOT EXISTS protection_materials (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    category VARCHAR(50) NOT NULL,
    manufacturer VARCHAR(100),
    -- 材料属性评分 (1-10分，越高越好)
    weather_resistance DECIMAL(4, 2) NOT NULL,      -- 耐候性
    permeability DECIMAL(4, 2) NOT NULL,            -- 透气性
    adhesion DECIMAL(4, 2) NOT NULL,                -- 附着力
    reversibility DECIMAL(4, 2) NOT NULL,           -- 可再处理性
    durability DECIMAL(4, 2) NOT NULL,              -- 耐久性
    environmental_friendliness DECIMAL(4, 2) NOT NULL, -- 环保性
    cost_per_unit DECIMAL(10, 2) NOT NULL,          -- 单位成本
    coverage_rate DECIMAL(8, 2) NOT NULL,           -- 覆盖率(m²/kg)
    applicable_rock_types TEXT NOT NULL,            -- 适用岩石类型(JSON数组)
    construction_difficulty SMALLINT DEFAULT 3,     -- 施工难度(1-5)
    lifespan_years DECIMAL(6, 2),                   -- 预期寿命(年)
    description TEXT,
    status SMALLINT DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_materials_category ON protection_materials(category);

-- ============================================================
-- 6. 告警记录表
-- ============================================================
CREATE TABLE IF NOT EXISTS alerts (
    id SERIAL PRIMARY KEY,
    point_id INTEGER NOT NULL REFERENCES monitoring_points(id) ON DELETE CASCADE,
    alert_type VARCHAR(50) NOT NULL,        -- CRACK_EXCESSIVE, HARDNESS_DROP, etc.
    severity VARCHAR(20) NOT NULL,          -- INFO, WARNING, CRITICAL
    title VARCHAR(200) NOT NULL,
    message TEXT,
    current_value DECIMAL(12, 4),
    threshold_value DECIMAL(12, 4),
    sensor_data_time TIMESTAMPTZ NOT NULL,
    is_acknowledged BOOLEAN DEFAULT FALSE,
    acknowledged_at TIMESTAMP,
    acknowledged_by VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_alerts_point_id ON alerts(point_id);
CREATE INDEX IF NOT EXISTS idx_alerts_created ON alerts(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_alerts_severity ON alerts(severity);
CREATE INDEX IF NOT EXISTS idx_alerts_ack ON alerts(is_acknowledged);

-- ============================================================
-- 7. 风化预测结果表
-- ============================================================
CREATE TABLE IF NOT EXISTS weathering_predictions (
    id SERIAL PRIMARY KEY,
    point_id INTEGER NOT NULL REFERENCES monitoring_points(id) ON DELETE CASCADE,
    model_version VARCHAR(50) NOT NULL,
    input_temperature DECIMAL(6, 2) NOT NULL,
    input_humidity DECIMAL(6, 2) NOT NULL,
    input_temperature_range DECIMAL(6, 2),
    predicted_rate DECIMAL(12, 8) NOT NULL,
    prediction_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    confidence_level DECIMAL(5, 2) NOT NULL
);

-- ============================================================
-- 8. 材料推荐结果表
-- ============================================================
CREATE TABLE IF NOT EXISTS material_recommendations (
    id SERIAL PRIMARY KEY,
    cave_id INTEGER NOT NULL REFERENCES caves(id) ON DELETE CASCADE,
    point_id INTEGER REFERENCES monitoring_points(id) ON DELETE SET NULL,
    request_params JSONB NOT NULL,
    results JSONB NOT NULL,
    recommendation_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- 种子数据：10处著名石窟
-- ============================================================
INSERT INTO caves (name, location, province, latitude, longitude, rock_type, dynasty, description) VALUES
('敦煌莫高窟', '甘肃省敦煌市东南25公里', '甘肃', 40.043333, 94.806389, '砂砾岩', '十六国至元', '世界文化遗产，现存洞窟735个，壁画4.5万平方米'),
('云冈石窟', '山西省大同市城西16公里', '山西', 40.103889, 113.143056, '砂岩', '北魏', '世界文化遗产，现存主要洞窟45个，造像5.1万余尊'),
('龙门石窟', '河南省洛阳市南郊伊河两岸', '河南', 34.536111, 112.471389, '石灰岩', '北魏至宋', '世界文化遗产，现存窟龛2345个，造像10万余尊'),
('麦积山石窟', '甘肃省天水市麦积区', '甘肃', 34.352222, 106.847222, '砂砾岩', '后秦至明清', '世界文化遗产，以泥塑艺术著称'),
('大足石刻', '重庆市大足区', '重庆', 29.691389, 105.701667, '砂岩', '唐宋', '世界文化遗产，摩崖造像5万余尊'),
('克孜尔千佛洞', '新疆拜城县克孜尔镇', '新疆', 41.755556, 82.518333, '砂砾岩', '东汉至唐', '中国开凿最早的大型石窟群'),
('炳灵寺石窟', '甘肃省永靖县小积石山', '甘肃', 35.830000, 103.268889, '砂岩', '西秦至明', '世界文化遗产，现存窟龛183个'),
('响堂山石窟', '河北省邯郸市峰峰矿区', '河北', 36.473333, 114.182500, '石灰岩', '北齐至元', '北齐皇家石窟，分南北两窟群'),
('天梯山石窟', '甘肃省武威市凉州区', '甘肃', 37.719722, 102.836667, '砂砾岩', '北凉至明清', '誉为"中国石窟之祖"'),
('榆林窟', '甘肃省瓜州县', '甘肃', 40.388889, 95.864722, '砂砾岩', '北魏至元', '莫高窟的姊妹窟，现存洞窟41个')
ON CONFLICT (name) DO NOTHING;

-- ============================================================
-- 种子数据：保护材料数据库
-- ============================================================
INSERT INTO protection_materials (name, category, manufacturer, weather_resistance, permeability, adhesion, reversibility, durability, environmental_friendliness, cost_per_unit, coverage_rate, applicable_rock_types, construction_difficulty, lifespan_years, description) VALUES
('纳米氢氧化钙分散液', '纳米石灰', '中科院材科所', 7.5, 9.0, 7.0, 9.5, 8.0, 9.5, 280.00, 2.5, '["石灰岩","砂岩","砂砾岩"]', 2, 15.0, '纳米石灰，与碳酸盐岩石兼容性极佳，透气性好，环保可再处理'),
('甲基硅酸钾水溶液', '有机硅', '德国Wacker', 9.0, 8.5, 8.0, 5.0, 9.5, 7.0, 350.00, 5.0, '["砂岩","砂砾岩","石灰岩"]', 1, 25.0, '有机硅防水剂，耐候性好，渗透型防水'),
('正硅酸乙酯(TEOS)', '有机硅', '德国Degussa', 9.5, 7.0, 9.0, 4.0, 9.0, 6.5, 520.00, 4.0, '["砂岩","石灰岩","花岗岩"]', 3, 30.0, '溶胶凝胶法，形成SiO2网络，增强效果显著'),
('纳米TiO2光触媒涂层', '纳米材料', '中科院纳米所', 8.5, 8.0, 7.5, 6.0, 8.5, 9.0, 680.00, 3.5, '["石灰岩","砂岩","砂砾岩"]', 2, 20.0, '兼具保护与自清洁功能，环保无毒'),
('改性丙烯酸乳液', '丙烯酸', '美国Dow', 8.0, 6.5, 8.5, 5.5, 7.5, 6.0, 180.00, 8.0, '["砂砾岩","砂岩"]', 1, 12.0, '经济实用，耐候性良好，施工简便'),
('水性氟碳树脂', '氟碳树脂', '日本大金', 9.5, 5.5, 8.5, 3.5, 10.0, 5.0, 850.00, 6.0, '["石灰岩","砂岩","花岗岩"]', 3, 35.0, '耐久性极佳，耐候性最好，但透气性和可逆性一般'),
('环氧树脂加固剂', '环氧树脂', '美国Huntsman', 7.5, 3.0, 10.0, 2.0, 8.5, 4.5, 450.00, 1.5, '["所有类型"]', 4, 20.0, '结构加固专用，粘结强度极高，不适合表面封护'),
('纳米SiO2溶胶', '纳米材料', '清华大学材料系', 8.0, 8.5, 7.5, 8.0, 8.0, 9.5, 320.00, 3.0, '["石灰岩","砂岩","砂砾岩"]', 2, 18.0, '纳米二氧化硅，与石材成分一致，兼容性好'),
('硅丙乳液', '有机硅改性丙烯酸', '国内某厂', 8.5, 7.5, 8.0, 5.0, 8.5, 6.5, 220.00, 6.5, '["砂岩","砂砾岩"]', 1, 18.0, '有机硅改性丙烯酸，综合性能优良'),
('亚麻油-蜂蜡混合制剂', '传统材料', '传统工艺研究所', 5.0, 6.0, 6.5, 9.0, 5.0, 9.5, 120.00, 10.0, '["木材","砂岩"]', 1, 8.0, '传统保护材料，环保可再处理，适合修复性保护')
ON CONFLICT DO NOTHING;

-- ============================================================
-- 辅助视图：监测点最新传感器数据
-- ============================================================
CREATE OR REPLACE VIEW v_latest_sensor_data AS
SELECT DISTINCT ON (point_id)
    point_id,
    time,
    temperature,
    humidity,
    surface_hardness,
    crack_width,
    wind_speed,
    rainfall,
    solar_radiation
FROM sensor_data
ORDER BY point_id, time DESC;

-- ============================================================
-- 辅助视图：石窟监测点统计
-- ============================================================
CREATE OR REPLACE VIEW v_cave_statistics AS
SELECT
    c.id AS cave_id,
    c.name AS cave_name,
    c.province,
    c.rock_type,
    COUNT(mp.id) AS total_points,
    COUNT(a.id) FILTER (WHERE a.is_acknowledged = FALSE) AS active_alerts
FROM caves c
LEFT JOIN monitoring_points mp ON mp.cave_id = c.id
LEFT JOIN alerts a ON a.point_id = mp.id AND a.is_acknowledged = FALSE
GROUP BY c.id, c.name, c.province, c.rock_type;

-- ============================================================
-- 触发器：更新updated_at字段
-- ============================================================
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_caves_updated_at ON caves;
CREATE TRIGGER trg_caves_updated_at
    BEFORE UPDATE ON caves
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

DROP TRIGGER IF EXISTS trg_monitoring_points_updated_at ON monitoring_points;
CREATE TRIGGER trg_monitoring_points_updated_at
    BEFORE UPDATE ON monitoring_points
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

-- ============================================================
-- 连续聚合视图：按天聚合传感器数据
-- ============================================================
CREATE MATERIALIZED VIEW IF NOT EXISTS sensor_data_daily
WITH (timescaledb.continuous) AS
SELECT
    point_id,
    time_bucket('1 day', time) AS bucket_day,
    AVG(temperature) AS avg_temperature,
    MIN(temperature) AS min_temperature,
    MAX(temperature) AS max_temperature,
    AVG(humidity) AS avg_humidity,
    MIN(humidity) AS min_humidity,
    MAX(humidity) AS max_humidity,
    AVG(surface_hardness) AS avg_hardness,
    LAST(surface_hardness, time) AS last_hardness,
    FIRST(surface_hardness, time) AS first_hardness,
    AVG(crack_width) AS avg_crack_width,
    LAST(crack_width, time) AS last_crack_width,
    FIRST(crack_width, time) AS first_crack_width,
    SUM(rainfall) AS total_rainfall,
    AVG(solar_radiation) AS avg_solar_radiation
FROM sensor_data
GROUP BY point_id, time_bucket('1 day', time)
WITH NO DATA;

-- 添加刷新策略（每天刷新）
-- SELECT add_continuous_aggregate_policy('sensor_data_daily',
--     start_offset => INTERVAL '3 days',
--     end_offset => INTERVAL '1 hour',
--     schedule_interval => INTERVAL '1 day');

-- ============================================================
-- 数据生成辅助函数（用于测试）
-- ============================================================
CREATE OR REPLACE FUNCTION generate_mock_monitoring_points()
RETURNS VOID AS $$
DECLARE
    cave RECORD;
    point_name TEXT;
    point_code TEXT;
    rock_type_var TEXT;
    hardness_base DECIMAL;
    pos_x DECIMAL;
    pos_y DECIMAL;
    pos_z DECIMAL;
BEGIN
    FOR cave IN SELECT * FROM caves LOOP
        FOR i IN 1..20 LOOP
            point_name := cave.name || '监测点#' || LPAD(i::TEXT, 3, '0');
            point_code := 'MP' || cave.id || '_' || LPAD(i::TEXT, 3, '0');
            rock_type_var := cave.rock_type;
            
            -- 根据岩石类型设定基准硬度
            CASE rock_type_var
                WHEN '石灰岩' THEN hardness_base := 55 + (random() * 15);
                WHEN '砂岩' THEN hardness_base := 45 + (random() * 15);
                WHEN '砂砾岩' THEN hardness_base := 30 + (random() * 15);
                ELSE hardness_base := 40 + (random() * 20);
            END CASE;
            
            pos_x := (random() - 0.5) * 100;
            pos_y := (random() - 0.5) * 100;
            pos_z := random() * 50;
            
            INSERT INTO monitoring_points (cave_id, name, code, position_x, position_y, position_z, rock_type, initial_hardness, initial_crack_width, area_description, status)
            VALUES (
                cave.id,
                point_name,
                point_code,
                pos_x,
                pos_y,
                pos_z,
                rock_type_var,
                hardness_base,
                CASE WHEN random() < 0.3 THEN random() * 0.3 ELSE 0 END,
                '区域编号A' || i,
                1
            ) ON CONFLICT (code) DO NOTHING;
        END LOOP;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- 生成监测点数据
SELECT generate_mock_monitoring_points();

-- ============================================================
-- 历史数据生成函数
-- ============================================================
CREATE OR REPLACE FUNCTION generate_mock_sensor_data(days_back INTEGER DEFAULT 365)
RETURNS VOID AS $$
DECLARE
    point RECORD;
    current_time TIMESTAMPTZ;
    base_temp DECIMAL;
    base_humidity DECIMAL;
    temp_var DECIMAL;
    temp_decay DECIMAL;
    crack_growth DECIMAL;
    crack_var DECIMAL;
    hardness_drop DECIMAL;
    v_temp DECIMAL;
    v_hum DECIMAL;
    v_hard DECIMAL;
    v_crack DECIMAL;
    hour_offset INTEGER;
BEGIN
    FOR point IN SELECT * FROM monitoring_points LOOP
        -- 根据省份确定基准温湿度
        CASE
            WHEN point.rock_type = '砂砾岩' THEN 
                base_temp := 12 + random() * 6;
                base_humidity := 35 + random() * 15;
            WHEN point.rock_type = '砂岩' THEN
                base_temp := 10 + random() * 8;
                base_humidity := 45 + random() * 15;
            WHEN point.rock_type = '石灰岩' THEN
                base_temp := 14 + random() * 6;
                base_humidity := 55 + random() * 15;
            ELSE
                base_temp := 12 + random() * 8;
                base_humidity := 45 + random() * 20;
        END CASE;
        
        -- 计算整体风化衰减
        temp_decay := days_back / 365.0;
        hardness_drop := (random() * 5 + 1) * temp_decay;  -- 1~6%年降
        crack_growth := (random() * 0.2 + 0.05) * temp_decay;  -- 0.05~0.25mm/年增长
        
        FOR hour_offset IN REVERSE (days_back * 24)..0 LOOP
            current_time := NOW() - (hour_offset || ' hours')::INTERVAL;
            
            -- 季节变化 (年周期)
            temp_var := 12 * SIN(2 * PI() * EXTRACT(DOY FROM current_time) / 365 - PI() / 2);
            -- 日变化
            temp_var := temp_var + 5 * SIN(2 * PI() * EXTRACT(HOUR FROM current_time) / 24 - PI() / 2);
            
            v_temp := base_temp + temp_var + (random() - 0.5) * 3;
            v_hum := base_humidity - 8 * SIN(2 * PI() * EXTRACT(DOY FROM current_time) / 365 - PI() / 2) * 0.3 
                     - 5 * SIN(2 * PI() * EXTRACT(HOUR FROM current_time) / 24 - PI() / 2) * 0.2
                     + (random() - 0.5) * 8;
            v_hum := GREATEST(5, LEAST(95, v_hum));
            
            -- 硬度：随时间下降 + 波动
            v_hard := point.initial_hardness 
                      - hardness_drop * (1 - hour_offset::DECIMAL / (days_back * 24))
                      + (random() - 0.5) * 0.5;
            
            -- 裂隙宽度：随时间增长 + 波动
            crack_var := crack_growth * (1 - hour_offset::DECIMAL / (days_back * 24));
            v_crack := point.initial_crack_width + crack_var + (random() - 0.5) * 0.02;
            v_crack := GREATEST(0, v_crack);
            
            -- 每400小时插入一批
            IF hour_offset % 1 = 0 THEN
                INSERT INTO sensor_data (
                    time, point_id, temperature, humidity, surface_hardness, crack_width,
                    wind_speed, rainfall, solar_radiation, co2_concentration, vibration
                ) VALUES (
                    current_time,
                    point.id,
                    v_temp,
                    v_hum,
                    v_hard,
                    v_crack,
                    random() * 8,
                    CASE WHEN random() < 0.02 THEN random() * 15 ELSE 0 END,
                    GREATEST(0, 600 * SIN(2 * PI() * EXTRACT(HOUR FROM current_time) / 24 - PI() / 2) + (random() - 0.5) * 100),
                    400 + random() * 100,
                    random() * 0.05
                );
            END IF;
        END LOOP;
        
        -- 每1000条提交一次，防止日志膨胀
        COMMIT;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- 注意：以下函数调用会生成大量数据，首次初始化时可以只生成少量数据或跳过
-- SELECT generate_mock_sensor_data(30);  -- 生成30天数据用于测试
