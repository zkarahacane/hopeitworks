import { describe, it, expect } from 'vitest'
import { usePagination } from '../usePagination'

describe('usePagination', () => {
  it('starts with default values', () => {
    const { page, perPage, total, totalPages } = usePagination()
    expect(page.value).toBe(1)
    expect(perPage.value).toBe(20)
    expect(total.value).toBe(0)
    expect(totalPages.value).toBe(0)
  })

  it('accepts custom perPage option', () => {
    const { perPage } = usePagination({ perPage: 10 })
    expect(perPage.value).toBe(10)
  })

  it('computes totalPages correctly', () => {
    const { totalPages, setTotal } = usePagination({ perPage: 10 })
    setTotal(25)
    expect(totalPages.value).toBe(3)
  })

  it('navigates to next page', () => {
    const { page, nextPage, setTotal } = usePagination({ perPage: 10 })
    setTotal(30)
    nextPage()
    expect(page.value).toBe(2)
  })

  it('does not go past last page', () => {
    const { page, nextPage, setTotal } = usePagination({ perPage: 10 })
    setTotal(20)
    nextPage()
    nextPage()
    nextPage()
    expect(page.value).toBe(2)
  })

  it('navigates to previous page', () => {
    const { page, nextPage, prevPage, setTotal } = usePagination({ perPage: 10 })
    setTotal(30)
    nextPage()
    nextPage()
    expect(page.value).toBe(3)
    prevPage()
    expect(page.value).toBe(2)
  })

  it('does not go below page 1', () => {
    const { page, prevPage } = usePagination()
    prevPage()
    expect(page.value).toBe(1)
  })

  it('goes to a specific page', () => {
    const { page, goToPage, setTotal } = usePagination({ perPage: 10 })
    setTotal(50)
    goToPage(3)
    expect(page.value).toBe(3)
  })

  it('clamps goToPage to valid range', () => {
    const { page, goToPage, setTotal } = usePagination({ perPage: 10 })
    setTotal(30)
    goToPage(100)
    expect(page.value).toBe(3)
    goToPage(0)
    expect(page.value).toBe(1)
  })

  it('resets page to 1', () => {
    const { page, nextPage, reset, setTotal } = usePagination({ perPage: 10 })
    setTotal(30)
    nextPage()
    nextPage()
    expect(page.value).toBe(3)
    reset()
    expect(page.value).toBe(1)
  })
})
