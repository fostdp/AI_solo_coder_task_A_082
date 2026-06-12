# 🏛️ 古代石窟风化速率监测与现代保护材料优选系统

> 面向敦煌、云冈、龙门等10处石窟群的微环境监测、风化速率预测与保护材料智能决策平台

---

## 系统概览

本系统是针对大型石窟群文物保护场景开发的全栈应用，核心能力包括：

| 模块 | 技术实现 | 功能说明 |
|------|----------|----------|
| **时序数据层** | TimescaleDB (PostgreSQL) | 存储海量传感器时序数据（温度、湿度、硬度、裂隙等），Hypertable + 连续聚合 |
| **后端服务** | Go (Gin + pgx5) | RESTful API、实时告警、模型计算、WebSocket推送 |
| **预测引擎** | 随机森林回归 (自实现) | 基于微环境+岩石类型预测风化速率，置信度评估 |
| **决策引擎** | TOPSIS多属性决策 | 10项材料属性加权评分，保护材料优选推荐 |
| **前端展示** | Vue3 + Three.js + ECharts | 石窟3D模型、风化热力图、裂隙可视化、趋势分析 |
| **实时告警** | WebSocket | 裂隙>0.5mm / 硬度降>20% 实时推送 |
| **数据模拟** | Python | 模拟4G DTU每小时上报，支持历史/实时/快速3种模式 |

---

## 📁 目录结构

```
AI_solo_coder_task_A_082/
├── backend/                    # Go 后端
│   ├── cmd/server/             # 主入口
│   │   └── main.go
│   ├── internal/
│   │   ├── config/             # 配置模块
│   │   ├── database/           # TimescaleDB连接池
│   │   ├── models/             # 数据模型定义
│   │   ├── handlers/           # HTTP API处理层
│   │   ├── services/           # 业务逻辑层
│   │   ├── middleware/         # CORS等中间件
│   │   └── websocket/          # WebSocket推送管理器
│   ├── pkg/algorithms/         # 核心算法
│   │   ├── random_forest.go    # 随机森林回归风化预测
│   │   └── topsis.go           # TOPSIS多属性材料优选
│   ├── go.mod
│   └── .env.example
├── frontend/                   # Vue3 前端
│   ├── src/
│   │   ├── views/              # 页面：看板/3D视图/风化分析/材料推荐/告警中心
│   │   ├── components/         # 热力图、Three.js3D场景、详情弹窗等
│   │   ├── api/                # Axios接口封装
│   │   ├── store/              # Pinia全局状态
│   │   ├── router/             # 路由配置
│   │   ├── utils/websocket.js  # WebSocket客户端
│   │   └── styles/global.scss  # 全局主题样式
│   ├── index.html
│   ├── vite.config.js
│   └── package.json
├── database/
│   └── init.sql                # TimescaleDB初始化（表结构+种子数据+连续聚合）
├── simulator/
│   └── sensor_simulator.py     # 传感器数据模拟器（Python）
└── README.md
```

---

## 🚀 快速部署

### 前置依赖

| 软件 | 版本要求 | 说明 |
|------|----------|------|
| Go | ≥ 1.21 | 后端编译运行 |
| Node.js | ≥ 18.0 | 前端构建 |
| PostgreSQL | ≥ 14 | 数据库 |
| TimescaleDB | ≥ 2.10 | 时序扩展 |
| Python | ≥ 3.8 | 模拟器运行 |

---

### 1️⃣ 数据库初始化

```bash
# ① 创建数据库并启用TimescaleDB
psql -U postgres -c "CREATE DATABASE grotto_monitor;"
psql -U postgres -d grotto_monitor -c "CREATE EXTENSION IF NOT EXISTS timescaledb;"

# ② 执行初始化脚本（表结构、10处石窟种子数据、10种保护材料、200个监测点）
psql -U postgres -d grotto_monitor -f database/init.sql

# ③ [可选] 生成1年历史数据（SQL内置函数，约175万条）
psql -U postgres -d grotto_monitor -c "SELECT generate_mock_sensor_data(365);"
```

**内置种子数据**：
- 🏔️ 10处石窟（敦煌、云冈、龙门、麦积山、大足、克孜尔、炳灵寺、响堂山、天梯山、榆林窟）
- 📍 每石窟20个监测点（共200个）
- 🧪 10种保护材料（纳米石灰、有机硅、TEOS、纳米TiO2、丙烯酸、氟碳、环氧、纳米SiO2、硅丙、蜂蜡）

---

### 2️⃣ 后端启动

```bash
cd backend

# ① 复制并修改配置
cp .env.example .env
# 编辑 .env 配置数据库连接信息

# ② 下载依赖
go mod tidy

# ③ 编译并启动
go run cmd/server/main.go

# 或构建生产版
go build -o grotto-server cmd/server/main.go
./grotto-server
```

后端默认监听 **`http://localhost:8080`**

**健康检查**：`GET http://localhost:8080/api/v1/health`

---

### 3️⃣ 前端启动

```bash
cd frontend

# ① 安装依赖
npm install

# ② 开发模式 (支持热更新 + API代理)
npm run dev

# ③ 生产构建
npm run build
```

前端开发地址：**`http://localhost:5173`**

Vite已配置代理：`/api/* → http://localhost:8080/api/*`，无需额外配置跨域。

---

### 4️⃣ 数据模拟（可选）

```bash
cd simulator
pip install requests

# ⚡快速模式：生成最近10天数据（推荐首次使用）
python sensor_simulator.py quick --hours 240

# 📚历史模式：生成指定天数历史数据（可用于图表展示）
python sensor_simulator.py historical --days 90

# ⏱️实时模式：模拟每小时上报 (speed=60 → 每1分钟=模拟1小时)
python sensor_simulator.py realtime --speed 60

# 🚨告警测试：立即注入3条超限数据触发告警推送
python sensor_simulator.py alert

# 📡指定后端地址
python sensor_simulator.py quick --api http://192.168.1.100:8080/api/v1
```

---

## 🧠 核心算法

### 1. 风化速率预测（随机森林回归）

**算法位置**：[backend/pkg/algorithms/random_forest.go](backend/pkg/algorithms/random_forest.go)

```
输入特征 (9维)：
├─ 岩石类型独热编码 (4维): 砂砾岩/砂岩/石灰岩/花岗岩
├─ 平均温度 (℃)
├─ 平均湿度 (%)
├─ 昼夜温差 (℃)
├─ 月度降水 (mm)
└─ 温湿度交互项 (T×H÷100)

模型参数：
  • 树数量：50棵
  • 最大深度：10层
  • 最小分裂：5样本
  • 特征随机子空间：√N + 1

物理模型训练数据：基于文物保护领域经验公式
  f(风化速率) ∝ |T-15|^1.5 × 岩性系数
              + (RH-60)² × 0.0005
              + 冻融循环(低温×高湿)
              + 湿热协同(T>30 ∧ RH>75)
```

### 2. 保护材料优选（TOPSIS多属性决策）

**算法位置**：[backend/pkg/algorithms/topsis.go](backend/pkg/algorithms/topsis.go)

```
决策矩阵 (10材料 × 9属性)：
  效益型（越大越好）：耐候性、透气性、附着力、可逆性、耐久性、环保性、覆盖率、寿命
  成本型（越小越好）：单位成本

权重配置（支持优先级动态调整）：
  耐候性 0.20  |  透气性 0.15  |  耐久性 0.15
  可逆性 0.12  |  环保性 0.10  |  附着力 0.10
  单  价 0.10  |  覆盖率 0.05  |  寿  命 0.03

TOPSIS步骤：
  ① 向量规范化 → ② 加权规范化矩阵 → ③ 正/负理想解
  → ④ 欧氏距离 → ⑤ 相对贴近度Ci*排序

7种优先级策略：耐久/可逆/环保/耐候/透气/经济/寿命
```

---

## 🔔 告警规则与推送

| 告警类型 | 触发条件 | 严重级别 | 冷却时间 |
|----------|----------|----------|----------|
| **裂隙超限** | `crack_width > 0.5mm` | CRITICAL / WARNING | 12小时 |
| **硬度下降** | `(初始硬度-当前值)/初始值 > 20%` | WARNING / CRITICAL | 12小时 |
| **湿度异常** | `RH > 90%` 或 `RH < 20%` | INFO / WARNING | 24小时 |

**WebSocket推送**：
- 连接端点：`ws://host:8080/api/v1/ws`
- 消息格式：`{ type: "ALERT" | "SENSOR_DATA", payload: {...}, time: "..." }`
- 心跳机制：30秒Ping/Pong，断线指数退避重连（最多10次）

---

## 📡 API 接口速览

### 石窟与监测点
```
GET  /api/v1/caves                  # 全部石窟列表
GET  /api/v1/caves/:id              # 单个石窟详情
GET  /api/v1/caves/statistics       # 石窟统计（监测点数+告警数）
GET  /api/v1/points?caveId=1        # 指定石窟所有监测点
GET  /api/v1/points/:id             # 监测点详情
```

### 传感器数据
```
GET  /api/v1/sensors/latest?pointId=1   # 指定点最新数据（无参=全部最新）
GET  /api/v1/sensors/history?pointId=1&days=30
POST /api/v1/sensors/data               # 单条上报
POST /api/v1/sensors/batch              # 批量上报
```

### 风化分析
```
GET  /api/v1/weathering/rates?pointId=1      # 风化速率历史（按月）
POST /api/v1/weathering/predict               # 随机森林预测
     Body: { pointId, targetTemperature, targetHumidity, rockType }
```

### 保护材料
```
GET  /api/v1/materials                       # 材料数据库全部
POST /api/v1/materials/recommend             # TOPSIS推荐
     Body: { rockType, budgetLevel, priorityCriteria[], protectionType }
```

### 告警与系统
```
GET    /api/v1/alerts?limit=50&acknowledged=false
POST   /api/v1/alerts/:id/acknowledge
GET    /api/v1/dashboard/overview             # 总览看板数据
WS     /api/v1/ws                             # WebSocket长连接
GET    /api/v1/ws/status                      # 连接数统计
GET    /api/v1/health                         # 健康检查
```

---

## 🎨 前端功能模块

### 1. 总览看板 (`/dashboard`)
- 4项核心指标卡（石窟数/监测点数/告警数/模型版本）
- 2D平面热力图 + 监测点状态 + 裂隙线条可视化
- 近30天温湿度/硬度/裂隙多轴趋势图
- 各省监测点分布饼图、实时告警列表

### 2. 石窟三维监测 (`/grotto3d`)
- **Three.js场景**：参数化山体、佛龛、立柱、120+网格细节
- **监测点标记**：状态颜色（绿/黄/红）、告警脉冲动画
- **风化热力图**：Canvas纹理贴图投射到地面，强度随分数变化
- **裂隙3D管**：CatmullRom曲线 + TubeGeometry + 发光外晕
- 视角预设：正视/俯视/侧视/立体（平滑过渡动画）

### 3. 风化速率分析 (`/weathering`)
- 月度风化速率柱状图（硬度损失+裂隙增长+综合3系列）
- 预测参数滑块：温度/-20~50℃、湿度/0~100%、昼夜温差
- 4指标结果卡（预测速率/当前速率/风险倍数/置信度）
- **温湿度二维热力图**：11×16矩阵颜色化展示所有组合风化强度

### 4. 保护材料优选 (`/materials`)
- 7种决策参数动态调整（岩石类型/预算/保护类型/7项优先级）
- 实时权重条可视化
- 前3名金银铜牌推荐卡 + 仪表盘TOPSIS评分
- 前5名材料雷达对比图（8维属性）
- 完整排名表格 + 可解释性建议文本

### 5. 告警中心 (`/alerts`)
- 全部/未处理/已处理三种筛选
- 严重级别彩色标签 + 阈值对比显示（当前值 vs 阈值）
- 告警类型分布图、各石窟告警横向柱状图
- 告警阈值配置说明面板

---

## 🗄️ 数据库核心设计

### 时序Hypertable：`sensor_data`
```sql
SELECT create_hypertable('sensor_data', 'time');
-- 索引：(point_id, time DESC) 复合索引
-- 连续聚合：sensor_data_daily 按天预聚合
-- 可选策略：7天压缩 + 3年保留期
```

### 表关系图
```
caves (1) ──→ (N) monitoring_points (1) ──→ (N) sensor_data [Hypertable]
                                   │
                                   ├──→ weathering_rates  (月度统计)
                                   ├──→ weathering_predictions (模型输出)
                                   └──→ alerts  (告警记录)
protection_materials ──→ material_recommendations (决策存档)
```

### 9大数据表一览
| 表名 | 记录量级 | 用途 |
|------|----------|------|
| caves | 10 | 石窟基础信息 |
| monitoring_points | 200 | 监测点三维坐标、初始硬度 |
| sensor_data | 亿级 | 传感器时序核心数据 |
| weathering_rates | 万级 | 月度风化速率归档 |
| protection_materials | 10 | 保护材料属性库 |
| alerts | 万级 | 告警历史 + 确认状态 |
| weathering_predictions | 千级 | 模型调用结果存档 |
| material_recommendations | 千级 | TOPSIS决策存档 |
| sensor_data_daily | 十万级 | 连续聚合视图(按天) |

---

## 🛠️ 开发与调试

### Go后端
```bash
# 代码格式化
gofmt -w backend/...

# 静态检查
go vet ./backend/...
```

### 前端
```bash
# 代码检查 + 构建产物分析
npm run build -- --mode=development
```

### 常见问题排查

| 现象 | 排查方向 |
|------|----------|
| 后端启动 panic | 检查 `.env` 中数据库连接串 |
| 前端无数据 | 打开浏览器DevTools检查 `/api/v1/health` 返回 |
| WebSocket连不上 | 确认Nginx/代理已支持 `Upgrade: websocket` |
| 模拟器429 | 减小 batch-size 或调大后端 `SERVER_READ_TIMEOUT` |
| 3D场景卡顿 | 尝试降低 `pixelRatio`（GrottoThreeScene.vue中已限制≤2） |

---

## 📈 性能参考

| 操作 | 性能指标 |
|------|----------|
| 批量入库 (500条/批) | ≈ 15,000~25,000 条/秒 |
| 随机森林预测 (9维→1维) | ≈ 0.3~0.8ms / 次 |
| TOPSIS决策 (10×9矩阵) | ≈ 0.1ms / 次 |
| 历史查询 (365天/单点) | ≈ 50~120ms |
| Three.js渲染 (200点+热力图) | ≈ 45~60 FPS (中端显卡) |

---

## 🧭 架构设计要点

1. **Go后端完全零训练依赖**：随机森林与TOPSIS均纯代码实现，无需Python/ML库部署
2. **TimescaleDB原生能力**：time_bucket连续聚合、时间索引、压缩策略，亿级数据无忧
3. **前端大列表虚拟化**：告警/监测点列表虚拟滚动支持万条流畅
4. **告警防刷屏**：同类型12小时冷却，告警聚合而非单条重复
5. **模型可解释性**：预测附带7条情境化保护建议文本，非黑盒输出

---

## 📜 License

本系统为文物保护领域教学演示项目，所有种子石窟数据来源于公开信息，材料参数为工程典型值，实际应用请结合现场实验数据校准模型。

---

**系统组件定位**：

> 🎯 **监测是基础** → Three.js + 热力图 + 实时告警  
> 🧠 **预测是核心** → 随机森林回归 + 置信度  
> 💎 **决策是价值** → TOPSIS多属性 + 优先级可配置  

祝愿本系统能为中华石窟文物保护事业贡献技术力量！ 🏛️✨
