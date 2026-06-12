<template>
  <div class="grotto-3d-view">
    <div class="left-toolbar panel-card">
      <div class="section-title" style="margin-bottom:8px">
        <el-icon><Mountain /></el-icon>
        <span>石窟选择</span>
      </div>
      <div class="cave-list">
        <div
          v-for="c in store.caves"
          :key="c.id"
          class="cave-item"
          :class="{ active: c.id === store.selectedCaveId }"
          @click="store.selectCave(c.id)"
        >
          <div class="cave-name">{{ c.name }}</div>
          <div class="cave-meta">
            <el-tag size="small" effect="plain">{{ c.rockType }}</el-tag>
            <span class="province">{{ c.province }}</span>
          </div>
        </div>
      </div>

      <el-divider style="margin: 12px 0" />

      <div class="section-title" style="margin-bottom:8px">
        <el-icon><Cpu /></el-icon>
        <span>监测点列表</span>
        <span class="count">{{ store.monitoringPoints.length }}</span>
      </div>
      <div class="point-search">
        <el-input
          v-model="pointKeyword"
          placeholder="搜索监测点..."
          size="small"
          clearable
        >
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
      </div>
      <div class="point-list">
        <div
          v-for="pt in filteredPoints"
          :key="pt.id"
          class="point-item"
          :class="getPointStatusClass(pt.id)"
          @click="openPointDetail(pt.id)"
        >
          <div class="point-name">
            <span class="status-dot"></span>
            <span>{{ pt.code }}</span>
          </div>
          <div class="point-data">
            <span>硬: {{ getHardness(pt.id) }}</span>
            <span>裂: {{ getCrack(pt.id) }}</span>
          </div>
        </div>
      </div>
    </div>

    <div class="main-view panel-card">
      <GrottoThreeScene
        :cave="store.selectedCave"
        :points="store.monitoringPoints"
        :sensor-data="store.latestSensorData"
        @point-click="openPointDetail"
      />
    </div>

    <PointDetailDialog
      v-model:visible="pointDialogVisible"
      :point-id="currentPointId"
    />
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useMainStore } from '@/store/main'
import { Mountain, Cpu, Search } from '@element-plus/icons-vue'
import GrottoThreeScene from '@/components/GrottoThreeScene.vue'
import PointDetailDialog from '@/components/PointDetailDialog.vue'

const store = useMainStore()
const pointKeyword = ref('')
const pointDialogVisible = ref(false)
const currentPointId = ref(null)

const filteredPoints = computed(() => {
  const kw = pointKeyword.value.trim().toLowerCase()
  if (!kw) return store.monitoringPoints
  return store.monitoringPoints.filter(p =>
    p.name.toLowerCase().includes(kw) ||
    p.code.toLowerCase().includes(kw)
  )
})

function getPointStatusClass(pointId) {
  const d = store.getPointLatestData(pointId)
  const pt = store.monitoringPoints.find(p => p.id === pointId)
  if (!d || !pt) return 'unknown'
  const drop = pt.initialHardness > 0 ? (pt.initialHardness - d.surfaceHardness) / pt.initialHardness * 100 : 0
  if (d.crackWidth > 0.5 || drop > 20) return 'danger'
  if (d.crackWidth > 0.3 || drop > 12) return 'warning'
  return 'normal'
}

function getHardness(pointId) {
  const d = store.getPointLatestData(pointId)
  return d ? d.surfaceHardness?.toFixed(0) : '-'
}
function getCrack(pointId) {
  const d = store.getPointLatestData(pointId)
  return d ? (d.crackWidth?.toFixed(2) + 'mm') : '-'
}

function openPointDetail(pointId) {
  currentPointId.value = pointId
  pointDialogVisible.value = true
}
</script>

<style lang="scss" scoped>
.grotto-3d-view {
  display: grid;
  grid-template-columns: 260px 1fr;
  gap: 16px;
  height: 100%;
  min-height: 720px;
}

.left-toolbar {
  display: flex;
  flex-direction: column;
  overflow: hidden;

  .cave-list {
    max-height: 240px;
    overflow-y: auto;
    .cave-item {
      padding: 8px 10px;
      border-radius: 6px;
      cursor: pointer;
      margin-bottom: 4px;
      border: 1px solid transparent;
      &:hover { background: rgba(59,130,246,0.06); }
      &.active {
        background: linear-gradient(90deg, rgba(245,158,11,0.15), transparent);
        border-color: rgba(245,158,11,0.3);
      }
      .cave-name { font-size: 13px; font-weight: 500; color: var(--text-primary); }
      .cave-meta {
        margin-top: 4px;
        display: flex; align-items: center; gap: 6px;
        font-size: 11px; color: var(--text-muted);
      }
    }
  }

  .point-search { margin-bottom: 8px; }
  .count {
    margin-left: auto;
    font-size: 11px;
    background: rgba(59,130,246,0.15);
    color: var(--accent-blue);
    padding: 2px 8px;
    border-radius: 10px;
    font-weight: normal;
  }
  .point-list {
    flex: 1;
    overflow-y: auto;
    .point-item {
      padding: 8px;
      border-radius: 6px;
      cursor: pointer;
      margin-bottom: 3px;
      border: 1px solid transparent;
      &.normal:hover { background: rgba(16,185,129,0.08); border-color: rgba(16,185,129,0.2); }
      &.warning { background: rgba(245,158,11,0.06); border-color: rgba(245,158,11,0.15); }
      &.danger { background: rgba(239,68,68,0.08); border-color: rgba(239,68,68,0.2); }
      &.unknown { opacity: 0.5; }
      .point-name {
        display: flex; align-items: center; gap: 6px;
        font-size: 12px; font-weight: 600; color: var(--text-primary);
        .status-dot {
          width: 7px; height: 7px; border-radius: 50%;
        }
        .normal & .status-dot { background: var(--accent-green); box-shadow: 0 0 5px var(--accent-green); }
        .warning & .status-dot { background: var(--accent-gold); box-shadow: 0 0 5px var(--accent-gold); }
        .danger & .status-dot { background: var(--accent-red); box-shadow: 0 0 5px var(--accent-red); animation: pulse 1s infinite; }
        .unknown & .status-dot { background: var(--text-muted); }
      }
      .point-data {
        margin-top: 4px;
        display: flex; gap: 12px;
        font-size: 11px; color: var(--text-secondary);
      }
    }
  }
}

.main-view {
  padding: 0;
  overflow: hidden;
  position: relative;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}

@media (max-width: 1200px) {
  .grotto-3d-view { grid-template-columns: 1fr; }
  .left-toolbar { max-height: 280px; }
}
</style>
