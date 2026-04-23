import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { AlertInfo } from '@/types/edgex'
import { alertApi } from '@/api/index'

export const useAlertStore = defineStore('alert', () => {
  const alerts = ref<AlertInfo[]>([])
  const loading = ref(false)

  const unacknowledgedCount = computed(
    () => alerts.value.filter(a => a.status === 'active').length
  )

  async function fetchAlerts(status?: string) {
    loading.value = true
    try {
      const result = await alertApi.list(status)
      alerts.value = Array.isArray(result) ? result : []
    } catch (error) {
      console.error('Failed to fetch alerts:', error)
      alerts.value = []
    } finally {
      loading.value = false
    }
  }

  async function acknowledge(id: string) {
    await alertApi.acknowledge(id)
    const alert = alerts.value.find(a => a.id === id)
    if (alert) {
      alert.status = 'acknowledged'
      alert.acknowledged_at = Date.now()
    }
  }

  // WebSocket 实时新增告警
  function applyNewAlert(payload: AlertInfo) {
    const existing = alerts.value.find(a => a.id === payload.id)
    if (!existing) {
      alerts.value.unshift(payload)
    }
  }

  return {
    alerts, loading,
    unacknowledgedCount,
    fetchAlerts, acknowledge, applyNewAlert,
  }
})
