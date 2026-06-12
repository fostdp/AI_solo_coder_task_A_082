import axios from 'axios'
import { ElMessage } from 'element-plus'

const baseURL = import.meta.env.VITE_API_BASE || '/api/v1'

const request = axios.create({
  baseURL,
  timeout: 30000
})

request.interceptors.request.use(config => {
  config.headers['Content-Type'] = 'application/json'
  return config
})

request.interceptors.response.use(
  response => {
    const res = response.data
    if (res.code !== 0 && res.code !== undefined) {
      ElMessage.error(res.message || '请求失败')
    }
    return res
  },
  error => {
    console.error('API Error:', error)
    ElMessage.error(error.message || '网络错误')
    return Promise.reject(error)
  }
)

export const caveApi = {
  getAll: () => request.get('/caves'),
  getById: (id) => request.get(`/caves/${id}`),
  getStatistics: () => request.get('/caves/statistics')
}

export const pointApi = {
  getByCave: (caveId) => request.get(`/points?caveId=${caveId}`),
  getById: (id) => request.get(`/points/${id}`)
}

export const sensorApi = {
  getLatest: (pointId) => {
    const url = pointId ? `/sensors/latest?pointId=${pointId}` : '/sensors/latest'
    return request.get(url)
  },
  getHistory: (pointId, days = 30) => request.get(`/sensors/history?pointId=${pointId}&days=${days}`),
  postData: (data) => request.post('/sensors/data', data),
  postBatch: (data) => request.post('/sensors/batch', data)
}

export const weatheringApi = {
  getRates: (pointId) => request.get(`/weathering/rates?pointId=${pointId}`),
  predict: (data) => request.post('/weathering/predict', data)
}

export const materialApi = {
  getAll: () => request.get('/materials'),
  recommend: (data) => request.post('/materials/recommend', data)
}

export const alertApi = {
  getAll: (limit = 50, acknowledged = null) => {
    let url = `/alerts?limit=${limit}`
    if (acknowledged !== null) url += `&acknowledged=${acknowledged}`
    return request.get(url)
  },
  acknowledge: (id, user = 'admin') => request.post(`/alerts/${id}/acknowledge`, { user })
}

export const dashboardApi = {
  getOverview: () => request.get('/dashboard/overview')
}

export const healthApi = {
  check: () => request.get('/health')
}

export default request
