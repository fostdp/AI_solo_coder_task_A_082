<template>
  <div class="material-recommendation">
    <el-row :gutter="16">
      <el-col :xs="24" :lg="8">
        <div class="panel-card">
          <div class="section-title">
            <el-icon style="color:var(--accent-blue)"><Filter /></el-icon>
            <span>决策参数</span>
          </div>
          <el-form label-position="top" size="default">
            <el-form-item label="岩石类型 (必填)">
              <el-select v-model="form.rockType" style="width:100%">
                <el-option label="砂砾岩" value="砂砾岩" />
                <el-option label="砂岩" value="砂岩" />
                <el-option label="石灰岩" value="石灰岩" />
                <el-option label="花岗岩" value="花岗岩" />
              </el-select>
            </el-form-item>

            <el-form-item label="所属石窟">
              <el-select v-model="form.caveId" clearable style="width:100%">
                <el-option
                  v-for="c in store.caves"
                  :key="c.id"
                  :label="c.name"
                  :value="c.id"
                />
              </el-select>
            </el-form-item>

            <el-form-item label="保护类型">
              <el-radio-group v-model="form.protectionType" style="width:100%">
                <el-radio-button value="SURFACE">表面封护</el-radio-button>
                <el-radio-button value="WATERPROOF">防水处理</el-radio-button>
                <el-radio-button value="REINFORCEMENT">结构加固</el-radio-button>
              </el-radio-group>
            </el-form-item>

            <el-form-item label="预算等级">
              <el-radio-group v-model="form.budgetLevel" style="width:100%">
                <el-radio-button value="LOW">经济</el-radio-button>
                <el-radio-button value="MEDIUM">中等</el-radio-button>
                <el-radio-button value="HIGH">高端</el-radio-button>
              </el-radio-group>
            </el-form-item>

            <el-form-item label="气候区域">
              <el-select v-model="form.climateZone" style="width:100%">
                <el-option label="干旱区 (西北)" value="ARID" />
                <el-option label="半湿润区 (中原)" value="SEMI_HUMID" />
                <el-option label="湿润区 (南方)" value="HUMID" />
                <el-option label="严寒区 (北方)" value="COLD" />
              </el-select>
            </el-form-item>

            <el-form-item label="优先级指标 (可多选)">
              <el-checkbox-group v-model="form.priorityCriteria">
                <el-checkbox label="耐候性优先" border />
                <el-checkbox label="透气性优先" border />
                <el-checkbox label="可逆性优先" border />
                <el-checkbox label="耐久性优先" border />
                <el-checkbox label="环保优先" border />
                <el-checkbox label="经济性优先" border />
                <el-checkbox label="寿命优先" border />
              </el-checkbox-group>
            </el-form-item>

            <el-button
              type="primary"
              style="width:100%"
              @click="runRecommendation"
              :loading="recommending"
            >
              <el-icon><Histogram /></el-icon>
              运行 TOPSIS 多属性决策
            </el-button>
          </el-form>
        </div>

        <div class="panel-card" style="margin-top:16px">
          <div class="section-title">
            <el-icon style="color:var(--accent-green)"><Scale /></el-icon>
            <span>当前权重配置</span>
          </div>
          <div v-if="currentWeights" class="weight-list">
            <div
              v-for="(val, key) in weightLabels"
              :key="key"
              class="weight-item"
            >
              <span class="label">{{ val }}</span>
              <div class="bar-bg">
                <div
                  class="bar-fill"
                  :style="{ width: (currentWeights[key] || 0) * 500 + '%', background: weightColor(key) }"
                />
              </div>
              <span class="val">{{ ((currentWeights[key] || 0) * 100).toFixed(0) }}%</span>
            </div>
          </div>
        </div>
      </el-col>

      <el-col :xs="24" :lg="16">
        <div class="panel-card">
          <div class="section-title">
            <el-icon style="color:var(--accent-gold)"><Trophy /></el-icon>
            <span>材料推荐结果</span>
            <span v-if="results.length > 0" class="result-count">
              筛选出 <b>{{ results.length }}</b> 种适用材料
            </span>
          </div>
          <div v-if="results.length > 0">
            <div class="top3-cards">
              <div
                v-for="(r, idx) in results.slice(0, 3)"
                :key="r.id"
                class="top-card"
                :class="'rank-' + (idx + 1)"
              >
                <div class="rank-badge">#{{ idx + 1 }}</div>
                <div class="score-ring">
                  <el-progress
                    type="dashboard"
                    :percentage="(r.topsisScore * 100).toFixed(1)"
                    :color="idx === 0 ? '#f59e0b' : idx === 1 ? '#94a3b8' : '#b45309'"
                    :width="100"
                  />
                </div>
                <div class="card-name">{{ r.name }}</div>
                <div class="card-category">
                  <el-tag size="small" effect="plain">{{ r.category }}</el-tag>
                </div>
                <div class="card-stats">
                  <div><span>成本</span><b>¥{{ r.costPerUnit }}/kg</b></div>
                  <div><span>寿命</span><b>{{ r.lifespanYears }}年</b></div>
                </div>
              </div>
            </div>

            <div class="section-title" style="margin-top:16px">
              <el-icon><PieChart /></el-icon>
              <span>材料属性雷达对比 (前5名)</span>
            </div>
            <v-chart :option="radarOption" style="height:360px;width:100%" autoresize />

            <div class="section-title" style="margin-top:8px">
              <el-icon><List /></el-icon>
              <span>完整排名</span>
            </div>
            <el-table :data="results" stripe size="small" style="width:100%">
              <el-table-column prop="rank" label="排名" width="60" align="center">
                <template #default="{ row }">
                  <span v-if="row.rank === 1" class="rank-gold">🥇</span>
                  <span v-else-if="row.rank === 2" class="rank-silver">🥈</span>
                  <span v-else-if="row.rank === 3" class="rank-bronze">🥉</span>
                  <span v-else>#{{ row.rank }}</span>
                </template>
              </el-table-column>
              <el-table-column prop="name" label="材料名称" min-width="150" />
              <el-table-column prop="category" label="类别" width="130">
                <template #default="{ row }">
                  <el-tag size="small" effect="plain">{{ row.category }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="topsisScore" label="TOPSIS评分" width="110" sortable>
                <template #default="{ row }">
                  <b :style="{ color: row.rank <= 3 ? '#f59e0b' : '#94a3b8' }">
                    {{ row.topsisScore.toFixed(4) }}
                  </b>
                </template>
              </el-table-column>
              <el-table-column prop="weatherResistance" label="耐候性" width="80" align="center">
                <template #default="{ row }">
                  <el-rate disabled :model-value="row.weatherResistance / 2" size="small" />
                </template>
              </el-table-column>
              <el-table-column prop="permeability" label="透气性" width="80" align="center">
                <template #default="{ row }">
                  <el-rate disabled :model-value="row.permeability / 2" size="small" />
                </template>
              </el-table-column>
              <el-table-column prop="reversibility" label="可逆性" width="80" align="center">
                <template #default="{ row }">
                  <el-rate disabled :model-value="row.reversibility / 2" size="small" />
                </template>
              </el-table-column>
              <el-table-column prop="costPerUnit" label="单价(¥/kg)" width="90" sortable />
              <el-table-column prop="lifespanYears" label="寿命(年)" width="80" sortable />
            </el-table>

            <div class="suggestion-block" v-if="recommendations.length > 0">
              <div class="block-title">
                <el-icon style="color:var(--accent-gold)"><ChatDotRound /></el-icon>
                推荐决策建议
              </div>
              <ul>
                <li v-for="(r, i) in recommendations" :key="i">{{ r }}</li>
              </ul>
            </div>
          </div>
          <el-empty v-else description="请设置参数后运行TOPSIS决策" :image-size="100" />
        </div>

        <div class="panel-card" style="margin-top:16px">
          <div class="section-title">
            <el-icon style="color:var(--accent-purple)"><Goods /></el-icon>
            <span>保护材料数据库</span>
            <el-tag size="small" type="info" style="margin-left:auto">共 {{ allMaterials.length }} 种</el-tag>
          </div>
          <el-table :data="allMaterials" stripe size="small" max-height="280" style="width:100%">
            <el-table-column prop="name" label="材料名称" min-width="150" />
            <el-table-column prop="category" label="类别" width="110">
              <template #default="{ row }">
                <el-tag size="small" effect="plain">{{ row.category }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="manufacturer" label="生产商" width="130" />
            <el-table-column prop="durability" label="耐久性" width="80" />
            <el-table-column prop="weatherResistance" label="耐候性" width="80" />
            <el-table-column prop="environmentalFriendliness" label="环保性" width="80" />
            <el-table-column prop="costPerUnit" label="单价" width="90" sortable />
            <el-table-column prop="lifespanYears" label="寿命年" width="80" sortable />
          </el-table>
        </div>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { ref, reactive, computed, watch, onMounted } from 'vue'
import { useMainStore } from '@/store/main'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { RadarChart } from 'echarts/charts'
import {
  TitleComponent, TooltipComponent, LegendComponent,
  GridComponent, RadarComponent
} from 'echarts/components'
import {
  Filter, Histogram, Scale, Trophy, PieChart, List,
  ChatDotRound, Goods
} from '@element-plus/icons-vue'
import { materialApi } from '@/api'

use([CanvasRenderer, RadarChart, TitleComponent, TooltipComponent, LegendComponent, GridComponent, RadarComponent])

const store = useMainStore()
const allMaterials = ref([])
const results = ref([])
const recommendations = ref([])
const recommending = ref(false)
const currentWeights = ref(null)

const form = reactive({
  caveId: null,
  pointId: null,
  rockType: '砂砾岩',
  climateZone: 'SEMI_HUMID',
  budgetLevel: 'MEDIUM',
  priorityCriteria: ['耐久性优先', '可逆性优先'],
  protectionType: 'SURFACE'
})

const weightLabels = {
  weatherResistance: '耐候性',
  permeability: '透气性',
  adhesion: '附着力',
  reversibility: '可逆性',
  durability: '耐久性',
  environmentalFriendliness: '环保性',
  costPerUnit: '经济性',
  coverageRate: '覆盖率',
  lifespanYears: '使用寿命'
}

function weightColor(key) {
  const map = {
    weatherResistance: '#ef4444',
    permeability: '#3b82f6',
    adhesion: '#8b5cf6',
    reversibility: '#10b981',
    durability: '#f59e0b',
    environmentalFriendliness: '#14b8a6',
    costPerUnit: '#ec4899',
    coverageRate: '#6366f1',
    lifespanYears: '#f97316'
  }
  return map[key] || '#64748b'
}

const radarOption = computed(() => {
  const top = results.value.slice(0, 5)
  const indicators = [
    { name: '耐候性', max: 10 },
    { name: '透气性', max: 10 },
    { name: '附着力', max: 10 },
    { name: '可逆性', max: 10 },
    { name: '耐久性', max: 10 },
    { name: '环保性', max: 10 },
    { name: '性价比', max: 10 },
    { name: '寿命', max: 10 }
  ]
  const colors = ['#f59e0b', '#94a3b8', '#b45309', '#3b82f6', '#10b981']
  return {
    backgroundColor: 'transparent',
    tooltip: { backgroundColor: 'rgba(15,23,42,0.95)', borderColor: '#334155', textStyle: { color: '#f1f5f9' } },
    legend: { data: top.map(t => t.name), textStyle: { color: '#94a3b8' }, bottom: 0 },
    radar: {
      indicator: indicators,
      center: ['50%', '50%'],
      radius: '60%',
      splitNumber: 4,
      axisName: { color: '#94a3b8', fontSize: 11 },
      splitLine: { lineStyle: { color: 'rgba(148,163,184,0.15)' } },
      splitArea: { areaStyle: { color: ['rgba(245,158,11,0.01)', 'rgba(245,158,11,0.03)'] } },
      axisLine: { lineStyle: { color: 'rgba(148,163,184,0.25)' } }
    },
    series: [{
      type: 'radar',
      data: top.map((t, i) => ({
        name: t.name,
        value: [
          t.weatherResistance, t.permeability, t.adhesion, t.reversibility,
          t.durability, t.environmentalFriendliness,
          Math.min(10, (1000 / Math.max(100, t.costPerUnit)) * t.coverageRate),
          Math.min(10, t.lifespanYears)
        ],
        areaStyle: { color: colors[i] + '22' },
        lineStyle: { color: colors[i], width: 2 },
        itemStyle: { color: colors[i] }
      }))
    }]
  }
})

async function loadMaterials() {
  try {
    const res = await materialApi.getAll()
    allMaterials.value = res.data || []
  } catch (e) { console.error(e) }
}

async function runRecommendation() {
  if (!form.rockType) return
  recommending.value = true
  try {
    const res = await materialApi.recommend({
      caveId: form.caveId,
      pointId: form.pointId,
      rockType: form.rockType,
      climateZone: form.climateZone,
      budgetLevel: form.budgetLevel,
      priorityCriteria: form.priorityCriteria,
      protectionType: form.protectionType
    })
    results.value = res.data?.results || []
    recommendations.value = res.data?.recommendations || []
    currentWeights.value = res.data?.weights || null
  } finally { recommending.value = false }
}

watch(() => store.caves, (caves) => {
  if (caves.length > 0 && !form.caveId) {
    form.caveId = caves[0].id
    form.rockType = caves[0].rockType
  }
}, { immediate: true })

onMounted(() => {
  loadMaterials()
})
</script>

<style lang="scss" scoped>
.material-recommendation {
  .weight-list {
    .weight-item {
      display: flex;
      align-items: center;
      gap: 8px;
      padding: 4px 0;
      font-size: 12px;
      .label { width: 60px; color: var(--text-secondary); flex-shrink: 0; }
      .bar-bg {
        flex: 1;
        height: 8px;
        background: rgba(148,163,184,0.1);
        border-radius: 4px;
        overflow: hidden;
        .bar-fill {
          height: 100%;
          border-radius: 4px;
          transition: width 0.3s ease;
        }
      }
      .val { width: 36px; text-align: right; color: var(--text-muted); font-weight: 600; }
    }
  }
  .result-count {
    margin-left: auto;
    font-size: 12px;
    color: var(--text-muted);
    font-weight: normal;
    b { color: var(--accent-gold); font-size: 14px; }
  }
  .top3-cards {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 16px;
    margin-bottom: 8px;
    @media (max-width: 900px) { grid-template-columns: 1fr; }
    .top-card {
      position: relative;
      background: rgba(15,23,42,0.5);
      border: 1px solid var(--border-color);
      border-radius: 12px;
      padding: 20px 16px;
      text-align: center;
      overflow: hidden;
      .rank-badge {
        position: absolute;
        top: 10px; left: 10px;
        font-size: 13px;
        font-weight: 700;
        padding: 2px 8px;
        border-radius: 10px;
      }
      &.rank-1 {
        border-color: rgba(245,158,11,0.5);
        background: linear-gradient(135deg, rgba(245,158,11,0.1), rgba(15,23,42,0.6));
        box-shadow: 0 0 20px rgba(245,158,11,0.1);
        .rank-badge { background: linear-gradient(135deg, #f59e0b, #d97706); color: #fff; }
      }
      &.rank-2 {
        .rank-badge { background: linear-gradient(135deg, #94a3b8, #64748b); color: #fff; }
      }
      &.rank-3 {
        .rank-badge { background: linear-gradient(135deg, #b45309, #92400e); color: #fff; }
      }
      .score-ring { margin: 8px 0; }
      .card-name {
        font-size: 16px;
        font-weight: 700;
        color: var(--text-primary);
        margin-bottom: 6px;
      }
      .card-category { margin-bottom: 12px; }
      .card-stats {
        display: flex;
        justify-content: space-around;
        padding-top: 10px;
        border-top: 1px dashed var(--border-color);
        div {
          display: flex; flex-direction: column; gap: 2px;
          span { font-size: 10px; color: var(--text-muted); }
          b { font-size: 13px; color: var(--accent-gold); }
        }
      }
    }
  }

  .suggestion-block {
    margin-top: 14px;
    background: rgba(245,158,11,0.05);
    border: 1px solid rgba(245,158,11,0.2);
    border-radius: 8px;
    padding: 12px 16px;
    .block-title {
      display: flex; align-items: center; gap: 6px;
      color: var(--accent-gold); font-weight: 600; margin-bottom: 8px;
    }
    ul { margin: 0; padding-left: 18px; }
    li {
      font-size: 13px; color: var(--text-secondary);
      padding: 3px 0; line-height: 1.7;
    }
  }
}
</style>
