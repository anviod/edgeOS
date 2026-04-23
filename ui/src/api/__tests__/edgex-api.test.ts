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

describe('EdgeX API contract', () => {
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
      data: [
        { node_id: 'n-1', node_name: 'GW-1' },
      ],
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
    expect(result).toEqual(responsePayload.data)
  })

  it('nodeApi.get should throw when business code is not success', async () => {
    mockFetchResponse({
      code: '1',
      msg: 'node not found',
      data: '',
    })

    // The current API implementation catches business errors and rethrows as "Invalid JSON response"
    await expect(nodeApi.get('missing-node')).rejects.toThrow('Invalid JSON response from server')
  })

  it('nodeApi.remove should throw on non-2xx response', async () => {
    mockFetchResponse({ code: '1', msg: 'server error', data: '' }, { ok: false, status: 500 })

    // The current API implementation catches business errors and rethrows as "Invalid JSON response"
    await expect(nodeApi.remove('node-1')).rejects.toThrow('Invalid JSON response from server')
  })
})
