import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { EdgeXPointInfo, DataUpdatePayload, CommandResponsePayload } from '@/types/edgex'
import { pointApi } from '@/api/index'

// 差量更新记录：`${nodeId}/${deviceId}/${pointId}` -> timestamp
const deltaMap = ref<Record<string, number>>({})

// 命令执行追踪：requestId -> status
export type CmdStatus = 'pending' | 'success' | 'error' | 'timeout'
export interface CmdTrack {
  request_id: string
  node_id: string
  device_id: string
  point_id: string
  value: unknown
  status: CmdStatus
  error?: string
  ts: number
}

export const useRealtimeStore = defineStore('realtime', () => {
  // pointsByDevice: `${nodeId}/${deviceId}` -> points map (pointId -> PointInfo)
  const pointsByDevice = ref<Record<string, Record<string, EdgeXPointInfo>>>({})
  const loadingDevices = ref<Set<string>>(new Set())

  // 差量高亮的 pointId 集合：`${nodeId}/${deviceId}/${pointId}`
  const deltaKeys = ref<Set<string>>(new Set())

  // 命令追踪列表（最多保留100条）
  const cmdTracks = ref<CmdTrack[]>([])

  function deviceKey(nodeId: string, deviceId: string) {
    return `${nodeId}/${deviceId}`
  }

  async function fetchPoints(nodeId: string, deviceId: string) {
    const key = deviceKey(nodeId, deviceId)
    loadingDevices.value.add(key)
    const { points, snapshot } = await pointApi.listByDevice(nodeId, deviceId).finally(() => {
      loadingDevices.value.delete(key)
    })
    const map: Record<string, EdgeXPointInfo> = {}
    points.forEach(p => {
      // 如果 API 返回了 snapshot，使用 snapshot 中的实时值
      if (snapshot && snapshot[p.point_id] !== undefined) {
        p.current_value = snapshot[p.point_id]
        p.last_updated = Date.now() / 1000
      }
      map[p.point_id] = p
    })
    pointsByDevice.value[key] = map
  }

  // WebSocket 数据更新事件
  function applyDataUpdate(payload: DataUpdatePayload) {
    const key = deviceKey(payload.node_id, payload.device_id)

    if (payload.is_full_snapshot) {
      // 全量替换
      const map: Record<string, EdgeXPointInfo> = {}
      Object.entries(payload.points).forEach(([pointId, value]) => {
        const existing = pointsByDevice.value[key]?.[pointId]
        map[pointId] = {
          ...(existing ?? {
            point_id: pointId,
            point_name: pointId,
            device_id: payload.device_id,
            service_name: '',
            profile_name: '',
            point_type: 'read',
            data_type: 'String',
            read_write: false,
            default_value: null,
            units: '',
            description: '',
            properties: {},
            last_sync: 0,
          }),
          current_value: value,
          quality: payload.quality,
          last_updated: payload.timestamp,
        }
      })
      pointsByDevice.value[key] = map
    } else {
      // 差量 Merge
      const map = pointsByDevice.value[key] ?? {}
      Object.entries(payload.points).forEach(([pointId, value]) => {
        const deltaKey = `${key}/${pointId}`
        if (map[pointId]) {
          map[pointId] = {
            ...map[pointId],
            current_value: value,
            quality: payload.quality,
            last_updated: payload.timestamp,
          }
        } else {
          map[pointId] = {
            point_id: pointId,
            point_name: pointId,
            device_id: payload.device_id,
            service_name: '',
            profile_name: '',
            point_type: 'read',
            data_type: 'String',
            read_write: false,
            default_value: null,
            units: '',
            description: '',
            properties: {},
            last_sync: 0,
            current_value: value,
            quality: payload.quality,
            last_updated: payload.timestamp,
          }
        }
        // 标记差量高亮
        deltaKeys.value.add(deltaKey)
        deltaMap.value[deltaKey] = Date.now()
        setTimeout(() => {
          deltaKeys.value.delete(deltaKey)
        }, 1500)
      })
      pointsByDevice.value[key] = { ...map }
    }
  }

  // WebSocket 命令响应事件
  function applyCommandResponse(payload: CommandResponsePayload) {
    console.log('Received command response:', payload)
    const index = cmdTracks.value.findIndex(t => t.request_id === payload.request_id)
    console.log('Found command track at index:', index)
    if (index !== -1) {
      const track = cmdTracks.value[index]
      console.log('Current track:', track)
      const updatedTrack = {
        ...track,
        status: payload.success ? 'success' : 'error',
        error: payload.error
      }
      console.log('Updated track:', updatedTrack)
      cmdTracks.value.splice(index, 1, updatedTrack)
      console.log('Command tracks after update:', cmdTracks.value)
    } else {
      console.log('Command track not found for request_id:', payload.request_id)
      console.log('Current command tracks:', cmdTracks.value)
    }
  }

  function addCmdTrack(track: CmdTrack) {
    cmdTracks.value.unshift(track)
    if (cmdTracks.value.length > 1000) cmdTracks.value.pop()
    
    // 添加 10 秒超时处理
    if (track.status === 'pending') {
      setTimeout(() => {
        const t = cmdTracks.value.find(t => t.request_id === track.request_id)
        if (t && t.status === 'pending') {
          t.status = 'timeout'
          t.error = '命令执行超时'
        }
      }, 10000) // 10 秒超时
    }
  }

  function isDeltaKey(nodeId: string, deviceId: string, pointId: string) {
    return deltaKeys.value.has(`${nodeId}/${deviceId}/${pointId}`)
  }

  function getPoints(nodeId: string, deviceId: string): EdgeXPointInfo[] {
    const key = deviceKey(nodeId, deviceId)
    return Object.values(pointsByDevice.value[key] ?? {})
  }

  function isLoading(nodeId: string, deviceId: string) {
    return loadingDevices.value.has(deviceKey(nodeId, deviceId))
  }

  // WebSocket 点位元数据上报事件
  function applyPointReport(payload: any) {
    const { body } = payload
    if (!body) return

    const { node_id, device_id, points } = body
    if (!node_id || !device_id || !Array.isArray(points)) return

    const key = deviceKey(node_id, device_id)
    const map = pointsByDevice.value[key] ?? {}

    points.forEach(p => {
      const pointId = p.point_id
      if (!pointId) return

      map[pointId] = {
        ...(map[pointId] ?? {
          point_id: pointId,
          device_id: device_id,
          service_name: '',
          profile_name: '',
          point_type: p.rw === 'RW' ? 'readwrite' : 'read',
          read_write: p.rw === 'RW',
          default_value: null,
          description: '',
          properties: {},
          last_sync: 0,
        }),
        point_name: p.point_name || pointId,
        data_type: p.data_type || 'String',
        units: p.unit || '',
      }
    })

    pointsByDevice.value[key] = { ...map }
  }

  return {
    pointsByDevice, deltaKeys, cmdTracks,
    fetchPoints, applyDataUpdate, applyCommandResponse, applyPointReport,
    addCmdTrack, isDeltaKey, getPoints, isLoading,
  }
})
