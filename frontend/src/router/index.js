import { createRouter, createWebHashHistory } from 'vue-router'
import MainLayout from '@/views/MainLayout.vue'
import Dashboard from '@/views/Dashboard.vue'
import Grotto3DView from '@/views/Grotto3DView.vue'
import WeatheringAnalysis from '@/views/WeatheringAnalysis.vue'
import MaterialRecommendation from '@/views/MaterialRecommendation.vue'
import AlertCenter from '@/views/AlertCenter.vue'

const routes = [
  {
    path: '/',
    component: MainLayout,
    redirect: '/dashboard',
    children: [
      {
        path: 'dashboard',
        name: 'Dashboard',
        component: Dashboard,
        meta: { title: '总览看板', icon: 'DataAnalysis' }
      },
      {
        path: 'grotto3d/:id?',
        name: 'Grotto3D',
        component: Grotto3DView,
        meta: { title: '石窟三维监测', icon: 'Cpu' }
      },
      {
        path: 'weathering',
        name: 'WeatheringAnalysis',
        component: WeatheringAnalysis,
        meta: { title: '风化速率分析', icon: 'TrendCharts' }
      },
      {
        path: 'materials',
        name: 'MaterialRecommendation',
        component: MaterialRecommendation,
        meta: { title: '保护材料优选', icon: 'Goods' }
      },
      {
        path: 'alerts',
        name: 'AlertCenter',
        component: AlertCenter,
        meta: { title: '告警中心', icon: 'Warning' }
      }
    ]
  }
]

const router = createRouter({
  history: createWebHashHistory(),
  routes
})

export default router
