import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

import { useAnnouncementStore } from '@/stores/announcements'

const { listAnnouncements, markRead } = vi.hoisted(() => ({
  listAnnouncements: vi.fn(),
  markRead: vi.fn(),
}))

vi.mock('@/api', () => ({
  announcementsAPI: {
    list: listAnnouncements,
    markRead,
  },
}))

describe('useAnnouncementStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    vi.clearAllMocks()
    vi.spyOn(console, 'error').mockImplementation(() => {})
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-06-23T08:00:00Z'))
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.restoreAllMocks()
    localStorage.clear()
  })

  it('does not request announcements without an auth token', async () => {
    const store = useAnnouncementStore()

    await store.fetchAnnouncements(true)

    expect(listAnnouncements).not.toHaveBeenCalled()
  })

  it('backs off after transient fetch failures instead of retrying on every trigger', async () => {
    localStorage.setItem('auth_token', 'token')
    listAnnouncements.mockRejectedValueOnce({ status: 0, message: 'Network error' })
    const store = useAnnouncementStore()

    await store.fetchAnnouncements(true)
    await store.fetchAnnouncements()

    expect(listAnnouncements).toHaveBeenCalledTimes(1)

    listAnnouncements.mockResolvedValueOnce([])
    vi.advanceTimersByTime(60_001)
    await store.fetchAnnouncements()

    expect(listAnnouncements).toHaveBeenCalledTimes(2)
  })
})
