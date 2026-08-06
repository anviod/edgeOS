import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { nodeApi } from '@/api'

function mockFetchResponse(payload: unknown, options?: { ok?: boolean; status?: number }) {
  const ok = options?.ok ?? true
  const status = options?.status ?? 200
  const json = JSON.stringify(payload)

  return vi.spyOn(globalThis, 'fetch').mockImplementation(async () => {
    return {
      ok,
      status,
      text: async () => json,
    } as unknown as Response
  })
}

describe('edgeCore API contract', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('nodeApi.list should request correct endpoint and return data', async () => {
    const responsePayload = {
      code: '0',
      msg: 'Success',
      data: {
        nodes: [
          { node_id: 'n-1', node_name: 'GW-1' },
        ],
      },
    }

    const fetchSpy = mockFetchResponse(responsePayload)
    const result = await nodeApi.list()

    expect(fetchSpy).toHaveBeenCalledWith(
      '/api/nodes',
      expect.objectContaining({
        headers: expect.objectContaining({
          'Content-Type': 'application/json; charset=utf-8',
        }),
      }),
    )
    expect(result).toEqual(responsePayload.data.nodes)
  })

  it('nodeApi.get should throw when business code is not success', async () => {
    mockFetchResponse({
      code: '1',
      msg: 'node not found',
      data: '',
    })

    // API 解析业务错误码并抛出 msg | API parses business error code and throws msg
    await expect(nodeApi.get('missing-node')).rejects.toThrow('node not found')
  })

  it('nodeApi.remove should throw on non-2xx response', async () => {
    mockFetchResponse({ code: '1', msg: 'server error', data: '' }, { ok: false, status: 500 })

    // API 解析业务错误码并抛出 msg | API parses business error code and throws msg
    await expect(nodeApi.remove('node-1')).rejects.toThrow('server error')
  })
})
