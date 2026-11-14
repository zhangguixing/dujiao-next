import { api } from '../api/client'

const visitorStorageKey = 'dujiao_home_visitor_id'
let visitorId = ''

function getVisitorId() {
  try {
    const saved = localStorage.getItem(visitorStorageKey)
    if (saved && /^[a-f0-9]{32}$/.test(saved)) return saved
  } catch {
    // Keep a visitor identifier for the current page when storage is unavailable.
  }

  if (!visitorId) {
    const bytes = crypto.getRandomValues(new Uint8Array(16))
    visitorId = Array.from(bytes, byte => byte.toString(16).padStart(2, '0')).join('')
  }

  try {
    localStorage.setItem(visitorStorageKey, visitorId)
  } catch {
    return visitorId
  }
  return visitorId
}

export async function recordHomeVisit() {
  try {
    await api.post('/public/home-visitors', { visitor_id: getVisitorId() })
  } catch {
    // Analytics failures do not affect storefront navigation.
  }
}
