<template>
  <router-view />
</template>

<script setup>
import { onMounted } from 'vue'
import { useMainStore } from '@/store/main'
import { initWebSocket } from '@/utils/websocket'

const store = useMainStore()

onMounted(() => {
  store.loadInitialData()
  initWebSocket()
})
</script>
