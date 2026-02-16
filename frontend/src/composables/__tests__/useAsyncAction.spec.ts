import { describe, it, expect, vi } from 'vitest'
import { useAsyncAction } from '../useAsyncAction'

describe('useAsyncAction', () => {
  it('starts with default state', () => {
    const { data, error, isLoading } = useAsyncAction(async () => 'test')
    expect(data.value).toBeNull()
    expect(error.value).toBeNull()
    expect(isLoading.value).toBe(false)
  })

  it('sets data on successful execution', async () => {
    const fn = vi.fn().mockResolvedValue('result')
    const { data, error, isLoading, execute } = useAsyncAction(fn)

    const result = await execute()

    expect(result).toBe('result')
    expect(data.value).toBe('result')
    expect(error.value).toBeNull()
    expect(isLoading.value).toBe(false)
  })

  it('sets error on failed execution', async () => {
    const fn = vi.fn().mockRejectedValue(new Error('fail'))
    const { data, error, isLoading, execute } = useAsyncAction(fn)

    const result = await execute()

    expect(result).toBeNull()
    expect(data.value).toBeNull()
    expect(error.value).toBeInstanceOf(Error)
    expect(error.value?.message).toBe('fail')
    expect(isLoading.value).toBe(false)
  })

  it('wraps non-Error thrown values', async () => {
    const fn = vi.fn().mockRejectedValue('string error')
    const { error, execute } = useAsyncAction(fn)

    await execute()

    expect(error.value).toBeInstanceOf(Error)
    expect(error.value?.message).toBe('string error')
  })

  it('passes arguments through to the wrapped function', async () => {
    const fn = vi.fn().mockResolvedValue('ok')
    const { execute } = useAsyncAction(fn)

    await execute('arg1', 'arg2')

    expect(fn).toHaveBeenCalledWith('arg1', 'arg2')
  })

  it('resets error on subsequent execution', async () => {
    const fn = vi
      .fn()
      .mockRejectedValueOnce(new Error('fail'))
      .mockResolvedValueOnce('success')
    const { data, error, execute } = useAsyncAction(fn)

    await execute()
    expect(error.value).not.toBeNull()

    await execute()
    expect(error.value).toBeNull()
    expect(data.value).toBe('success')
  })
})
