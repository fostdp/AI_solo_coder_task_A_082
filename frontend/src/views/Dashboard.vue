<template>
  <div class="dashboard-view">
    <div class="stats-row">
      <div class="stat-card" v-for="(stat, idx) in statCards" :key="idx">
        <div class="stat-header">
          <el-icon :size="22" :style="{ color: stat.color }"><component :is="stat.icon" /></el-icon>
        </div>
        <div class="stat-value" :class="stat.colorClass">{{ stat.value }}</div>
        <div class="stat-label">{{ stat.label }}</div>
        <div class="stat-sub">{{ stat.sub }}</div>
      </div>
    </div>

    <el-row :gutter="16" class="content-row">
      <el-col :xs="24" :lg="14">
        <div class="panel-card">
          <div class="section-title">
            <el-icon class="title-icon"><Location /></el-icon>
            <span>石窟群监测概览</span>
            <el-select
              v-model="selectedCaveId"
              size="small"
              style="margin-left: auto; width: 160px"
              @change="handleCaveChange"
            >
              <el-option
                v-for="c in store.caves"
                :key="c.id"
                :label="c.name"
                :value="c.id"
              />
            </el-select>
          </div>
          <div class="cave-map-container">
            <HeatmapMap
              :cave="currentCave"
              :points="currentPoints"
              :sensorData="store.latestSensorData"
              @point-click="handlePointClick"
            />
          </div>
        </div>

        <div class="panel-card" style="margin-top: 16px;">
          <div class="section-title">
            <el-icon class="title-icon"><TrendCharts /></el-icon>
            <span>温湿度与风化趋势（近30天）</span>
          </div>
          <v-chart v-if="currentPointId" :option="trendChartOption" class="trend-chart" autoresize />
          <el-empty v-else description="请选择监测点查看趋势" />
        </div>
      </el-col>

      <el-col :xs="24" :lg="10">
        <div class="panel-card">
          <div class="section-title">
            <el-icon class="title-icon"><PieChart /></el-icon>
            <span>石窟群分布</span>
          </div>
          <v-chart :option="distributionOption" class="distribution-chart" autoresize />
        </div>

        <div class="panel-card" style="margin-top: 16px;">
          <div class="section-title">
            <el-icon class="title-icon"><Warning /></el-icon>
            <span>最近告警</span>
            <el-tag size="small" type="danger" effect="plain" style="margin-left:auto">
              {{ store.activeAlerts.length }} 未处理
            </el-tag>
          </div>
          <div class="recent-alerts">
            <div v-if="store.recentAlerts.length === 0" class="empty">暂无告警</div>
            <div
              v-for="alert in store.recentAlerts.slice(0, 8)"
              :key="alert.id"
              class="alert-row"
              :class="{ ack: alert.isAcknowledged }"
            >
              <span class="badge-tag" :class="severityClass(alert.severity)">{{ alert.severity }}</span>
              <span class="alert-text">{{ alert.title }}</span>
              <span class="alert-time">{{ formatTime(alert.createdAt) }}</span>
            </div>
          </div>
        </div>

        <div class="panel-card" style="margin-top: 16px;">
          <div class="section-title">
            <el-icon class="title-icon"><Cpu /></el-icon>
            <span>监测点实时状态</span>
          </div>
          <div class="point-status-list">
            <div
              v-for="pt in currentPoints.slice(0, 8)"
              :key="pt.id"
              class="point-row"
              @click="handlePointClick(pt.id)"
            >
              <div class="point-name">
                <span class="status-dot" :class="getPointStatusClass(pt.id)"></span>
                {{ pt.name }}
              </div>
              <div class="point-metrics">
                <span>硬度: <b>{{ getHardness(pt.id) }}</b></span>
                <span>裂隙: <b>{{ getCrack(pt.id) }}mm</b></span>
              </div>
            </div>
          </div>
        </div>
      </el-col>
    </el-row>

    <PointDetailDialog
      v-model:visible="pointDialogVisible"
      :point-id="currentPointId"
    />
  </div>
</template>

<script setup>import { ref, computed, watch, onMounted, markRaw } from 'vue';
import { useMainStore } from '@/store/main';
import VChart from 'vue-echarts';
import { use } from 'echarts/core';
import { CanvasRenderer } from 'echarts/renderers';
import { LineChart, PieChart, GaugeChart } from 'echarts/charts';
import {
 TitleComponent, TooltipComponent, LegendComponent,
 GridComponent, DataZoomComponent
} from 'echarts/components';
import {
 DataAnalysis, Cpu, Location, Warning, PieChart as PieIcon,
 TrendCharts, Histogram, Bell
} from '@element-plus/icons-vue';
import dayjs from 'dayjs';
import HeatmapMap from '@/components/HeatmapMap.vue';
import PointDetailDialog from '@/components/PointDetailDialog.vue';
import { sensorApi } from '@/api';
use([
 CanvasRenderer, LineChart, PieChart, GaugeChart,
 TitleComponent, TooltipComponent, LegendComponent,
 GridComponent, DataZoomComponent
]);
const store = useMainStore();
const selectedCaveId = ref(store.selectedCaveId || 1);
const currentPointId = ref(null);
const pointDialogVisible = ref(false);
const trendData = ref({ times: [], temps: [], hums: [], hardness: [], cracks: [] });
const statCards = computed(() => [
 {
 value: store.dashboardStats.totalCaves || store.caves.length,
 label: '石窟群总数',
 sub: '分布于6个省份',
 icon: markRaw(Location),
 color: '#f59e0b', colorClass: 'gold'
 },
 {
 value: store.dashboardStats.totalPoints || store.monitoringPoints.length,
 label: '监测点数量',
 sub: '每小时上报数据',
 icon: markRaw(Cpu),
 color: '#3b82f6', colorClass: 'blue'
 },
 {
 value: store.dashboardStats.totalActiveAlerts ?? store.activeAlerts.length,
 label: '活跃告警数',
 sub: '裂隙超限/硬度下降',
 icon: markRaw(Warning),
 color: '#ef4444', colorClass: 'red'
 },
 {
 value: 'RF-v1.0',
 label: '预测模型版本',
 sub: '随机森林+TOPSIS',
 icon: markRaw(TrendCharts),
 color: '#10b981', colorClass: 'green'
 }
]);
const currentCave = computed(() => store.caves.find(c => c.id === selectedCaveId.value));
const currentPoints = computed(() => store.monitoringPoints);
const distributionOption = computed(() => {
 const provinceMap = {};
 store.caveStatistics.forEach(s => {
 provinceMap[s.province] = (provinceMap[s.province] || 0) + s.totalPoints;
 });
 const data = Object.entries(provinceMap).map(([name, value]) => ({ name, value }));
 return {
 backgroundColor: 'transparent',
 tooltip: { trigger: 'item', formatter: '{b}: {c}个监测点 ({d}%)' },
 legend: {
 bottom: 0, textStyle: { color: '#94a3b8', fontSize: 11 },
 type: 'scroll'
 },
 series: [{
 type: 'pie',
 radius: ['35%', '65%'],
 center: ['50%', '42%'],
 avoidLabelOverlap: true,
 itemStyle: { borderColor: '#0f172a', borderWidth: 2, borderRadius: 4 },
 label: { show: false },
 emphasis: {
 label: { show: true, fontSize: 13, fontWeight: 600, color: '#f1f5f9' }
 },
 labelLine: { show: false },
 data
 }],
 color: ['#f59e0b', '#3b82f6', '#10b981', '#8b5cf6', '#ef4444', '#ec4899']
 };
});
const trendChartOption = computed(() => ({
 backgroundColor: 'transparent',
 animation: false,
 tooltip: {
 trigger: 'axis',
 backgroundColor: 'rgba(15,23,42,0.95)',
 borderColor: '#334155',
 textStyle: { color: '#f1f5f9' }
 },
 legend: {
 data: ['温度(℃)', '湿度(%)', '表面硬度', '裂隙宽度(×10)'],
 textStyle: { color: '#94a3b8', fontSize: 11 },
 top: 0
 },
 grid: { left: 40, right: 40, top: 40, bottom: 50 },
 dataZoom: [
 { type: 'inside', start: 70, end: 100 },
 { type: 'slider', height: 18, bottom: 8, start: 70, end: 100,
 borderColor: 'transparent', textStyle: { color: '#64748b', fontSize: 10 } }
 ],
 xAxis: {
 type: 'category',
 data: trendData.value.times,
 axisLine: { lineStyle: { color: '#334155' } },
 axisLabel: { color: '#64748b', fontSize: 10, hideOverlap: true }
 },
 yAxis: [
 {
 type: 'value',
 splitLine: { lineStyle: { color: 'rgba(148,163,184,0.08)' } },
 axisLabel: { color: '#64748b', fontSize: 10 }
 },
 {
 type: 'value',
 splitLine: { show: false },
 axisLabel: { color: '#64748b', fontSize: 10 }
 }
 ],
 series: [
 {
 name: '温度(℃)', type: 'line', smooth: true, showSymbol: false,
 data: trendData.value.temps,
 itemStyle: { color: '#f59e0b' }, lineStyle: { width: 2 },
 areaStyle: { color: {
 type: 'linear', x:0,y:0,x2:0,y2:1,
 colorStops: [{offset:0,color:'rgba(245,158,11,0.25)'},{offset:1,color:'rgba(245,158,11,0)'}]
 } }
 },
 {
 name: '湿度(%)', type: 'line', smooth: true, showSymbol: false,
 data: trendData.value.hums,
 itemStyle: { color: '#3b82f6' }, lineStyle: { width: 2 },
 },
 {
 name: '表面硬度', type: 'line', smooth: true, showSymbol: false, yAxisIndex: 1,
 data: trendData.value.hardness,
 itemStyle: { color: '#10b981' }, lineStyle: { width: 2 }
 },
 {
 name: '裂隙宽度(×10)', type: 'line', smooth: true, showSymbol: false, yAxisIndex: 1,
 data: trendData.value.cracks.map(v => (v * 10).toFixed(3)),
 itemStyle: { color: '#ef4444' }, lineStyle: { width: 2 }
 }
 ]
}));
async function loadTrendData(pointId) {
 if (!pointId) return;
 try {
 const res = await sensorApi.getHistory(pointId, 30);
 const data = res.data || [];
 const times = [], temps = [], hums = [], hardness = [], cracks = [];
 const step = Math.max(1, Math.floor(data.length / 200));
 for (let i = 0; i < data.length; i += step) {
 const d = data[i];
 times.push(dayjs(d.time).format('MM-DD HH:mm'));
 temps.push(d.temperature?.toFixed(1));
 hums.push(d.humidity?.toFixed(1));
 hardness.push(d.surfaceHardness?.toFixed(1));
 cracks.push(d.crackWidth || 0);
 }
 trendData.value = { times, temps, hums, hardness, cracks };
 }
 catch (e) {
 console.error(e);
 }
}
function handleCaveChange() {
 store.selectCave(selectedCaveId.value);
 if (currentPoints.value.length > 0) {
 handlePointClick(currentPoints.value[0].id);
 }
}
function handlePointClick(pointId) {
 currentPointId.value = pointId;
 loadTrendData(pointId);
}
function severityClass(sev) {
 const map = { CRITICAL: 'critical', WARNING: 'warning', INFO: 'info' };
 return map[sev] || 'default';
}
function formatTime(t) { return dayjs(t).format('MM-DD HH:mm'); }
function getPointStatusClass(pointId) {
 const d = store.getPointLatestData(pointId);
 if (!d)
 return 'unknown';
 const pt = store.monitoringPoints.find(p => p.id === pointId);
 const initial = pt?.initialHardness || 50;
 if (d.crackWidth > 0.5)
 return 'danger';
 if (initial && (initial - d.surfaceHardness) / initial * 100 > 20)
 return 'danger';
 if (d.crackWidth > 0.3)
 return 'warning';
 return 'normal';
}
function getHardness(pointId) {
 const d = store.getPointLatestData(pointId);
 return d ? d.surfaceHardness?.toFixed(1) : '--';
}
function getCrack(pointId) {
 const d = store.getPointLatestData(pointId);
 return d ? d.crackWidth?.toFixed(3) : '--';
}
watch(() => store.selectedCaveId, (val) => { selectedCaveId.value = val; });
onMounted(() => {
 if (currentPoints.value.length > 0) {
 handlePointClick(currentPoints.value[0].id);
 }
});
</script>

<style lang="scss" scoped>
.dashboard-view {
  .stats-row {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 16px;
    margin-bottom: 16px;
    @media (max-width: 1200px) { grid-template-columns: repeat(2, 1fr); }
    @media (max-width: 640px) { grid-template-columns: 1fr; }
  }
}

.stat-card {
  .stat-header { margin-bottom: 8px; }
}

.cave-map-container {
  width: 100%;
  height: 420px;
  border-radius: 8px;
  overflow: hidden;
}

.trend-chart {
  height: 320px;
  width: 100%;
}

.distribution-chart {
  height: 260px;
  width: 100%;
}

.recent-alerts {
  .empty { text-align: center; color: var(--text-muted); padding: 24px; font-size: 13px; }
  .alert-row {
    display: flex; align-items: center; gap: 10px;
    padding: 8px 6px; border-radius: 6px;
    border-bottom: 1px solid var(--border-color);
    &:last-child { border-bottom: none; }
    &.ack { opacity: 0.5; }
    .alert-text {
      flex: 1; font-size: 13px; color: var(--text-primary);
      overflow: hidden; white-space: nowrap; text-overflow: ellipsis;
    }
    .alert-time { font-size: 11px; color: var(--text-muted); flex-shrink: 0; }
  }
}

.point-status-list {
  max-height: 280px;
  overflow-y: auto;
  .point-row {
    padding: 10px 8px;
    border-radius: 6px;
    border-bottom: 1px solid var(--border-color);
    cursor: pointer;
    &:hover { background: rgba(59,130,246,0.06); }
    &:last-child { border-bottom: none; }
    .point-name {
      display: flex; align-items: center; gap: 8px;
      font-size: 13px; color: var(--text-primary); font-weight: 500;
      .status-dot {
        width: 8px; height: 8px; border-radius: 50%;
        &.normal { background: var(--accent-green); box-shadow: 0 0 6px var(--accent-green); }
        &.warning { background: var(--accent-gold); box-shadow: 0 0 6px var(--accent-gold); }
        &.danger { background: var(--accent-red); box-shadow: 0 0 6px var(--accent-red); animation: blink 1s infinite; }
        &.unknown { background: var(--text-muted); }
      }
    }
    .point-metrics {
      margin-top: 6px;
      display: flex; gap: 16px; font-size: 11px; color: var(--text-secondary);
      b { color: var(--text-primary); font-weight: 600; }
    }
  }
  @keyframes blink {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.4; }
  }
}
</style>
