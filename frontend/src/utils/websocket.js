import { ElNotification } from 'element-plus'
import { useMainStore } from '@/store/main'

let ws = null
let reconnectTimer = null
let heartbeatTimer = null
let reconnectCount = 0
const MAX_RECONNECT = 10

export function initWebSocket() {
  if (ws) return

  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const wsBase = import.meta.env.VITE_WS_BASE || `${protocol}//${window.location.host}`
  const url = `${wsBase}/api/v1/ws`

  try {
    ws = new WebSocket(url)
    setupEventHandlers()
  } catch (e) {
    console.error('WebSocket init error:', e)
    scheduleReconnect()
  }
}

function setupEventHandlers() {
  if (!ws) return

  ws.onopen = () => {
    console.log('[WebSocket] 连接已建立')
    reconnectCount = 0
    startHeartbeat()
  }

  ws.onmessage = (event) => {
    try {
      const msg = JSON.parse(event.data)
      handleMessage(msg)
    } catch (e) {
      console.error('WebSocket parse error:', e)
    }
  }

  ws.onerror = (e) => {
    console.error('[WebSocket] 错误:', e)
    stopHeartbeat()
  }

  ws.onclose = () => {
    console.warn('[WebSocket] 连接已关闭')
    stopHeartbeat()
    scheduleReconnect()
  }
}

function handleMessage(msg) {
  const store = useMainStore()

  switch (msg.type) {
    case 'ALERT':
      const alert = msg.payload
      store.addAlert(alert)
      showAlertNotification(alert)
      break
    case 'SENSOR_DATA':
      store.updateSensorData(msg.payload)
      break
    case 'STATUS':
      console.log('[WS Status]', msg.payload)
      break
  }
}

function showAlertNotification(alert) {
  const typeMap = {
    'CRITICAL': { type: 'error', duration: 10000, icon: '🚨' },
    'WARNING':  { type: 'warning', duration: 8000, icon: '⚠️' },
    'INFO':     { type: 'info', duration: 5000, icon: 'ℹ️' }
  }
  const cfg = typeMap[alert.severity] || typeMap['INFO']

  ElNotification({
    title: `${cfg.icon}  ${alert.title}`,
    message: alert.message,
    type: cfg.type,
    duration: cfg.duration,
    offset: 80,
    onClick: () => {
      window.location.hash = '#/alerts'
    }
  })
}

function startHeartbeat() {
  stopHeartbeat()
  heartbeatTimer = setInterval(() => {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ type: 'PING', time: Date.now() }))
    }
  }, 30000)
}

function stopHeartbeat() {
  if (heartbeatTimer) {
    clearInterval(heartbeatTimer)
    heartbeatTimer = null
  }
}

function scheduleReconnect() {
  if (reconnectTimer) return
  if (reconnectCount >= MAX_RECONNECT) {
    console.error('[WebSocket] 超过最大重连次数，停止重连')
    return
  }

  const delay = Math.min(1000 * Math.pow(2, reconnectCount), 30000)
  reconnectCount++
  console.log(`[WebSocket] 将在 ${delay/1000}s 后进行第 ${reconnectCount} 次重连`)

  reconnectTimer = setTimeout(() => {
    reconnectTimer = null
    ws = null
    initWebSocket()
  }, delay)
}

export function closeWebSocket() {
  stopHeartbeat()
  if (reconnectTimer) {
    clearTimeout(reconnectTimer)
    reconnectTimer = null
  }
  if (ws) {
    ws.close()
    ws = null
  }
}
