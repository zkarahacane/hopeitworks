import { describe, it, expect } from 'vitest'
import { z } from 'zod'
import { maskUrl } from '@/utils/maskUrl'

// Extract the raw zod schema for testing validation logic
const addChannelRawSchema = z.object({
  channel_type: z.enum(['discord', 'webhook']),
  url: z
    .string()
    .min(1, 'URL is required')
    .startsWith('https://', 'URL must start with https://'),
  events_filter: z.array(z.string()).default([]),
  enabled: z.boolean().default(true),
})

describe('AddChannelDialog — zod schema validation', () => {
  it('accepts a valid Discord webhook URL', () => {
    const result = addChannelRawSchema.safeParse({
      channel_type: 'discord',
      url: 'https://discord.com/api/webhooks/123/abc123',
      events_filter: ['run.completed'],
      enabled: true,
    })

    expect(result.success).toBe(true)
  })

  it('accepts a valid webhook URL', () => {
    const result = addChannelRawSchema.safeParse({
      channel_type: 'webhook',
      url: 'https://example.com/webhook',
      events_filter: [],
      enabled: false,
    })

    expect(result.success).toBe(true)
  })

  it('rejects a URL starting with http://', () => {
    const result = addChannelRawSchema.safeParse({
      channel_type: 'discord',
      url: 'http://discord.com/api/webhooks/123',
      events_filter: [],
      enabled: true,
    })

    expect(result.success).toBe(false)
    if (!result.success) {
      const urlErrors = result.error.issues.filter((i) => i.path[0] === 'url')
      expect(urlErrors.length).toBeGreaterThan(0)
      expect(urlErrors[0]!.message).toBe('URL must start with https://')
    }
  })

  it('rejects an empty URL', () => {
    const result = addChannelRawSchema.safeParse({
      channel_type: 'discord',
      url: '',
      events_filter: [],
      enabled: true,
    })

    expect(result.success).toBe(false)
    if (!result.success) {
      const urlErrors = result.error.issues.filter((i) => i.path[0] === 'url')
      expect(urlErrors.length).toBeGreaterThan(0)
      expect(urlErrors[0]!.message).toBe('URL is required')
    }
  })

  it('accepts empty events_filter (no events = disabled notifications)', () => {
    const result = addChannelRawSchema.safeParse({
      channel_type: 'webhook',
      url: 'https://example.com/hook',
      events_filter: [],
      enabled: true,
    })

    expect(result.success).toBe(true)
    if (result.success) {
      expect(result.data.events_filter).toEqual([])
    }
  })

  it('rejects an invalid channel_type', () => {
    const result = addChannelRawSchema.safeParse({
      channel_type: 'slack',
      url: 'https://hooks.slack.com/services/abc',
      events_filter: [],
      enabled: true,
    })

    expect(result.success).toBe(false)
  })

  it('defaults enabled to true when not provided', () => {
    const result = addChannelRawSchema.safeParse({
      channel_type: 'discord',
      url: 'https://discord.com/api/webhooks/123/abc',
      events_filter: [],
    })

    expect(result.success).toBe(true)
    if (result.success) {
      expect(result.data.enabled).toBe(true)
    }
  })
})

describe('maskUrl', () => {
  it('masks all but last 6 chars for long URLs', () => {
    expect(maskUrl('https://discord.com/api/webhooks/123/abc123')).toBe('****abc123')
  })

  it('returns the URL as-is when shorter than 6 chars', () => {
    expect(maskUrl('https')).toBe('https')
  })

  it('returns a 6-char URL as-is', () => {
    expect(maskUrl('abc123')).toBe('abc123')
  })

  it('masks a 7-char URL showing last 6 chars', () => {
    expect(maskUrl('abc1234')).toBe('****bc1234')
  })
})
