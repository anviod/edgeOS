import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { EdgeXNodeInfo, EdgeXDeviceInfo, NodeStatusPayload, DashboardStats, DeviceOnlinePayload, DeviceOfflinePayload } from '@/types/edgex'
import { nodeApi, deviceApi, dashboardApi } from '@/api/index'

export const useEdgeStore = defineStore('edge', () => {
  const nodes = ref<EdgeXNodeInfo[]>([])
  const nodesLoading = ref(false)

  // devicesByNode: nodeId -> devices
  const devicesByNode = ref<Record<string, EdgeXDeviceInfo[]>>({})
  const devicesLoading = ref(false)

  const stats = ref<DashboardStats>({
    total_nodes: 0,
    online_nodes: 0,
    total_devices: 0,
    today_alerts: 0,
  })

  async function fetchStats() {
    stats.value = await dashboardApi.getStats()
  }

  async function fetchNodes() {
    nodesLoading.value = true
    try {
      const result = await nodeApi.list()
      nodes.value = Array.isArray(result) ? result : []
    } catch (error) {
      console.error('Failed to fetch nodes:', error)
      nodes.value = []
    } finally {
      nodesLoading.value = false
    }
  }

  async function fetchDevices(nodeId: string) {
    devicesLoading.value = true
    try {
      const result = await deviceApi.listByNode(nodeId)
      devicesByNode.value[nodeId] = Array.isArray(result) ? result : []
    } catch (error) {
      console.error('Failed to fetch devices:', error)
      devicesByNode.value[nodeId] = []
    } finally {
      devicesLoading.value = false
    }
  }

  async function removeNode(nodeId: string) {
    await nodeApi.remove(nodeId)
    nodes.value = nodes.value.filter(n => n.node_id !== nodeId)
  }

  // WebSocket 实时节点状态
  function applyNodeStatus(payload: NodeStatusPayload) {
    console.log('Received node status update:', payload)
    const node = nodes.value.find(n => n.node_id === payload.node_id)
    if (node) {
      node.status = payload.status
      node.last_seen = payload.last_seen
      console.log('Updated node:', node.node_id, 'last_seen:', node.last_seen)
    } else {
      // 新节点注册，触发重新拉取
      fetchNodes()
    }
  }

  // WebSocket 设备同步事件
  function applyDeviceSynced(nodeId: string) {
    fetchDevices(nodeId)
  }

  // WebSocket 设备上线事件
  function applyDeviceOnline(payload: DeviceOnlinePayload) {
    const devices = devicesByNode.value[payload.node_id]
    if (devices) {
      const device = devices.find(d => d.device_id === payload.device_id)
      if (device) {
        device.operating_state = 'online'
      }
    }
  }

  // WebSocket 设备下线事件
  function applyDeviceOffline(payload: DeviceOfflinePayload) {
    const devices = devicesByNode.value[payload.node_id]
    if (devices) {
      const device = devices.find(d => d.device_id === payload.device_id)
      if (device) {
        device.operating_state = 'offline'
      }
    }
  }

  return {
    nodes, nodesLoading,
    devicesByNode, devicesLoading,
    stats,
    fetchStats, fetchNodes, fetchDevices, removeNode,
    applyNodeStatus, applyDeviceSynced, applyDeviceOnline, applyDeviceOffline,
  }
})
