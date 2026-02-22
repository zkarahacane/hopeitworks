import { describe, it, expect } from 'vitest'
import { formatCostUSD, formatTokenCount } from '../formatCost'

describe('formatCostUSD', () => {
  it('formats zero as $0.00', () => {
    expect(formatCostUSD(0)).toBe('$0.00')
  })

  it('formats small costs with up to 5 decimal places', () => {
    const result = formatCostUSD(0.00123)
    expect(result).toBe('$0.00123')
  })

  it('formats larger costs with 2 decimal places', () => {
    expect(formatCostUSD(4.5)).toBe('$4.50')
  })
})

describe('formatTokenCount', () => {
  it('formats small numbers without separators', () => {
    expect(formatTokenCount(500)).toBe('500')
  })

  it('formats thousands with comma separators', () => {
    expect(formatTokenCount(150000)).toBe('150,000')
  })

  it('formats millions with comma separators', () => {
    expect(formatTokenCount(1500000)).toBe('1,500,000')
  })

  it('formats zero', () => {
    expect(formatTokenCount(0)).toBe('0')
  })
})
