import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, post } = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    get,
    post,
  },
}))

import { paymentAPI } from '@/api/payment'

describe('payment api', () => {
  beforeEach(() => {
    get.mockReset()
    post.mockReset()
    get.mockResolvedValue({ data: {} })
    post.mockResolvedValue({ data: {} })
  })

  it('keeps legacy public out_trade_no verification for upgrade compatibility', async () => {
    await paymentAPI.verifyOrderPublic('legacy-order-no')

    expect(post).toHaveBeenCalledWith('/payment/public/orders/verify', {
      out_trade_no: 'legacy-order-no',
    })
  })

  it('keeps signed public resume-token resolve endpoint', async () => {
    await paymentAPI.resolveOrderPublicByResumeToken('resume-token-123')

    expect(post).toHaveBeenCalledWith('/payment/public/orders/resolve', {
      resume_token: 'resume-token-123',
    })
  })

  it('sends an idempotency key when creating a payment order', async () => {
    await paymentAPI.createOrder({
      amount: 10,
      payment_type: 'stripe',
      order_type: 'balance',
    }, 'payment-attempt-123')

    expect(post).toHaveBeenCalledWith(
      '/payment/orders',
      {
        amount: 10,
        payment_type: 'stripe',
        order_type: 'balance',
      },
      { headers: { 'Idempotency-Key': 'payment-attempt-123' } },
    )
  })

  it('passes order filters through to the user order history endpoint', async () => {
    await paymentAPI.getMyOrders({
      page: 2,
      page_size: 100,
      status: 'COMPLETED',
      order_type: 'balance',
      payment_type: 'stripe',
    })

    expect(get).toHaveBeenCalledWith('/payment/orders/my', {
      params: {
        page: 2,
        page_size: 100,
        status: 'COMPLETED',
        order_type: 'balance',
        payment_type: 'stripe',
      },
    })
  })
})
