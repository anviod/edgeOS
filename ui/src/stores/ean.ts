import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type {
  EANAgentDescriptor,
  EANCapabilityDescriptor,
  EANPointChangeEvent,
  EANAuditRecord,
  EANHealth,
  EANInvokeCallResult,
  EANInvokeRequest,
} from '@/types/ean'
import { eanApi } from '@/api/index'

/**
 * EAN Store - EAN 2.0 协调中心状态管理
 * 管理发现索引、事件流、审计记录和健康状态
 */
export const useEanStore = defineStore('ean', () => {
  const health = ref<EANHealth | null>(null)
  const agents = ref<EANAgentDescriptor[]>([])
  const capabilitiesByAgent = ref<Record<string, EANCapabilityDescriptor[]>>({})
  const events = ref<EANPointChangeEvent[]>([])
  const auditRecords = ref<EANAuditRecord[]>([])

  const loading = ref(false)
  const healthLoading = ref(false)
  const lastError = ref('')

  const onlineAgents = computed(() =>
    agents.value.filter(a => a.status === 'online')
  )
  const offlineAgents = computed(() =>
    agents.value.filter(a => a.status === 'offline')
  )
  const onlineCount = computed(() => onlineAgents.value.length)
  const totalAgents = computed(() => agents.value.length)
  /** health.status=ok 且 Bus 已 Start */
  const isEanEnabled = computed(() =>
    health.value?.status === 'ok' && health.value?.started === true
  )
  const isEanDisabled = computed(() =>
    health.value != null && !isEanEnabled.value
  )

  async function fetchHealth() {
    healthLoading.value = true
    try {
      health.value = await eanApi.health()
      if (health.value?.status === 'disabled') {
        lastError.value = health.value.message || 'EAN Bus 未启用'
      } else {
        lastError.value = ''
      }
    } catch (error) {
      console.error('Failed to fetch EAN health:', error)
      health.value = null
      lastError.value = (error as Error).message || '健康检查失败'
    } finally {
      healthLoading.value = false
    }
  }

  async function fetchAgents() {
    loading.value = true
    try {
      agents.value = await eanApi.listAgents()
      lastError.value = ''
    } catch (error) {
      console.error('Failed to fetch EAN agents:', error)
      agents.value = []
      lastError.value = (error as Error).message || '加载 Agent 失败'
    } finally {
      loading.value = false
    }
  }

  async function fetchAgentCapabilities(agentId: string) {
    try {
      const caps = await eanApi.getAgentCapabilities(agentId)
      capabilitiesByAgent.value[agentId] = caps
      return caps
    } catch (error) {
      console.error('Failed to fetch capabilities:', error)
      capabilitiesByAgent.value[agentId] = []
      lastError.value = (error as Error).message || '加载 Capability 失败'
      return []
    }
  }

  async function fetchEvents(n: number = 100) {
    try {
      events.value = await eanApi.recentEvents(n)
    } catch (error) {
      console.error('Failed to fetch EAN events:', error)
      events.value = []
      lastError.value = (error as Error).message || '加载事件失败'
    }
  }

  async function fetchAuditRecords(limit: number = 100) {
    try {
      auditRecords.value = await eanApi.auditRecords(limit)
    } catch (error) {
      console.error('Failed to fetch audit records:', error)
      auditRecords.value = []
      lastError.value = (error as Error).message || '加载审计失败'
    }
  }

  async function invokeCapability(req: EANInvokeRequest): Promise<EANInvokeCallResult | null> {
    try {
      const result = await eanApi.invoke(req)
      await fetchAuditRecords(100)
      lastError.value = ''
      return result
    } catch (error) {
      console.error('Invoke failed:', error)
      lastError.value = (error as Error).message || 'Invoke 失败'
      throw error
    }
  }

  async function fetchAll() {
    await Promise.all([
      fetchHealth(),
      fetchAgents(),
      fetchEvents(100),
      fetchAuditRecords(100),
    ])
  }

  return {
    health, agents, capabilitiesByAgent, events, auditRecords,
    loading, healthLoading, lastError,
    onlineAgents, offlineAgents, onlineCount, totalAgents, isEanEnabled, isEanDisabled,
    fetchHealth, fetchAgents, fetchAgentCapabilities,
    fetchEvents, fetchAuditRecords, invokeCapability, fetchAll,
  }
})
