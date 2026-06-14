# 石窟监测系统 (Grotto Monitor)

基于 Go + TimescaleDB 的石窟文物监测系统，支持 10 处石窟群、多监测点的温湿度和风化数据实时采集、预测与告警。

## 目录

- [系统架构](#系统架构)
- [技术栈](#技术栈)
- [项目结构](#项目结构)
- [快速部署](#快速部署)
- [传感器模拟器](#传感器模拟器)
- [API 接口](#api-接口)
- [可观测性](#可观测性)
- [配置说明](#配置说明)

---

## 系统架构

### 整体架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                        Nginx (端口 80)                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐  │
│  │  静态文件服务  │  │  API 代理     │  │  WebSocket 代理      │  │
│  │  (Gzip 压缩)  │  │  (/api/*)    │  │  (/ws/*)             │  │
│  └──────────────┘  └──────────────┘  └──────────────────────┘  │
└────────────────────────────────┬────────────────────────────────┘
                                 │
        ┌────────────────────────┼────────────────────────┐
        │                        │                        │
        ▼                        ▼                        ▼
┌───────────────┐       ┌────────────────┐       ┌───────────────────┐
│  Go 后端服务   │       │  Prometheus    │       │  Grafana (3000)   │
│   (端口 8080)  │──────▶│   (9090)       │──────▶│  可视化仪表盘      │
│  ┌───────────┐ │       └────────────────┘       └───────────────────┘
│  │ pprof     │ │
│  │ /debug/pprof│ │
│  ├───────────┤ │
│  │ metrics   │ │
│  │ /metrics  │ │
│  └───────────┘ │
└───────┬───────┘
        │
        │ 传感器数据上报
        ▼
┌───────────────┐       ┌──────────────────┐
│ TimescaleDB   │◀──────│  传感器模拟器     │
│  (端口 5432)  │       │   (端口 8081)    │
│  - 超表存储    │       │  - 10 处石窟群   │
│  - 自动压缩    │       │  - 40 个监测点   │
│  - 连续聚合    │       │  - 1 小时间隔    │
└───────────────┘       │  - 数据注入 API   │
                        └──────────────────┘
```

### 组件说明

| 组件 | 端口 | 作用 |
|------|------|------|
| **Nginx** | 80 | 前端静态文件服务（Gzip 压缩）、API 代理、WebSocket 代理 |
| **Go 后端** | 8080 | 业务逻辑、传感器数据接收、风化预测、告警推送、pprof/metrics |
| **TimescaleDB** | 5432 | 时序数据库，存储监测数据，自动压缩策略 |
| **传感器模拟器** | 8081 | 模拟 10 处石窟群的传感器数据，支持数据注入 |
| **Prometheus** | 9090 | 监控指标采集与存储 |
| **Grafana** | 3000 | 可视化监控仪表盘 |

---

## 技术栈

### 后端
- **语言**: Go 1.21
- **Web 框架**: Gorilla Mux
- **数据库**: PostgreSQL + TimescaleDB
- **ORM**: database/sql (原生 SQL)
- **监控**: Prometheus client + pprof
- **实时通信**: Gorilla WebSocket

### 数据库
- **时序数据库**: TimescaleDB (基于 PostgreSQL 16)
- **特性**: 超表 (Hypertable)、连续聚合 (Continuous Aggregate)、自动压缩 (Compression)
- **索引**: 时间降序索引、传感器 ID 索引、站点 ID 索引

### 前端
- **原生 HTML/CSS/JavaScript**
- **3D 可视化**: Three.js
- **热力图**: Canvas 2D
- **WebSocket 实时告警**

### 部署
- **容器编排**: Docker Compose
- **反向代理**: Nginx (Gzip 压缩)
- **监控**: Prometheus + Grafana

---

## 项目结构

```
grotto-monitor/
├── backend/                    # Go 后端服务
│   ├── main.go                 # 主入口 (pprof + Prometheus + Gzip)
│   ├── go.mod
│   ├── alarm_websocket/        # WebSocket 告警推送
│   ├── config/                 # 模型参数配置
│   ├── material_optimizer/     # 材料优化 (TOPSIS)
│   ├── models/                 # 数据模型
│   ├── repository/             # 数据库层
│   ├── sensor_receiver/        # 传感器数据接收
│   ├── simulator/              # 传感器模拟器
│   └── weathering_predictor/   # 风化预测
├── frontend/                   # 前端页面
│   ├── index.html
│   ├── css/style.css
│   └── js/
├── db/
│   ├── init.sql                # 数据库初始化脚本
│   └── postgresql.conf         # PostgreSQL 配置
├── nginx/
│   ├── nginx.conf              # Nginx 主配置 (Gzip)
│   └── default.conf            # 站点配置
├── prometheus/
│   └── prometheus.yml          # Prometheus 配置
├── grafana/
│   ├── provisioning/           # Grafana 预置配置
│   │   ├── datasources/
│   │   └── dashboards/
│   └── dashboards/
│       └── grotto-dashboard.json
├── .env                        # 环境变量配置
├── docker-compose.yml          # Docker Compose 编排
├── Dockerfile.backend          # 后端镜像构建
└── Dockerfile.simulator        # 模拟器镜像构建
```

---

## 快速部署

### 前置要求

- Docker 20.10+
- Docker Compose v2+
- 至少 4GB 可用内存
- 至少 10GB 可用磁盘空间

### 一键启动

```bash
# 克隆或进入项目目录
cd grotto-monitor

# 启动所有服务
docker-compose up -d

# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f backend
```

### 服务访问地址

| 服务 | 地址 | 说明 |
|------|------|------|
| **前端页面** | http://localhost | 石窟监测主界面 |
| **后端 API** | http://localhost/api/ | REST API 接口 |
| **Grafana** | http://localhost:3000 | 监控仪表盘 (admin/admin123) |
| **Prometheus** | http://localhost:9090 | Prometheus UI |
| **pprof** | http://localhost:8080/debug/pprof/ | Go 性能分析 |
| **Metrics** | http://localhost:8080/metrics | Prometheus 指标 |
| **模拟器控制** | http://localhost:8081/status | 模拟器状态与控制 |

### 首次启动说明

1. **数据库初始化**: TimescaleDB 首次启动会自动执行 `db/init.sql`，创建 10 处石窟群和 40 个传感器
2. **历史数据回灌**: 模拟器启动后会自动回灌 30 天的历史数据（可配置）
3. **实时数据上报**: 回灌完成后，模拟器以 1 小时间隔持续上报数据
4. **Grafana 仪表盘**: 自动加载预置的石窟监测仪表盘

### 停止服务

```bash
# 停止所有服务
docker-compose down

# 停止并清除数据（慎用）
docker-compose down -v
```

---

## 传感器模拟器

### 功能特性

- ✅ **10 处石窟群**: 敦煌莫高窟、云冈石窟、龙门石窟、麦积山石窟、大足石刻、响堂山石窟、巩义石窟、炳灵寺石窟、克孜尔石窟、须弥山石窟
- ✅ **40 个监测点**: 每处石窟群 4 个传感器（温度、湿度、表面硬度、裂隙宽度）
- ✅ **1 小时间隔**: 按小时周期模拟温湿度日变化和季节性变化
- ✅ **风化模拟**: 表面硬度缓慢下降、裂隙宽度缓慢扩张
- ✅ **数据注入 API**: 可手动注入异常数据，模拟极端场景
- ✅ **数据覆盖 API**: 可固定传感器值，用于告警测试

### 石窟群列表

| ID | 名称 | 位置 | 岩石类型 | 传感器数 |
|----|------|------|----------|----------|
| 1 | 敦煌莫高窟 | 甘肃省敦煌市 | 砂岩 | 4 |
| 2 | 云冈石窟 | 山西省大同市 | 砂岩 | 4 |
| 3 | 龙门石窟 | 河南省洛阳市 | 石灰岩 | 4 |
| 4 | 麦积山石窟 | 甘肃省天水市 | 砂岩 | 4 |
| 5 | 大足石刻 | 重庆市大足区 | 砂岩 | 4 |
| 6 | 响堂山石窟 | 河北省邯郸市 | 石灰岩 | 4 |
| 7 | 巩义石窟 | 河南省巩义市 | 砂岩 | 4 |
| 8 | 炳灵寺石窟 | 甘肃省永靖县 | 砂岩 | 4 |
| 9 | 克孜尔石窟 | 新疆拜城县 | 砂岩 | 4 |
| 10 | 须弥山石窟 | 宁夏固原市 | 砂岩 | 4 |

### 模拟器控制 API

#### 1. 获取状态

```bash
GET http://localhost:8081/status
```

返回示例：
```json
{
  "running": true,
  "interval_hours": 1,
  "backfill_days": 30,
  "api_base_url": "http://backend:8080",
  "sensor_count": 40,
  "site_count": 10,
  "total_data_sent": 28800,
  "last_report_time": "2024-01-15T10:00:00Z"
}
```

#### 2. 获取传感器列表

```bash
GET http://localhost:8081/sensors
```

#### 3. 数据注入

向指定传感器注入数据，可用于模拟异常场景。

```bash
POST http://localhost:8081/inject
Content-Type: application/json

{
  "sensor_id": 1,
  "site_id": null,
  "sensor_type": null,
  "value": 45.0,
  "start_time": "2024-01-15T10:00:00Z",
  "count": 24
}
```

**参数说明：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `sensor_id` | int | 否 | 指定传感器 ID |
| `site_id` | int | 否 | 指定石窟群 ID（该石窟群所有传感器） |
| `sensor_type` | string | 否 | 指定传感器类型（如 "温度"、"湿度"） |
| `value` | float | 是 | 注入的值 |
| `start_time` | datetime | 否 | 开始时间，默认当前时间 |
| `count` | int | 否 | 注入数据点数（按 1 小时间隔），默认 1 |
| `duration` | string | 否 | 持续时间，如 "24h"、"7d" |

**注入示例：**

```bash
# 向敦煌莫高窟的温度传感器注入高温（模拟极端天气）
curl -X POST http://localhost:8081/inject \
  -H "Content-Type: application/json" \
  -d '{
    "site_id": 1,
    "sensor_type": "温度",
    "value": 45.5,
    "duration": "48h"
  }'

# 向所有表面硬度传感器注入低值（模拟风化加剧）
curl -X POST http://localhost:8081/inject \
  -H "Content-Type: application/json" \
  -d '{
    "sensor_type": "表面硬度",
    "value": 30.0,
    "count": 168
  }'

# 向特定裂隙传感器注入大值（触发告警）
curl -X POST http://localhost:8081/inject \
  -H "Content-Type: application/json" \
  -d '{
    "sensor_id": 4,
    "value": 0.8,
    "count": 5
  }'
```

#### 4. 值覆盖（Override）

设置传感器值固定为某个值，用于持续测试告警。

```bash
POST http://localhost:8081/override
Content-Type: application/json

{
  "site_id": 1,
  "sensor_type": "裂隙宽度",
  "value": 0.6
}
```

**重置覆盖：**

```bash
POST http://localhost:8081/override
Content-Type: application/json

{
  "site_id": 1,
  "sensor_type": "裂隙宽度",
  "reset": true
}
```

### 模拟器环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `API_HOST` | localhost | 后端 API 地址 |
| `API_PORT` | 8080 | 后端 API 端口 |
| `CONTROL_PORT` | 8081 | 模拟器控制端口 |
| `INTERVAL_HOURS` | 1 | 数据上报间隔（小时） |
| `BACKFILL_DAYS` | 30 | 历史数据回灌天数 |
| `BACKFILL` | true | 是否启用历史数据回灌 |

---

## API 接口

### 石窟群与传感器

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/sites` | 获取所有石窟群列表 |
| GET | `/api/sites/{id}` | 获取单个石窟群详情 |
| GET | `/api/sites/{id}/sensors` | 获取石窟群下的所有传感器 |
| GET | `/api/sites/{id}/data` | 获取石窟群监测数据 |
| GET | `/api/sensors/{id}/hourly` | 获取传感器小时级聚合数据 |

### 数据上报

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/data` | 上报传感器数据 |

### 风化预测

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/predict` | 单传感器风化预测 |
| POST | `/api/predict/batch` | 批量风化预测 |

### 材料优化

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/materials` | 获取防护材料列表 |
| POST | `/api/topsis` | TOPSIS 材料优化分析 |

### 告警

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/alerts` | 获取告警列表 |
| PUT | `/api/alerts/{id}/acknowledge` | 确认告警 |
| WS | `/ws/alerts` | WebSocket 实时告警推送 |

---

## 可观测性

### Go pprof 性能分析

后端集成了 Go 标准库的 pprof，可用于性能分析。

| 路径 | 说明 |
|------|------|
| `/debug/pprof/` | pprof 首页 |
| `/debug/pprof/heap` | 内存堆分析 |
| `/debug/pprof/goroutine` | Goroutine 分析 |
| `/debug/pprof/profile` | CPU 性能分析 |
| `/debug/pprof/block` | 阻塞分析 |
| `/debug/pprof/mutex` | 锁竞争分析 |
| `/debug/pprof/trace` | 执行追踪 |

**使用示例：**

```bash
# CPU 性能分析（30秒）
go tool pprof http://localhost:8080/debug/pprof/profile?seconds=30

# 内存分析
go tool pprof http://localhost:8080/debug/pprof/heap

# Goroutine 分析
go tool pprof http://localhost:8080/debug/pprof/goroutine
```

### Prometheus 监控指标

后端暴露 `/metrics` 端点，提供以下指标：

| 指标名 | 类型 | 说明 |
|--------|------|------|
| `http_requests_total` | Counter | HTTP 请求总数（按 method、path、status 标签） |
| `http_request_duration_seconds` | Histogram | HTTP 请求延迟分布 |
| `sensor_data_received_total` | Counter | 接收的传感器数据点总数 |
| `alerts_triggered_total` | Counter | 触发的告警总数（按 site_id、alert_type、severity 标签） |
| `active_websocket_connections` | Gauge | 当前活跃 WebSocket 连接数 |
| `database_query_duration_seconds` | Histogram | 数据库查询延迟 |

**在 Prometheus UI 中查询：**

```promql
# 请求速率
rate(http_requests_total[5m])

# P95 延迟
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))

# 传感器数据接收速率
rate(sensor_data_received_total[5m])
```

### Grafana 仪表盘

预置了石窟监测仪表盘，包含：

- **概览统计**: 传感器数据总量、活跃连接数、告警总数、请求速率
- **性能指标**: HTTP 请求延迟 P95、请求速率趋势
- **监测数据**: 各石窟群温度/湿度 24 小时趋势图
- **数据表格**: 石窟群监测概览、最近告警记录

访问地址: http://localhost:3000 (默认账号: admin / admin123)

---

## TimescaleDB 自动压缩

系统配置了 TimescaleDB 的自动压缩策略，以优化存储和查询性能。

### 压缩配置

| 参数 | 值 | 说明 |
|------|-----|------|
| 压缩触发时间 | 7 天 | 超过 7 天的数据自动压缩 |
| 调度间隔 | 1 天 | 每天执行一次压缩任务 |
| 分段字段 | site_id, sensor_id | 按站点和传感器分段 |
| 排序字段 | time DESC | 按时间降序排列 |
| Chunk 时间间隔 | 1 天 | 每个 Chunk 覆盖 1 天数据 |

### 连续聚合

配置了小时级连续聚合视图 `monitoring_hourly`，自动预计算每小时的统计数据。

| 参数 | 值 |
|------|-----|
| 聚合粒度 | 1 小时 |
| 刷新策略 | 每小时刷新一次 |
| 刷新范围 | 过去 3 小时到 1 小时前 |

### 验证压缩状态

```sql
-- 查看压缩后的 Chunk 数量
SELECT count(*) FROM timescaledb_information.chunks WHERE is_compressed = true;

-- 查看压缩率
SELECT
  hypertable_name,
  pg_size_pretty(before_compression_total_bytes) as before,
  pg_size_pretty(after_compression_total_bytes) as after,
  round((1 - (after_compression_total_bytes::numeric / before_compression_total_bytes::numeric)) * 100, 2) as compression_ratio
FROM timescaledb_information.compression_stats;
```

---

## 前端 Gzip 压缩

Nginx 配置了 Gzip 压缩，减少前端资源传输体积。

### Gzip 配置

| 参数 | 值 |
|------|-----|
| 压缩级别 | 6 (平衡压缩率与 CPU) |
| 最小文件大小 | 256 字节 |
| 支持类型 | text/plain, text/css, application/json, application/javascript, image/svg+xml, 字体文件 |

### 验证 Gzip

```bash
curl -I -H "Accept-Encoding: gzip" http://localhost/
# 响应头应包含: Content-Encoding: gzip
```

---

## 配置说明

### 环境变量 (.env)

```env
# 数据库配置
DB_HOST=timescaledb
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=grotto_monitor
DB_SSLMODE=disable

# 后端配置
API_PORT=8080

# 模拟器配置
API_HOST=backend
API_PORT=8080
INTERVAL_HOURS=1
BACKFILL_DAYS=30
BACKFILL=true
SIMULATOR_PORT=8081

# Nginx 配置
NGINX_PORT=80

# Prometheus 配置
PROM_PORT=9090

# Grafana 配置
GRAFANA_PORT=3000
GRAFANA_USER=admin
GRAFANA_PASSWORD=admin123
```

### 自定义配置

修改 `.env` 文件后重启服务：

```bash
docker-compose down
docker-compose up -d
```

---

## 常见问题

### 1. TimescaleDB 启动慢？

首次启动需要初始化数据库和扩展，可能需要 1-2 分钟。可通过健康检查确认状态：

```bash
docker-compose ps timescaledb
```

### 2. 如何重置所有数据？

```bash
docker-compose down -v
docker-compose up -d
```

**注意**: 这会删除所有持久化数据，包括数据库、Prometheus 和 Grafana 数据。

### 3. 模拟器数据上报延迟？

模拟器在 Docker 中运行时，1 小时的间隔是真实时间。如需快速测试，可修改 `INTERVAL_HOURS` 为更小的值（如 0.01 小时 ≈ 36 秒）。

### 4. 如何查看日志？

```bash
# 查看所有服务日志
docker-compose logs -f

# 查看特定服务日志
docker-compose logs -f backend
docker-compose logs -f simulator
docker-compose logs -f timescaledb
```

---

## License

MIT
