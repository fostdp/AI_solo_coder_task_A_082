<template>
  <el-dialog
    :model-value="visible"
    :title="point ? point.name : '监测点详情'"
    width="960px"
    top="4vh"
    destroy-on-close
    @update:model-value="val => $emit('update:visible', val)"
  >
    <div v-if="point" class="point-detail">
      <el-row :gutter="16">
        <el-col :span="9">
          <div class="info-card">
            <div class="card-title">基础信息</div>
            <div class="info-grid">
              <div class="info-item"><span>监测点编号</span><b>{{ point.code }}</b></div>
              <div class="info-item"><span>所属石窟</span><b>{{ caveName }}</b></div>
              <div class="info-item"><span>岩石类型</span><b>{{ point.rockType }}</b></div>
              <div class="info-item"><span>初始硬度</span><b>{{ point.initialHardness.toFixed(1) }}</b></div>
              <div class="info-item"><span>初始裂隙</span><b>{{ point.initialCrackWidth.toFixed(3) }}mm</b></div>
              <div class="info-item"><span>区域</span><b>{{ point.areaDescription }}</b></div>
            </div>
          </div>

          <div class="info-card" style="margin-top:12px">
            <div class="card-title">
              <el-icon :size="18" style="color:var(--accent-blue)"><Cpu /></el-icon>
              最新传感器数据
              <span class="update-time">{{ formatTime(latest?.time) }}</span>
            </div>
            <div v-if="latest" class="metrics-grid">
              <div class="metric-item" :class="getTempClass(latest.temperature)">
                <el-icon :size="18"><Sunny /></el-icon>
                <div class="metric-value">{{ latest.temperature?.toFixed(1) }}<small>℃</small></div>
                <div class="metric-label">温度</div>
              </div>
              <div class="metric-item" :class="getHumClass(latest.humidity)">
                <el-icon :size="18"><Watermelon /></el-icon>
                <div class="metric-value">{{ latest.humidity?.toFixed(1) }}<small>%</small></div>
                <div class="metric-label">湿度</div>
              </div>
              <div class="metric-item" :class="getHardnessClass()">
                <el-icon :size="18"><CircleCheck /></el-icon>
                <div class="metric-value">{{ latest.surfaceHardness?.toFixed(1) }}<small>/{{ point.initialHardness.toFixed(0) }}</small></div>
                <div class="metric-label">表面硬度</div>
              </div>
              <div class="metric-item" :class="getCrackClass()">
                <el-icon :size="18"><Warning /></el-icon>
                <div class="metric-value">{{ latest.crackWidth?.toFixed(4) }}<small>mm</small></div>
                <div class="metric-label">裂隙宽度</div>
              </div>
            </div>

            <el-progress
              v-if="latest"
              style="margin-top:8px"
              :percentage="hardnessDropPercent"
              :color="hardnessDropPercent > 20 ? '#ef4444' : hardnessDropPercent > 12 ? '#f59e0b' : '#10b981'"
            >
              <template #default="{ percentage }">
                <span class="progress-label">硬度已下降 {{ percentage.toFixed(1) }}%</span>
              </template>
            </el-progress>
          </div>

          <div class="info-card" style="margin-top:12px">
            <div class="card-title">
              <el-icon :size="18" style="color:var(--accent-purple)"><MagicStick /></el-icon>
              风化速率预测
            </div>
            <div class="predict-form">
              <el-row :gutter="8">
                <el-col :span="12">
                  <el-form-item label="温度(℃)" size="small" label-width="80px">
                    <el-input-number v-model="predictTemp" :step="0.5" :precision="1" style="width:100%" />
                  </el-form-item>
                </el-col>
                <el-col :span="12">
                  <el-form-item label="湿度(%)" size="small" label-width="80px">
                    <el-input-number v-model="predictHum" :step="1" :precision="0" style="width:100%" />
                  </el-form-item>
                </el-col>
              </el-row>
              <el-button size="small" type="primary" @click="runPrediction" :loading="predicting">
                运行预测模型
              </el-button>
            </div>
            <div v-if="predictionResult" class="predict-result">
              <div class="predict-main">
                <div class="rate-value" :class="getRiskClass(predictionResult.prediction.predictedRate)">
                  {{ (predictionResult.prediction.predictedRate * 100).toFixed(4) }}<small>%/天</small>
                </div>
                <div class="rate-label">预测风化速率</div>
              </div>
              <div class="predict-meta">
                <el-tag size="small" :type="predictionResult.riskMultiplier > 1.5 ? 'danger' : 'success'">
                  {{ predictionResult.riskMultiplier > 1.5 ? '加速风险' : '稳定' }} ×{{ predictionResult.riskMultiplier.toFixed(2) }}
                </el-tag>
                <el-tag size="small" type="info">置信度 {{ predictionResult.prediction.confidenceLevel.toFixed(1) }}%</el-tag>
              </div>
              <div class="suggestions">
                <div v-for="(s, i) in predictionResult.suggestions" :key="i" class="suggestion-item">
                  <el-icon style="color:var(--accent-gold)"><ChatDotRound /></el-icon>
                  <span>{{ s }}</span>
                </div>
              </div>
            </div>
          </div>
        </el-col>

        <el-col :span="15">
          <div class="info-card">
            <div class="card-title">
              <el-icon :size="18" style="color:var(--accent-gold)"><TrendCharts /></el-icon>
              微环境趋势（近1年）
              <div class="chart-tabs">
                <el-radio-group v-model="chartRange" size="small" @change="loadHistory">
                  <el-radio-button value="7">7天</el-radio-button>
                  <el-radio-button value="30">30天</el-radio-button>
                  <el-radio-button value="90">90天</el-radio-button>
                  <el-radio-button value="365">1年</el-radio-button>
                </el-radio-group>
              </div>
            </div>
            <v-chart :option="historyChartOption" style="height:280px;width:100%" autoresize />
          </div>

          <div class="info-card" style="margin-top:12px">
            <div class="card-title">
              <el-icon :size="18" style="color:var(--accent-red)"><DataLine /></el-icon>
              风化速率变化
            </div>
            <v-chart :option="weatheringRateChartOption" style="height:200px;width:100%" autoresize />
          </div>

          <div class="info-card" style="margin-top:12px">
            <div class="card-title">
              <el-icon :size="18" style="color:var(--accent-green)"><Histogram /></el-icon>
              硬度 & 裂隙详情趋势
            </div>
            <v-chart :option="detailChartOption" style="height:200px;width:100%" autoresize />
          </div>
        </el-col>
      </el-row>
    </div>
  </el-dialog>
</template>

<script setup>import { ref, computed, watch, onMounted } from 'vue';
import { useMainStore } from '@/store/main';
import VChart from 'vue-echarts';
import { use } from 'echarts/core';
import { CanvasRenderer } from 'echarts/renderers';
import { LineChart, BarChart, ScatterChart } from 'echarts/charts';
import {
 TitleComponent, TooltipComponent, LegendComponent,
 GridComponent, DataZoomComponent, VisualMapComponent
} from 'echarts/components';
import {
 Cpu, Sunny, Warning, CircleCheck, MagicStick,
 TrendCharts, DataLine, Histogram, ChatDotRound, Watermelon
} from '@element-plus/icons-vue';
import dayjs from 'dayjs';
import { sensorApi, weatheringApi } from '@/api';
use([
 CanvasRenderer, LineChart, BarChart, ScatterChart,
 TitleComponent, TooltipComponent, LegendComponent,
 GridComponent, DataZoomComponent, VisualMapComponent
]);
const props = defineProps({
 visible: Boolean,
 pointId: [Number, String]
});
const emit = defineEmits(['update:visible']);
const store = useMainStore();
const point = ref(null);
const latest = ref(null);
const history = ref([]);
const weatheringRates = ref([]);
const chartRange = ref('30');
const predictTemp = ref(15);
const predictHum = ref(50);
const predicting = ref(false);
const predictionResult = ref(null);
const caveName = computed(() => store.caves.find(c => c.id === point.value?.caveId)?.name || '');
const hardnessDropPercent = computed(() => {
 if (!latest.value || !point.value) return 0;
 const initial = point.value.initialHardness;
 if (!initial) return 0;
 return Math.max(0, (initial - latest.value.surfaceHardness) / initial * 100);
});
function formatTime(t) {
 if (!t) return '--';
 return dayjs(t).format('YYYY-MM-DD HH:mm');
}
function getTempClass(t) {
 if (t > 35 || t < -5) return 'danger';
 if (t > 30 || t < 0) return 'warning';
 return 'info';
}
function getHumClass(h) {
 if (h > 90 || h < 20) return 'danger';
 if (h > 80 || h < 30) return 'warning';
 return 'success';
}
function getHardnessClass() {
 return hardnessDropPercent.value > 20 ? 'danger' : hardnessDropPercent.value > 12 ? 'warning' : 'success';
}
function getCrackClass() {
 if (!latest.value) return 'info';
 const c = latest.value.crackWidth;
 if (c > 0.5) return 'danger';
 if (c > 0.3) return 'warning';
 return 'info';
}
function getRiskClass(rate) {
 if (rate > 0.005) return 'danger';
 if (rate > 0.002) return 'warning';
 return 'success';
}
const historyChartOption = computed(() => {
 const times = [], temps = [], hums = [];
 const step = Math.max(1, Math.floor(history.value.length / 200));
 for (let i = 0; i < history.value.length; i += step) {
 const d = history.value[i];
 times.push(dayjs(d.time).format('MM-DD HH:mm'));
 temps.push(d.temperature?.toFixed(1));
 hums.push(d.humidity?.toFixed(1));
 }
 return {
 backgroundColor: 'transparent',
 animation: false,
 tooltip: { trigger: 'axis', backgroundColor: 'rgba(15,23,42,0.95)', borderColor: '#334155', textStyle: { color: '#f1f5f9' } },
 legend: { data: ['温度', '湿度'], textStyle: { color: '#94a3b8' }, top: 0 },
 grid: { left: 40, right: 40, top: 30, bottom: 40 },
 dataZoom: [{ type: 'inside' }, { type: 'slider', height: 16, bottom: 4 }],
 xAxis: { type: 'category', data: times, axisLine: { lineStyle: { color: '#334155' } }, axisLabel: { color: '#64748b', fontSize: 10, hideOverlap: true } },
 yAxis: [
 { type: 'value', name: '℃', nameTextStyle: { color: '#64748b' }, splitLine: { lineStyle: { color: 'rgba(148,163,184,0.08)' } }, axisLabel: { color: '#64748b' } },
 { type: 'value', name: '%', nameTextStyle: { color: '#64748b' }, splitLine: { show: false }, axisLabel: { color: '#64748b' } }
 ],
 series: [
 { name: '温度', type: 'line', smooth: true, showSymbol: false, data: temps, itemStyle: { color: '#f59e0b' }, lineStyle: { width: 2 },
 areaStyle: { color: { type: 'linear', x:0,y:0,x2:0,y2:1, colorStops:[{offset:0,color:'rgba(245,158,11,0.3)'},{offset:1,color:'rgba(245,158,11,0)'}] } } },
 { name: '湿度', type: 'line', smooth: true, showSymbol: false, yAxisIndex: 1, data: hums, itemStyle: { color: '#3b82f6' }, lineStyle: { width: 2 } }
 ]
 };
});
const weatheringRateChartOption = computed(() => {
 const labels = weatheringRates.value.map(r => r.periodStart);
 const rates = weatheringRates.value.map(r => (r.overallRate * 100).toFixed(6));
 const levels = weatheringRates.value.map(r => r.riskLevel);
 return {
 backgroundColor: 'transparent',
 tooltip: { trigger: 'axis', backgroundColor: 'rgba(15,23,42,0.95)', borderColor: '#334155', textStyle: { color: '#f1f5f9' } },
 grid: { left: 50, right: 20, top: 20, bottom: 30 },
 xAxis: { type: 'category', data: labels, axisLine: { lineStyle: { color: '#334155' } }, axisLabel: { color: '#64748b', fontSize: 10 } },
 yAxis: { type: 'value', name: '×10⁻² %/天', nameTextStyle: { color: '#64748b' }, splitLine: { lineStyle: { color: 'rgba(148,163,184,0.08)' } }, axisLabel: { color: '#64748b' } },
 series: [{
 type: 'bar', data: rates, barWidth: '60%',
 itemStyle: {
 borderRadius: [4,4,0,0],
 color: params => {
 const level = levels[params.dataIndex];
 return { LOW:'#10b981', MEDIUM:'#3b82f6', HIGH:'#f59e0b', CRITICAL:'#ef4444' }[level] || '#64748b';
 }
 }
 }]
 };
});
const detailChartOption = computed(() => {
 const times = [], hardness = [], cracks = [];
 const step = Math.max(1, Math.floor(history.value.length / 150));
 for (let i = 0; i < history.value.length; i += step) {
 const d = history.value[i];
 times.push(dayjs(d.time).format('MM-DD'));
 hardness.push(d.surfaceHardness?.toFixed(1));
 cracks.push((d.crackWidth * 100)?.toFixed(2));
 }
 return {
 backgroundColor: 'transparent',
 animation: false,
 tooltip: { trigger: 'axis', backgroundColor: 'rgba(15,23,42,0.95)', borderColor: '#334155', textStyle: { color: '#f1f5f9' } },
 legend: { data: ['表面硬度', '裂隙宽度(×100)'], textStyle: { color: '#94a3b8' }, top: 0 },
 grid: { left: 50, right: 50, top: 30, bottom: 30 },
 xAxis: { type: 'category', data: times, axisLine: { lineStyle: { color: '#334155' } }, axisLabel: { color: '#64748b', fontSize: 10, hideOverlap: true } },
 yAxis: [
 { type: 'value', name: '硬度', nameTextStyle: { color: '#64748b' }, splitLine: { lineStyle: { color: 'rgba(148,163,184,0.08)' } }, axisLabel: { color: '#64748b' } },
 { type: 'value', name: '×10⁻²mm', nameTextStyle: { color: '#64748b' }, splitLine: { show: false }, axisLabel: { color: '#64748b' } }
 ],
 series: [
 { name: '表面硬度', type: 'line', smooth: true, showSymbol: false, data: hardness, itemStyle: { color: '#10b981' }, areaStyle: { color: 'rgba(16,185,129,0.1)' } },
 { name: '裂隙宽度(×100)', type: 'line', smooth: true, showSymbol: false, yAxisIndex: 1, data: cracks, itemStyle: { color: '#ef4444' } }
 ]
 };
});
async function loadHistory() {
 if (!props.pointId) return;
 try {
 const res = await sensorApi.getHistory(props.pointId, parseInt(chartRange.value));
 history.value = res.data || [];
 } catch (e) { console.error(e); }
}
async function loadWeatheringRates() {
 if (!props.pointId) return;
 try {
 const res = await weatheringApi.getRates(props.pointId);
 weatheringRates.value = res.data || [];
 } catch (e) { console.error(e); }
}
async function runPrediction() {
 if (!props.pointId) return;
 predicting.value = true;
 try {
 const res = await weatheringApi.predict({
 pointId: props.pointId,
 targetTemperature: predictTemp.value,
 targetHumidity: predictHum.value,
 temperatureRange: 10
 });
 predictionResult.value = res.data;
 } finally { predicting.value = false; }
}
watch(() => props.pointId, async (id) => {
 if (!id) return;
 const pt = store.monitoringPoints.find(p => p.id == id);
 if (pt) {
 point.value = pt;
 const data = store.getPointLatestData(id);
 if (data) latest.value = data;
 else {
 const res = await sensorApi.getLatest(id);
 latest.value = res.data;
 }
 if (latest.value) {
 predictTemp.value = latest.value.temperature || 15;
 predictHum.value = latest.value.humidity || 50;
 }
 await Promise.all([loadHistory(), loadWeatheringRates()]);
 }
}, { immediate: true });
watch(() => props.visible, (val) => {
 if (val && props.pointId) {
 predictionResult.value = null;
 }
});
</script>

<style lang="scss" scoped>
.point-detail {
  .info-card {
    background: rgba(15,23,42,0.4);
    border: 1px solid var(--border-color);
    border-radius: 8px;
    padding: 14px;
  }
  .card-title {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 13px;
    font-weight: 600;
    color: var(--text-primary);
    padding-bottom: 10px;
    margin-bottom: 10px;
    border-bottom: 1px solid var(--border-color);
    .update-time {
      margin-left: auto;
      font-size: 11px;
      color: var(--text-muted);
      font-weight: normal;
    }
    .chart-tabs { margin-left: auto; }
  }
  .info-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 8px;
    .info-item {
      display: flex;
      flex-direction: column;
      gap: 2px;
      span { font-size: 11px; color: var(--text-muted); }
      b { font-size: 13px; color: var(--text-primary); }
    }
  }
  .metrics-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 10px;
    .metric-item {
      background: rgba(59,130,246,0.06);
      border: 1px solid rgba(59,130,246,0.15);
      border-radius: 8px;
      padding: 10px;
      text-align: center;
      color: #3b82f6;
      &.danger { background: var(--danger-bg); border-color: rgba(239,68,68,0.3); color: var(--accent-red); }
      &.warning { background: var(--warning-bg); border-color: rgba(245,158,11,0.3); color: var(--accent-gold); }
      &.success { background: var(--success-bg); border-color: rgba(16,185,129,0.3); color: var(--accent-green); }
      .metric-value {
        font-size: 20px;
        font-weight: 700;
        line-height: 1.2;
        margin-top: 4px;
        small { font-size: 11px; opacity: 0.7; font-weight: normal; }
      }
      .metric-label { font-size: 11px; opacity: 0.8; margin-top: 2px; }
    }
  }
  .progress-label { font-size: 11px; color: var(--text-secondary); }
  .predict-form { margin-bottom: 12px; }
  .predict-result {
    background: rgba(139,92,246,0.06);
    border: 1px solid rgba(139,92,246,0.2);
    border-radius: 8px;
    padding: 12px;
    .predict-main {
      display: flex;
      align-items: baseline;
      gap: 12px;
      margin-bottom: 8px;
      .rate-value {
        font-size: 26px;
        font-weight: 700;
        color: var(--accent-green);
        small { font-size: 12px; opacity: 0.7; font-weight: normal; }
        &.danger { color: var(--accent-red); }
        &.warning { color: var(--accent-gold); }
      }
      .rate-label { font-size: 11px; color: var(--text-muted); }
    }
    .predict-meta { display: flex; gap: 6px; margin-bottom: 10px; }
    .suggestions {
      border-top: 1px dashed var(--border-color);
      padding-top: 8px;
      display: flex; flex-direction: column; gap: 6px;
      .suggestion-item {
        display: flex; gap: 6px; align-items: flex-start;
        font-size: 12px; color: var(--text-secondary); line-height: 1.5;
      }
    }
  }
}
</style>
