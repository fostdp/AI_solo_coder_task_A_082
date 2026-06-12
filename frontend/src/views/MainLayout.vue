<template>
  <el-container class="main-layout">
    <el-aside class="sidebar">
      <div class="logo">
        <div class="logo-icon">🏛️</div>
        <div class="logo-text">
          <div class="title">石窟风化监测</div>
          <div class="subtitle">Grotto Monitor</div>
        </div>
      </div>

      <el-menu
        :default-active="activeMenu"
        router
        class="nav-menu"
        background-color="transparent"
        text-color="#94a3b8"
        active-text-color="#f59e0b"
      >
        <el-menu-item
          v-for="route in menuRoutes"
          :key="route.path"
          :index="route.path"
        >
          <el-icon><component :is="route.meta.icon" /></el-icon>
          <span>{{ route.meta.title }}</span>
        </el-menu-item>
      </el-menu>

      <div class="sidebar-footer">
        <div class="status-dot" :class="{ online: true }"></div>
        <span>系统正常运行</span>
      </div>
    </el-aside>

    <el-container class="main-area">
      <el-header class="app-header">
        <div class="header-left">
          <el-breadcrumb separator="/">
            <el-breadcrumb-item :to="{ path: '/dashboard' }">首页</el-breadcrumb-item>
            <el-breadcrumb-item>{{ currentTitle }}</el-breadcrumb-item>
          </el-breadcrumb>
        </div>
        <div class="header-right">
          <el-select
            v-model="store.selectedCaveId"
            class="cave-selector"
            placeholder="选择石窟"
            size="default"
            @change="handleCaveChange"
          >
            <el-option
              v-for="cave in store.caves"
              :key="cave.id"
              :label="cave.name"
              :value="cave.id"
            />
          </el-select>

          <el-popover
            placement="bottom-end"
            :width="400"
            trigger="click"
            popper-class="alert-popover"
          >
            <template #reference>
              <el-badge :value="store.activeAlerts.length" :max="99" class="alert-badge">
                <el-button circle size="default" type="danger" plain>
                  <el-icon :size="18"><Bell /></el-icon>
                </el-button>
              </el-badge>
            </template>
            <div class="alert-preview">
              <div class="alert-preview-title">
                <span>最新告警</span>
                <el-tag size="small" type="danger" effect="plain">{{ store.activeAlerts.length }} 未处理</el-tag>
              </div>
              <el-divider style="margin: 8px 0" />
              <div v-if="store.recentAlerts.length === 0" class="empty-alert">
                暂无告警
              </div>
              <div v-else class="alert-list">
                <div
                  v-for="alert in store.recentAlerts.slice(0, 5)"
                  :key="alert.id"
                  class="alert-item"
                  @click="goToAlerts"
                >
                  <span class="badge-tag" :class="severityClass(alert.severity)">{{ alert.severity }}</span>
                  <span class="alert-title">{{ alert.title }}</span>
                  <span class="alert-time">{{ formatTime(alert.createdAt) }}</span>
                </div>
              </div>
              <div class="alert-footer" @click="goToAlerts">查看全部 →</div>
            </div>
          </el-popover>

          <el-tooltip content="WebSocket状态">
            <div class="ws-indicator" :class="wsStatus">
              <div class="dot"></div>
            </div>
          </el-tooltip>
        </div>
      </el-header>

      <el-main class="app-main">
        <router-view v-slot="{ Component }">
          <transition name="fade" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup>import { computed, ref, onMounted } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useMainStore } from '@/store/main';
import { Bell, DataAnalysis, Cpu, TrendCharts, Goods, Warning } from '@element-plus/icons-vue';
import dayjs from 'dayjs';
const store = useMainStore();
const route = useRoute();
const router = useRouter();
const wsStatus = ref('connected');
const menuRoutes = [
 { path: '/dashboard', meta: { title: '总览看板', icon: DataAnalysis } },
 { path: '/grotto3d', meta: { title: '石窟三维监测', icon: Cpu } },
 { path: '/weathering', meta: { title: '风化速率分析', icon: TrendCharts } },
 { path: '/materials', meta: { title: '保护材料优选', icon: Goods } },
 { path: '/alerts', meta: { title: '告警中心', icon: Warning } }
];
const activeMenu = computed(() => route.path);
const currentTitle = computed(() => route.meta.title || '');
function handleCaveChange() {
 store.loadMonitoringPoints();
}
function severityClass(sev) {
 const map = { CRITICAL: 'critical', WARNING: 'warning', INFO: 'info' };
 return map[sev] || 'default';
}
function formatTime(t) {
 return dayjs(t).format('MM-DD HH:mm');
}
function goToAlerts() {
 router.push('/alerts');
}
onMounted(() => {
 window.addEventListener('online', () => wsStatus.value = 'connected');
 window.addEventListener('offline', () => wsStatus.value = 'disconnected');
});
</script>

<style lang="scss" scoped>
.main-layout {
  width: 100%;
  height: 100%;
}

.sidebar {
  width: 230px !important;
  background: linear-gradient(180deg, #0f172a 0%, #020617 100%);
  border-right: 1px solid var(--border-color);
  display: flex;
  flex-direction: column;

  .logo {
    padding: 20px 16px;
    display: flex;
    align-items: center;
    gap: 12px;
    border-bottom: 1px solid var(--border-color);

    .logo-icon {
      font-size: 32px;
      filter: drop-shadow(0 0 8px rgba(245,158,11,0.3));
    }
    .logo-text {
      .title {
        font-size: 16px;
        font-weight: 700;
        color: var(--text-primary);
        letter-spacing: 1px;
      }
      .subtitle {
        font-size: 10px;
        color: var(--text-muted);
        letter-spacing: 2px;
        margin-top: 2px;
      }
    }
  }

  .nav-menu {
    flex: 1;
    padding: 12px 8px;
    :deep(.el-menu-item) {
      border-radius: 8px;
      margin: 4px 0;
      height: 44px;
      line-height: 44px;
      &.is-active {
        background: linear-gradient(90deg, rgba(245,158,11,0.15), transparent);
        border-left: 3px solid var(--accent-gold);
      }
    }
  }

  .sidebar-footer {
    padding: 14px 16px;
    font-size: 12px;
    color: var(--text-muted);
    display: flex;
    align-items: center;
    gap: 8px;
    border-top: 1px solid var(--border-color);

    .status-dot {
      width: 8px; height: 8px;
      border-radius: 50%;
      background: var(--accent-green);
      box-shadow: 0 0 8px var(--accent-green);
      animation: pulse 2s infinite;
      &.online { background: var(--accent-green); }
    }
    @keyframes pulse {
      0%,100% { opacity: 1; }
      50% { opacity: 0.5; }
    }
  }
}

.main-area {
  display: flex;
  flex-direction: column;
  background: var(--primary-bg);
}

.app-header {
  height: 60px !important;
  padding: 0 20px !important;
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: var(--panel-bg);
  border-bottom: 1px solid var(--border-color);
  backdrop-filter: blur(10px);

  .header-right {
    display: flex;
    align-items: center;
    gap: 12px;

    .cave-selector {
      width: 220px;
    }
    .alert-badge {
      :deep(.el-badge__content) { background: var(--accent-red); border: none; }
    }
    .ws-indicator {
      width: 36px; height: 36px;
      border-radius: 50%;
      display: flex; align-items: center; justify-content: center;
      background: rgba(16,185,129,0.1);
      border: 1px solid rgba(16,185,129,0.3);
      cursor: help;
      &.connected .dot { background: var(--accent-green); box-shadow: 0 0 8px var(--accent-green); }
      &.disconnected { background: rgba(239,68,68,0.1); border-color: rgba(239,68,68,0.3); .dot { background: var(--accent-red); } }
      .dot { width: 8px; height: 8px; border-radius: 50%; }
    }
  }
}

.app-main {
  padding: 16px;
  overflow: auto;
  background: var(--primary-bg);
}

.fade-enter-active, .fade-leave-active {
  transition: opacity 0.2s ease;
}
.fade-enter-from, .fade-leave-to {
  opacity: 0;
}

:deep(.alert-popover) {
  background: var(--secondary-bg) !important;
  border: 1px solid var(--border-color) !important;

  .alert-preview {
    .alert-preview-title {
      display: flex;
      align-items: center;
      justify-content: space-between;
      font-weight: 600;
      font-size: 14px;
      color: var(--text-primary);
    }
    .empty-alert {
      text-align: center;
      padding: 24px;
      color: var(--text-muted);
      font-size: 13px;
    }
    .alert-list {
      max-height: 320px;
      overflow-y: auto;
    }
    .alert-item {
      display: flex;
      align-items: center;
      gap: 8px;
      padding: 8px 6px;
      border-radius: 6px;
      cursor: pointer;
      &:hover { background: rgba(59,130,246,0.06); }
      .alert-title {
        flex: 1;
        font-size: 13px;
        color: var(--text-primary);
        overflow: hidden;
        white-space: nowrap;
        text-overflow: ellipsis;
      }
      .alert-time {
        font-size: 11px;
        color: var(--text-muted);
        flex-shrink: 0;
      }
    }
    .alert-footer {
      margin-top: 10px;
      padding-top: 8px;
      border-top: 1px solid var(--border-color);
      text-align: right;
      color: var(--accent-gold);
      font-size: 13px;
      cursor: pointer;
      &:hover { text-decoration: underline; }
    }
  }
}
</style>
