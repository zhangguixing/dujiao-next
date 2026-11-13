import { computed, watch, type Ref } from 'vue'
import { useDocumentVisibility, useIntervalFn } from '@vueuse/core'
import type { ApiResponse } from '../api/client'

const awaitingDelivery = new Set(['paid', 'fulfilling', 'partially_delivered'])

export function useOrderDetailPolling(
  order: Ref,
  canRefresh: () => boolean,
  fetchOrder: () => Promise<{ data: ApiResponse }>,
) {
  const visibility = useDocumentVisibility()
  const enabled = computed(() => {
    if (visibility.value !== 'visible' || !canRefresh()) return false

    const current = order.value
    return awaitingDelivery.has(current?.status) || (
      current?.status === 'partially_refunded' &&
      current.children?.some((child: { status: string }) => awaitingDelivery.has(child.status))
    )
  })
  let refreshing = false

  const { isActive, pause, resume } = useIntervalFn(async () => {
    if (!enabled.value || refreshing) return

    const current = order.value
    refreshing = true
    try {
      const { data } = await fetchOrder()
      if (isActive.value && order.value === current && canRefresh() && data.data?.order_no === current.order_no) {
        order.value = data.data
      }
    } catch {
      // Keep the visible details and retry on the next interval.
    } finally {
      refreshing = false
    }
  }, 5000, { immediate: false })

  watch(enabled, active => {
    if (active) resume()
    else pause()
  }, { immediate: true })
}
