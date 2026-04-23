<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { initWebSocket, destroyWebSocket } from '@/composables/useWebSocket'
import { useWebSocket } from '@/composables/useWebSocket'
import { useEdgeStore } from '@/stores/edge'
import { useAlertStore } from '@/stores/alert'
import { useMiddlewareStore } from '@/stores/middleware'
import { useRealtimeStore } from '@/stores/realtime'
import type { NodeStatusPayload, DataUpdatePayload, CommandResponsePayload, MiddlewareStatusPayload, AlertInfo, DeviceOnlinePayload, DeviceOfflinePayload } from '@/types/edgex'

const route = useRoute()
const router = useRouter()
const { on } = useWebSocket()

const edgeStore = useEdgeStore()
const alertStore = useAlertStore()
const middlewareStore = useMiddlewareStore()
const realtimeStore = useRealtimeStore()

onMounted(() => {
  if (route.path !== '/login') {
    const token = localStorage.getItem('token')
    if (!token) {
      router.push('/login')
      return
    }
    startWS()
  }
})

function startWS() {
  initWebSocket()
  on<NodeStatusPayload>('node_status', payload => edgeStore.applyNodeStatus(payload))
  on<DataUpdatePayload>('data_update', payload => realtimeStore.applyDataUpdate(payload))
  on<{ node_id: string }>('device_synced', payload => edgeStore.applyDeviceSynced(payload.node_id))
  on<DeviceOnlinePayload>('device_online', payload => edgeStore.applyDeviceOnline(payload))
  on<DeviceOfflinePayload>('device_offline', payload => edgeStore.applyDeviceOffline(payload))
  on<CommandResponsePayload>('command_response', payload => realtimeStore.applyCommandResponse(payload))
  on<AlertInfo>('alert', payload => alertStore.applyNewAlert(payload))
  on<MiddlewareStatusPayload>('middleware_status', payload => middlewareStore.applyStatusEvent(payload))
  on<{ node_id: string; device_id: string; count?: number }>('point_synced', payload => {
    if (payload.device_id) {
      realtimeStore.fetchPoints(payload.node_id, payload.device_id)
    }
  })
  on('point_report', payload => realtimeStore.applyPointReport(payload))
}

onUnmounted(() => {
  destroyWebSocket()
})
</script>

<template>
  <router-view />
</template>
