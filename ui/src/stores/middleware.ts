import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { MiddlewareConfig, MiddlewareForm, MiddlewareStatusPayload } from '@/types/edgex'
import { middlewareApi } from '@/api/index'

export const useMiddlewareStore = defineStore('middleware', () => {
  const list = ref<MiddlewareConfig[]>([])
  const loading = ref(false)

  async function fetchList() {
    loading.value = true
    try {
      const result = await middlewareApi.list()
      list.value = Array.isArray(result) ? result : []
    } catch (error) {
      console.error('Failed to fetch middlewares:', error)
      list.value = []
    } finally {
      loading.value = false
    }
  }

  async function create(form: MiddlewareForm) {
    const item = await middlewareApi.create(form)
    list.value.push(item)
    return item
  }

  async function update(id: string, form: Partial<MiddlewareForm>) {
    const item = await middlewareApi.update(id, form)
    const idx = list.value.findIndex(m => m.id === id)
    if (idx !== -1) list.value[idx] = item
    return item
  }

  async function remove(id: string) {
    await middlewareApi.remove(id)
    list.value = list.value.filter(m => m.id !== id)
  }

  async function connect(id: string) {
    await middlewareApi.connect(id)
    // 重新获取中间件列表，确保状态更新
    await fetchList()
  }

  // WebSocket 实时更新
  function applyStatusEvent(payload: MiddlewareStatusPayload) {
    const item = list.value.find(m => m.id === payload.id)
    if (item) {
      item.status = payload.status
      if (payload.error) item.last_error = payload.error
    }
  }

  return { list, loading, fetchList, create, update, remove, connect, applyStatusEvent }
})
