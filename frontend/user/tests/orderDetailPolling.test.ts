import test from 'node:test'
import assert from 'node:assert/strict'
import { effectScope, nextTick, ref } from 'vue'
import { useOrderDetailPolling } from '../src/composables/useOrderDetailPolling.ts'

test('order detail retries a failed refresh and stops after showing the delivery', async t => {
  t.mock.timers.enable({ apis: ['setInterval'] })
  const scope = effectScope()
  t.after(() => scope.stop())

  const initial = { order_no: 'DJ-1001', status: 'fulfilling' }
  const delivered = { ...initial, status: 'delivered', fulfillment: { payload: 'test-code' } }
  const order = ref(initial)
  const fetchOrder = t.mock.fn(async () => ({ data: { status_code: 0, msg: '', data: delivered } }))
  fetchOrder.mock.mockImplementationOnce(async () => { throw new Error('Network unavailable') })
  scope.run(() => useOrderDetailPolling(order, () => true, fetchOrder))

  t.mock.timers.tick(4999)
  assert.equal(fetchOrder.mock.callCount(), 0)
  t.mock.timers.tick(1)
  await nextTick()
  assert.deepEqual(order.value, initial)

  t.mock.timers.tick(5000)
  await nextTick()
  await nextTick()
  assert.deepEqual(order.value, delivered)

  t.mock.timers.tick(5000)
  assert.equal(fetchOrder.mock.callCount(), 2)
})

test('order detail avoids overlapping requests and discards a result after leaving', async t => {
  t.mock.timers.enable({ apis: ['setInterval'] })
  const scope = effectScope()
  t.after(() => scope.stop())

  const initial = { order_no: 'DJ-1002', status: 'paid' }
  const order = ref(initial)
  const delivered = { ...initial, status: 'delivered' }
  const response = { data: { status_code: 0, msg: '', data: delivered } }
  const { promise, resolve } = Promise.withResolvers<typeof response>()
  const fetchOrder = t.mock.fn(() => promise)
  scope.run(() => useOrderDetailPolling(order, () => true, fetchOrder))

  t.mock.timers.tick(5000)
  t.mock.timers.tick(5000)
  assert.equal(fetchOrder.mock.callCount(), 1)

  scope.stop()
  resolve(response)
  await nextTick()
  assert.deepEqual(order.value, initial)

  t.mock.timers.tick(5000)
  assert.equal(fetchOrder.mock.callCount(), 1)
})
