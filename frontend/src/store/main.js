import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { caveApi, pointApi, dashboardApi, alertApi, sensorApi } from '@/api'

export const useMainStore = defineStore('main', () => {
  const caves = ref([])
  const caveStatistics = ref([])
  const selectedCaveId = ref(1)
  const monitoringPoints = ref([])
  const latestSensorData = ref([])
  const recentAlerts = ref([])
  const activeAlerts = ref([])
  const dashboardStats = ref({})
  const currentPoint = ref(null)
  const loading = ref(false)

  const selectedCave = computed(() => caves.value.find(c => c.id === selectedCaveId.value))

  async function loadInitialData() {
    try {
      loading.value = true
      const [cavesRes, statsRes] = await Promise.all([
        caveApi.getAll(),
        caveApi.getStatistics()
      ])
      caves.value = cavesRes.data || []
      caveStatistics.value = statsRes.data || []

      if (caves.value.length > 0 && !selectedCaveId.value) {
        selectedCaveId.value = caves.value[0].id
      }
      await loadMonitoringPoints()
      await loadDashboard()
      await loadAlerts()
    } catch (e) {
      console.error('Load initial data failed:', e)
    } finally {
      loading.value = false
    }
  }

  async function loadMonitoringPoints() {
    if (!selectedCaveId.value) return
    try {
      const res = await pointApi.getByCave(selectedCaveId.value)
      monitoringPoints.value = res.data || []
      await loadLatestSensorData()
    } catch (e) {
      console.error('Load monitoring points failed:', e)
    }
  }

  async function loadLatestSensorData() {
    try {
      const res = await sensorApi.getLatest()
      latestSensorData.value = res.data || []
    } catch (e) {
      console.error('Load sensor data failed:', e)
    }
  }

  async function loadDashboard() {
    try {
      const res = await dashboardApi.getOverview()
      dashboardStats.value = res.data || {}
    } catch (e) {}
  }

  async function loadAlerts(limit = 50) {
    try {
      const res = await alertApi.getAll(limit)
      recentAlerts.value = res.data || []
      activeAlerts.value = recentAlerts.value.filter(a => !a.isAcknowledged)
    } catch (e) {}
  }

  function selectCave(caveId) {
    selectedCaveId.value = caveId
    loadMonitoringPoints()
  }

  function getPointLatestData(pointId) {
    return latestSensorData.value.find(d => d.pointId === pointId)
  }

  function addAlert(alert) {
    recentAlerts.value.unshift(alert)
    if (!alert.isAcknowledged) {
      activeAlerts.value.unshift(alert)
      if (dashboardStats.value.totalActiveAlerts !== undefined) {
        dashboardStats.value.totalActiveAlerts++
      }
    }
    if (recentAlerts.value.length > 200) recentAlerts.value.length = 200
  }

  function updateSensorData(data) {
    const idx = latestSensorData.value.findIndex(d => d.pointId === data.pointId)
    if (idx >= 0) {
      latestSensorData.value[idx] = { ...latestSensorData.value[idx], ...data }
    } else {
      latestSensorData.value.push(data)
    }
  }

  return {
    caves, caveStatistics, selectedCaveId, selectedCave,
    monitoringPoints, latestSensorData, recentAlerts, activeAlerts,
    dashboardStats, currentPoint, loading,
    loadInitialData, loadMonitoringPoints, loadLatestSensorData,
    loadDashboard, loadAlerts, selectCave, getPointLatestData,
    addAlert, updateSensorData
  }
})
