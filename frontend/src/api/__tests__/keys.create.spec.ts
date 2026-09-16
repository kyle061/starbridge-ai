import { beforeEach, describe, expect, it, vi } from 'vitest'
import { create } from '../keys'

const { post } = vi.hoisted(() => ({ post: vi.fn() }))
vi.mock('../client', () => ({ apiClient: { post } }))

describe('API key creation', () => {
  beforeEach(() => {
    post.mockReset()
    post.mockResolvedValue({ data: { id: 1, key: 'sk-test' } })
  })

  it('uses the account balance instead of sending a per-key quota', async () => {
    // The sixth positional argument is retained for callers compiled against
    // the old signature, but it must not cross the user API boundary.
    await create('default', 7, undefined, undefined, undefined, 25)

    expect(post).toHaveBeenCalledWith('/keys', { name: 'default', group_id: 7 })
  })
})
