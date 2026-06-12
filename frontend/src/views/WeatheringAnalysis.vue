<template>
  <div class="weathering-analysis">
    <el-row :gutter="16">
      <el-col :xs="24" :lg="8">
        <div class="panel-card">
          <div class="section-title">
            <el-icon style="color:var(--accent-gold)"><Slides /></el-icon>
            <span>监测点选择</span>
          </div>
          <el-select
            v-model="selectedPointId"
            placeholder="请选择监测点"
            filterable
            style="width:100%"
            size="default"
            @change="loadAllData"
          >
            <el-option-group
              v-for="c in store.caves"
              :key="c.id"
              :label="c.name"
            >
              <el-option
                v-for="pt in getCavePoints(c.id)"
                :key="pt.id"
                :label="pt.name"
                :value="pt.id"
              />
            </el-option-group>
          </el-select>

          <div v-if="currentPoint" class="point-info">
            <div class="info-row"><span>岩石类型</span><b>{{ currentPoint.rockType }}</b></div>
            <div class="info-row"><span>初始硬度</span><b>{{ currentPoint.initialHardness.toFixed(1) }}</b></div>
            <div class="info-row"><span>初始裂隙</span><b>{{ currentPoint.initialCrackWidth.toFixed(3) }}mm</b></div>
            <div class="info-row"><span>区域描述</span><b>{{ currentPoint.areaDescription }}</b></div>
          </div>

          <el-divider />

          <div class="section-title">
            <el-icon style="color:var(--accent-purple)"><MagicStick /></el-icon>
            <span>风化速率预测</span>
          </div>
          <div class="predict-form">
            <el-form label-position="top" size="default">
              <el-form-item label="岩石类型">
                <el-select v-model="predictForm.rockType" style="width:100%">
                  <el-option label="砂砾岩" value="砂砾岩" />
                  <el-option label="砂岩" value="砂岩" />
                  <el-option label="石灰岩" value="石灰岩" />
                  <el-option label="花岗岩" value="花岗岩" />
                </el-select>
              </el-form-item>
              <el-row :gutter="8">
                <el-col :span="12">
                  <el-form-item label="温度 (℃)">
                    <el-slider v-model="predictForm.temperature" :min="-20" :max="50" :step="0.5" show-input size="small" />
                  </el-form-item>
                </el-col>
                <el-col :span="12">
                  <el-form-item label="湿度 (%)">
                    <el-slider v-model="predictForm.humidity" :min="0" :max="100" show-input size="small" />
                  </el-form-item>
                </el-col>
              </el-row>
              <el-form-item label="昼夜温差 (℃)">
                <el-slider v-model="predictForm.tempRange" :min="0" :max="30" show-input size="small" />
              </el-form-item>
              <el-button
                type="primary"
                style="width:100%"
                @click="runPrediction"
                :loading="predicting"
              >
                <el-icon><Cpu /></el-icon>
                运行随机森林预测
              </el-button>
            </el-form>
          </div>
        </div>
      </el-col>

      <el-col :xs="24" :lg="16">
        <div class="panel-card">
          <div class="section-title">
            <el-icon style="color:var(--accent-red)"><DataLine /></el-icon>
            <span>风化速率统计（月度）</span>
          </div>
          <v-chart v-if="weatheringRates.length > 0" :option="rateChartOption" style="height:280px" autoresize />
          <el-empty v-else description="请选择监测点查看风化速率" />
        </div>

        <div class="panel-card" style="margin-top:16px">
          <div class="section-title">
            <el-icon style="color:var(--accent-green)"><Odometer /></el-icon>
            <span>预测结果</span>
            <el-tag v-if="prediction" :type="prediction.riskMultiplier > 1.5 ? 'danger' : 'success'" effect="plain" style="margin-left:auto">
              {{ prediction.riskMultiplier > 1.5 ? '风化将加速' : '风化较稳定' }}
            </el-tag>
          </div>
          <div v-if="prediction" class="predict-results">
            <div class="result-metrics">
              <div class="metric-box" :class="getRiskClass(prediction.prediction.predictedRate)">
                <div class="metric-num">
                  {{ (prediction.prediction.predictedRate * 100).toFixed(5) }}
                  <small>%/天</small>
                </div>
                <div class="metric-desc">预测风化速率</div>
              </div>
              <div class="metric-box">
                <div class="metric-num blue">
                  {{ prediction.currentRate ? (prediction.currentRate * 100).toFixed(5) : '---' }}
                  <small>%/天</small>
                </div>
                <div class="metric-desc">当前风化速率</div>
              </div>
              <div class="metric-box" :class="prediction.riskMultiplier > 1.5 ? 'danger' : 'success'">
                <div class="metric-num">
                  ×{{ prediction.riskMultiplier.toFixed(2) }}
                </div>
                <div class="metric-desc">风险倍数</div>
              </div>
              <div class="metric-box">
                <div class="metric-num gold">
                  {{ prediction.prediction.confidenceLevel.toFixed(1) }}<small>%</small>
                </div>
                <div class="metric-desc">模型置信度</div>
              </div>
            </div>
            <div class="prediction-heatmap">
              <div class="heatmap-title">温湿度组合风化速率热力图 ({{ predictForm.rockType }})</div>
              <v-chart :option="heatmapOption" style="height:320px" autoresize />
            </div>
            <div class="suggestion-block">
              <div class="block-title">
                <el-icon><ChatLineRound /></el-icon>
                保护建议
              </div>
              <ul>
                <li v-for="(s, i) in prediction.suggestions" :key="i">{{ s }}</li>
              </ul>
            </div>
          </div>
          <el-empty v-else description="请运行预测模型查看结果" :image-size="80" />
        </div>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, markRaw } from 'vue'
import { useMainStore } from '@/store/main'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart, BarChart, HeatmapChart } from 'echarts/charts'
import {
  TitleComponent, TooltipComponent, LegendComponent,
  GridComponent, DataZoomComponent, VisualMapComponent, GridSimpleComponent
} from 'echarts/components'
import {
  Slides, MagicStick, DataLine, Odometer, Cpu, ChatLineRound
} from '@element-plus/icons-vue'
import { weatheringApi } from '@/api'


use([
  CanvasRenderer, LineChart, BarChart, HeatmapChart,
  TitleComponent, TooltipComponent, LegendComponent,
  GridComponent, DataZoomComponent, VisualMapComponent, GridSimpleComponent
])

const store = useMainStore()
const selectedPointId = ref(null)
const weatheringRates = ref([])
const predicting = ref(false)
const prediction = ref(null)

const predictForm = ref({
  rockType: '砂砾岩',
  temperature: 15,
  humidity: 50,
  tempRange: 10
})

const currentPoint = computed(() =>
  store.monitoringPoints.find(p => p.id === selectedPointId.value)
)

function getCavePoints(caveId) {
  return store.monitoringPoints.filter(p => p.caveId === caveId)
}

function getRiskClass(rate) {
  if (rate > 0.005) return 'danger'
  if (rate > 0.002) return 'warning'
  return 'success'
}

const rateChartOption = computed(() => {
  const labels = weatheringRates.value.map(r => r.periodStart)
  const hardnessLoss = weatheringRates.value.map(r => (r.hardnessLossRate * 1000).toFixed(4))
  const crackGrowth = weatheringRates.value.map(r => (r.crackGrowthRate * 10000).toFixed(4))
  const overall = weatheringRates.value.map(r => (r.overallRate * 100).toFixed(5))
  const riskColors = weatheringRates.value.map(r => ({
    LOW: '#10b981', MEDIUM: '#3b82f6', HIGH: '#f59e0b', CRITICAL: '#ef4444'
  }[r.riskLevel] || '#64748b'))

  return {
    backgroundColor: 'transparent',
    tooltip: {
      trigger: 'axis',
      backgroundColor: 'rgba(15,23,42,0.95)',
      borderColor: '#334155',
      textStyle: { color: '#f1f5f9' }
    },
    legend: {
      data: ['硬度损失率(‰/天)', '裂隙增长率(×10⁻⁴mm/天)', '综合速率(×10⁻²%/天)'],
      textStyle: { color: '#94a3b8' },
      top: 0
    },
    grid: { left: 60, right: 60, top: 40, bottom: 50 },
    dataZoom: [{ type: 'inside' }, { type: 'slider', bottom: 8, height: 16 }],
    xAxis: {
      type: 'category', data: labels,
      axisLine: { lineStyle: { color: '#334155' } },
      axisLabel: { color: '#64748b', fontSize: 10, rotate: 30 }
    },
    yAxis: [
      { type: 'value', name: '损失/增长', nameTextStyle: { color: '#64748b' }, splitLine: { lineStyle: { color: 'rgba(148,163,184,0.08)' } }, axisLabel: { color: '#64748b' } },
      { type: 'value', name: '综合速率', nameTextStyle: { color: '#64748b' }, splitLine: { show: false }, axisLabel: { color: '#64748b' } }
    ],
    series: [
      {
        name: '硬度损失率(‰/天)', type: 'bar', data: hardnessLoss, barWidth: 16,
        itemStyle: { borderRadius: [4,4,0,0], color: 'rgba(59,130,246,0.7)' }
      },
      {
        name: '裂隙增长率(×10⁻⁴mm/天)', type: 'bar', data: crackGrowth, barWidth: 16,
        itemStyle: { borderRadius: [4,4,0,0], color: 'rgba(139,92,246,0.7)' }
      },
      {
        name: '综合速率(×10⁻²%/天)', type: 'line', smooth: true, yAxisIndex: 1, data: overall,
        itemStyle: { color: '#ef4444' }, lineStyle: { width: 3 },
        symbol: 'circle', symbolSize: 8,
        markArea: { silent: true, itemStyle: { opacity: 0.15 },
          data: [
            [{ yAxis: 0, itemStyle: { color: '#10b981' } }, { yAxis: 0.2 }],
            [{ yAxis: 0.2, itemStyle: { color: '#f59e0b' } }, { yAxis: 0.5 }],
            [{ yAxis: 0.5, itemStyle: { color: '#ef4444' } }, { yAxis: 999 }]
          ]
        }
      }
    ]
  }
})

const heatmapOption = computed(() => {
  const temps = []
  for (let t = -10; t <= 40; t += 5) temps.push(t + '℃')
  const hums = []
  for (let h = 20; h <= 95; h += 5) hums.push(h + '%')
  const data = []

  const rockFactors = { '砂砾岩': 1.8, '砂岩': 1.4, '石灰岩': 1.0, '花岗岩': 0.6 }
  const factor = rockFactors[predictForm.value.rockType] || 1.0

  for (let i = 0; i < temps.length; i++) {
    for (let j = 0; j < hums.length; j++) {
      const t = parseInt(temps[i])
      const h = parseInt(hums[j])
      const tr = predictForm.value.tempRange
      const tempStress = Math.pow(Math.abs(t - 15), 1.5) * 0.002
      const humStress = Math.pow(h - 60, 2) * 0.0005 + (h < 30 ? (30 - h) * 0.01 : 0)
      const rangeStress = tr * 0.008
      const interaction = (t > 30 && h > 75 ? 0.05 : 0) + (t < -5 && h > 60 ? 0.08 : 0)
      let rate = (tempStress + humStress + rangeStress + interaction) * factor * 0.01
      rate = Math.max(0.0001, rate) * 10000
      data.push([j, i, rate.toFixed(3)])
    }
  }

  return {
    backgroundColor: 'transparent',
    tooltip: {
      position: 'top',
      backgroundColor: 'rgba(15,23,42,0.95)',
      borderColor: '#334155',
      textStyle: { color: '#f1f5f9' },
      formatter: p => `温度: ${temps[p.value[1]]}<br/>湿度: ${hums[p.value[0]]}<br/><b>风化速率: ${Number(p.value[2]).toFixed(2)} ×10⁻⁴%/天</b>`
    },
    grid: { left: 60, right: 40, top: 10, bottom: 80 },
    xAxis: { type: 'category', data: hums, splitArea: { show: true }, axisLabel: { color: '#64748b', fontSize: 10 }, name: '湿度', nameTextStyle: { color: '#94a3b8' } },
    yAxis: { type: 'category', data: temps, splitArea: { show: true }, axisLabel: { color: '#64748b', fontSize: 10 }, name: '温度', nameTextStyle: { color: '#94a3b8' } },
    visualMap: {
      min: 0, max: 80, calculable: true, orient: 'horizontal',
      left: 'center', bottom: 10,
      textStyle: { color: '#94a3b8', fontSize: 10 },
      inRange: { color: ['#10b981', '#3b82f6', '#f59e0b', '#ef4444', '#dc2626'] },
      text: ['高', '低']
    },
    series: [{
      name: '风化速率',
      type: 'heatmap',
      data: data,
      label: { show: false },
      emphasis: { itemStyle: { shadowBlur: 10, shadowColor: 'rgba(245,158,11,0.5)', borderColor: '#f59e0b', borderWidth: 2 } }
    }]
  }
})

async function loadAllData() {
  if (!selectedPointId.value) return
  try {
    const res = await weatheringApi.getRates(selectedPointId.value)
    weatheringRates.value = res.data || []
  } catch (e) { console.error(e) }
}

async function runPrediction() {
  if (!selectedPointId.value) return
  predicting.value = true
  try {
    const res = await weatheringApi.predict({
      pointId: selectedPointId.value,
      targetTemperature: predictForm.value.temperature,
      targetHumidity: predictForm.value.humidity,
      temperatureRange: predictForm.value.tempRange,
      rockType: predictForm.value.rockType
    })
    prediction.value = res.data
  } finally { predicting.value = false }
}

watch(() => currentPoint.value, (pt) => {
  if (pt) predictForm.value.rockType = pt.rockType
})

onMounted(() => {
  if (store.monitoringPoints.length > 0) {
    selectedPointId.value = store.monitoringPoints[0].id
    predictForm.value.rockType = store.monitoringPoints[0].rockType
    loadAllData()
  }
})
</script>

<style lang="scss" scoped>
.weathering-analysis {
  .point-info {
    margin-top: 14px;
    .info-row {
      display: flex; justify-content: space-between;
      padding: 5px 0; font-size: 13px;
      border-bottom: 1px dashed var(--border-color);
      &:last-child { border-bottom: none; }
      span { color: var(--text-muted); }
      b { color: var(--text-primary); font-weight: 600; }
    }
  }
  .predict-form {
    margin-top: 4px;
    :deep(.el-slider__runway) { background: rgba(148,163,184,0.2); }
  }

  .predict-results {
    .result-metrics {
      display: grid;
      grid-template-columns: repeat(4, 1fr);
      gap: 12px;
      margin-bottom: 16px;
      @media (max-width: 900px) { grid-template-columns: repeat(2, 1fr); }
      .metric-box {
        background: rgba(15,23,42,0.4);
        border: 1px solid var(--border-color);
        border-radius: 8px;
        padding: 14px;
        text-align: center;
        .metric-num {
          font-size: 22px; font-weight: 700;
          color: var(--accent-green);
          small { font-size: 11px; opacity: 0.7; font-weight: normal; }
          &.blue { color: var(--accent-blue); }
          &.gold { color: var(--accent-gold); }
        }
        .metric-desc { font-size: 11px; color: var(--text-muted); margin-top: 4px; }
        &.danger .metric-num { color: var(--accent-red); }
        &.warning .metric-num { color: var(--accent-gold); }
      }
    }
    .prediction-heatmap { margin-bottom: 16px;
      .heatmap-title {
        font-size: 13px; font-weight: 600; color: var(--text-primary);
        margin-bottom: 8px; padding-bottom: 8px;
        border-bottom: 1px solid var(--border-color);
      }
    }
    .suggestion-block {
      background: rgba(245,158,11,0.05);
      border: 1px solid rgba(245,158,11,0.2);
      border-radius: 8px;
      padding: 12px 16px;
      .block-title {
        display: flex; align-items: center; gap: 6px;
        color: var(--accent-gold); font-weight: 600; margin-bottom: 8px;
      }
      ul { margin: 0; padding-left: 16px; }
      li {
        font-size: 13px; color: var(--text-secondary);
        padding: 3px 0; line-height: 1.6;
      }
    }
  }
}
</style>
