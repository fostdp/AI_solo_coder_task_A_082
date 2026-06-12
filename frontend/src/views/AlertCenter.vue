<template>
  <div class="alert-center">
    <el-row :gutter="16">
      <el-col :xs="24" :lg="16">
        <div class="panel-card">
          <div class="section-title">
            <el-icon style="color:var(--accent-red)"><Warning /></el-icon>
            <span>告警列表</span>
            <div class="filters" style="margin-left:auto">
              <el-radio-group v-model="filterAck" size="small" @change="loadAlerts">
                <el-radio-button :value="null">全部</el-radio-button>
                <el-radio-button :value="false">未处理</el-radio-button>
                <el-radio-button :value="true">已处理</el-radio-button>
              </el-radio-group>
            </div>
          </div>
          <el-table
            :data="alerts"
            stripe
            style="width:100%"
            max-height="72vh"
            row-key="id"
          >
            <el-table-column width="60" align="center">
              <template #default="{ row }">
                <span class="badge-tag" :class="severityClass(row.severity)" style="transform:scale(1.1)">
                  {{ severityText(row.severity) }}
                </span>
              </template>
            </el-table-column>
            <el-table-column prop="alertType" label="类型" width="130">
              <template #default="{ row }">
                <el-icon style="vertical-align:-2px">
                  <component :is="alertIcon(row.alertType)" />
                </el-icon>
                <span style="margin-left:4px">{{ alertTypeText(row.alertType) }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="title" label="告警标题" min-width="180" />
            <el-table-column prop="message" label="详情" min-width="260" show-overflow-tooltip />
            <el-table-column label="阈值对比" width="160">
              <template #default="{ row }">
                <div class="threshold-compare">
                  <span class="current" :class="{ over: isOver(row) }">
                    {{ row.currentValue?.toFixed(2) }}
                  </span>
                  <el-icon style="color:var(--text-muted)"><Right /></el-icon>
                  <span class="threshold">{{ row.thresholdValue?.toFixed(2) }}</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="监测点" width="130">
              <template #default="{ row }">
                <span v-if="getPoint(row.pointId)">{{ getPoint(row.pointId).name }}</span>
                <el-tag v-else type="info" size="small">ID:{{ row.pointId }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="createdAt" label="时间" width="160">
              <template #default="{ row }">{{ formatTime(row.createdAt) }}</template>
            </el-table-column>
            <el-table-column label="状态" width="90" align="center">
              <template #default="{ row }">
                <el-tag
                  v-if="row.isAcknowledged"
                  type="success" size="small" effect="plain"
                >已处理</el-tag>
                <el-button
                  v-else
                  type="primary" size="small" plain
                  @click="acknowledge(row.id)"
                  :loading="ackingId === row.id"
                >处理</el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </el-col>

      <el-col :xs="24" :lg="8">
        <div class="panel-card">
          <div class="section-title">
            <el-icon style="color:var(--accent-gold)"><DataBoard /></el-icon>
            <span>告警统计</span>
          </div>
          <div class="stat-grid">
            <div class="stat critical">
              <div class="num">{{ stats.CRITICAL }}</div>
              <div class="lbl">严重告警</div>
            </div>
            <div class="stat warning">
              <div class="num">{{ stats.WARNING }}</div>
              <div class="lbl">警告</div>
            </div>
            <div class="stat info">
              <div class="num">{{ stats.INFO }}</div>
              <div class="lbl">提示</div>
            </div>
            <div class="stat ack">
              <div class="num">{{ stats.acknowledged }}</div>
              <div class="lbl">已处理</div>
            </div>
          </div>

          <v-chart :option="typeChartOption" style="height:220px;width:100%;margin-top:12px" autoresize />
        </div>

        <div class="panel-card" style="margin-top:16px">
          <div class="section-title">
            <el-icon style="color:var(--accent-blue)"><PieChart /></el-icon>
            <span>各石窟告警分布</span>
          </div>
          <v-chart :option="caveChartOption" style="height:260px;width:100%" autoresize />
        </div>

        <div class="panel-card" style="margin-top:16px">
          <div class="section-title">
            <el-icon style="color:var(--accent-purple)"><Bell /></el-icon>
            <span>告警阈值配置</span>
          </div>
          <div class="threshold-config">
            <div class="config-item">
              <div class="config-label">裂隙宽度阈值</div>
              <el-slider
                :model-value="0.5"
                disabled
                :step="0.05"
                :min="0.1" :max="2"
                size="small"
                show-input
              />
              <div class="config-desc">超过 <b>0.5mm</b> 触发告警</div>
            </div>
            <div class="config-item">
              <div class="config-label">硬度下降阈值</div>
              <el-slider
                :model-value="20"
                disabled
                :step="1"
                :min="5" :max="50"
                size="small"
                show-input
              />
              <div class="config-desc">下降超过 <b>20%</b> 触发告警</div>
            </div>
            <div class="config-item">
              <div class="config-label">湿度异常</div>
              <div class="range-display">
                <el-tag type="info" size="small">低于 20%</el-tag>
                <span>或</span>
                <el-tag type="warning" size="small">高于 90%</el-tag>
              </div>
            </div>
          </div>
        </div>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { ref, computed, reactive, watch, onMounted } from 'vue'
import { useMainStore } from '@/store/main'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { PieChart, BarChart } from 'echarts/charts'
import {
  TitleComponent, TooltipComponent, LegendComponent,
  GridComponent
} from 'echarts/components'
import {
  Warning, DataBoard, PieChart as PieIcon, Bell, Right,
  Cpu, Odometer, Watermelon, WarningFilled
} from '@element-plus/icons-vue'
import dayjs from 'dayjs'
import { alertApi } from '@/api'

use([CanvasRenderer, PieChart, BarChart, TitleComponent, TooltipComponent, LegendComponent, GridComponent])

const store = useMainStore()
const alerts = ref([])
const filterAck = ref(false)
const ackingId = ref(null)

const stats = reactive({
  CRITICAL: 0,
  WARNING: 0,
  INFO: 0,
  acknowledged: 0
})

function severityClass(sev) {
  const map = { CRITICAL: 'critical', WARNING: 'warning', INFO: 'info' }
  return map[sev] || 'default'
}
function severityText(sev) {
  const map = { CRITICAL: '严重', WARNING: '警告', INFO: '提示' }
  return map[sev] || '未知'
}
function alertIcon(type) {
  const map = {
    CRACK_EXCESSIVE: WarningFilled,
    HARDNESS_DROP: Odometer,
    EXTREME_HUMIDITY: Watermelon,
    TEMPERATURE_ALERT: Cpu
  }
  return map[type] || Warning
}
function alertTypeText(type) {
  const map = {
    CRACK_EXCESSIVE: '裂隙超限',
    HARDNESS_DROP: '硬度下降',
    EXTREME_HUMIDITY: '湿度异常',
    TEMPERATURE_ALERT: '温度异常'
  }
  return map[type] || type
}
function formatTime(t) {
  return dayjs(t).format('YYYY-MM-DD HH:mm:ss')
}
function isOver(row) {
  if (row.alertType === 'HARDNESS_DROP') return row.currentValue > row.thresholdValue
  if (row.alertType === 'CRACK_EXCESSIVE') return row.currentValue > row.thresholdValue
  if (row.alertType === 'EXTREME_HUMIDITY') return row.currentValue > 85 || row.currentValue < 20
  return false
}
function getPoint(id) {
  return store.monitoringPoints.find(p => p.id === id)
}

const typeChartOption = computed(() => ({
  backgroundColor: 'transparent',
  tooltip: { trigger: 'item', backgroundColor: 'rgba(15,23,42,0.95)', borderColor: '#334155', textStyle: { color: '#f1f5f9' } },
  legend: { bottom: 0, textStyle: { color: '#94a3b8', fontSize: 11 }, type: 'scroll' },
  series: [{
    type: 'pie',
    radius: ['40%', '65%'],
    center: ['50%', '42%'],
    avoidLabelOverlap: false,
    itemStyle: { borderRadius: 4, borderColor: '#0f172a', borderWidth: 2 },
    label: { show: false },
    labelLine: { show: false },
    data: [
      { value: stats.CRITICAL, name: '严重告警', itemStyle: { color: '#ef4444' } },
      { value: stats.WARNING, name: '警告', itemStyle: { color: '#f59e0b' } },
      { value: stats.INFO, name: '提示', itemStyle: { color: '#3b82f6' } }
    ]
  }]
}))

const caveChartOption = computed(() => {
  const caveAlerts = {}
  alerts.value.forEach(a => {
    const pt = store.monitoringPoints.find(p => p.id === a.pointId)
    if (pt) {
      const cave = store.caves.find(c => c.id === pt.caveId)
      if (cave) {
        caveAlerts[cave.name] = (caveAlerts[cave.name] || 0) + 1
      }
    }
  })
  const entries = Object.entries(caveAlerts).sort((a, b) => b[1] - a[1]).slice(0, 10)
  return {
    backgroundColor: 'transparent',
    tooltip: { trigger: 'axis', backgroundColor: 'rgba(15,23,42,0.95)', borderColor: '#334155', textStyle: { color: '#f1f5f9' } },
    grid: { left: 80, right: 20, top: 10, bottom: 10 },
    xAxis: {
      type: 'value',
      splitLine: { lineStyle: { color: 'rgba(148,163,184,0.08)' } },
      axisLabel: { color: '#64748b', fontSize: 10 }
    },
    yAxis: {
      type: 'category',
      data: entries.map(e => e[0]),
      axisLine: { lineStyle: { color: '#334155' } },
      axisLabel: { color: '#94a3b8', fontSize: 10 }
    },
    series: [{
      type: 'bar',
      data: entries.map(e => e[1]),
      barWidth: 14,
      itemStyle: {
        borderRadius: [0, 4, 4, 0],
        color: {
          type: 'linear', x: 0, y: 0, x2: 1, y2: 0,
          colorStops: [
            { offset: 0, color: '#f59e0b' },
            { offset: 1, color: '#ef4444' }
          ]
        }
      },
      label: {
        show: true, position: 'right',
        color: '#f59e0b', fontSize: 11, fontWeight: 600
      }
    }]
  }
})

async function loadAlerts() {
  try {
    const res = await alertApi.getAll(200, filterAck.value)
    alerts.value = res.data || []
    computeStats()
  } catch (e) { console.error(e) }
}

function computeStats() {
  stats.CRITICAL = alerts.value.filter(a => a.severity === 'CRITICAL').length
  stats.WARNING = alerts.value.filter(a => a.severity === 'WARNING').length
  stats.INFO = alerts.value.filter(a => a.severity === 'INFO').length
  stats.acknowledged = alerts.value.filter(a => a.isAcknowledged).length
}

async function acknowledge(id) {
  ackingId.value = id
  try {
    await alertApi.acknowledge(id, 'admin')
    await loadAlerts()
    await store.loadAlerts()
  } finally { ackingId.value = null }
}

watch(() => store.activeAlerts, () => loadAlerts(), { deep: true })

onMounted(() => loadAlerts())
</script>

<style lang="scss" scoped>
.alert-center {
  .stat-grid {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 10px;
    .stat {
      background: rgba(15,23,42,0.5);
      border: 1px solid var(--border-color);
      border-radius: 8px;
      padding: 12px;
      text-align: center;
      .num {
        font-size: 24px;
        font-weight: 700;
        line-height: 1.1;
      }
      .lbl {
        margin-top: 4px;
        font-size: 11px;
        color: var(--text-muted);
      }
      &.critical .num { color: var(--accent-red); }
      &.warning .num { color: var(--accent-gold); }
      &.info .num { color: var(--accent-blue); }
      &.ack .num { color: var(--accent-green); }
    }
  }
  .threshold-compare {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 12px;
    .current { color: var(--text-primary); font-weight: 600; font-size: 13px;
      &.over { color: var(--accent-red); }
    }
    .threshold { color: var(--text-muted); }
  }
  .threshold-config {
    .config-item {
      padding: 10px 0;
      border-bottom: 1px dashed var(--border-color);
      &:last-child { border-bottom: none; }
    }
    .config-label {
      font-size: 12px;
      color: var(--text-secondary);
      margin-bottom: 6px;
    }
    .config-desc {
      margin-top: 4px;
      font-size: 11px;
      color: var(--text-muted);
      b { color: var(--accent-gold); }
    }
    .range-display {
      display: flex; align-items: center; gap: 8px;
      span { font-size: 11px; color: var(--text-muted); }
    }
  }
}
</style>
