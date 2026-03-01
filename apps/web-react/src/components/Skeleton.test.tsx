import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { Skeleton, SkeletonCard, SkeletonTable, SkeletonText } from '../components/Skeleton'

describe('Skeleton', () => {
  it('renders with default classes', () => {
    render(<Skeleton />)
    const skeleton = screen.getByRole('generic')
    expect(skeleton).toHaveClass('animate-pulse', 'rounded-md', 'bg-slate-200', 'dark:bg-slate-700')
  })

  it('renders with custom className', () => {
    render(<Skeleton className="custom-class" />)
    const skeleton = screen.getByRole('generic')
    expect(skeleton).toHaveClass('custom-class')
  })

  it('passes through other props', () => {
    render(<Skeleton data-testid="skeleton" />)
    const skeleton = screen.getByTestId('skeleton')
    expect(skeleton).toBeInTheDocument()
  })
})

describe('SkeletonCard', () => {
  it('renders skeleton card structure', () => {
    render(<SkeletonCard />)
    const card = screen.getByRole('generic')
    expect(card).toHaveClass('rounded-2xl', 'border', 'bg-white', 'p-5', 'shadow-sm')
  })

  it('contains multiple skeleton lines', () => {
    render(<SkeletonCard />)
    const skeletons = screen.getAllByRole('generic').filter((el: HTMLElement) =>
      el.classList.contains('animate-pulse')
    )
    expect(skeletons.length).toBeGreaterThan(1)
  })
})

describe('SkeletonTable', () => {
  it('renders with default rows and cols', () => {
    render(<SkeletonTable />)
    // Should render table header + 5 rows * 4 cols = 20 skeleton elements
    const skeletons = screen.getAllByRole('generic').filter((el: HTMLElement) =>
      el.classList.contains('animate-pulse')
    )
    expect(skeletons.length).toBe(24) // 4 header + 5 rows * 4 cols
  })

  it('renders with custom rows and cols', () => {
    render(<SkeletonTable rows={3} cols={2} />)
    const skeletons = screen.getAllByRole('generic').filter((el: HTMLElement) =>
      el.classList.contains('animate-pulse')
    )
    expect(skeletons.length).toBe(8) // 2 header + 3 rows * 2 cols
  })
})

describe('SkeletonText', () => {
  it('renders with default lines', () => {
    render(<SkeletonText />)
    const skeletons = screen.getAllByRole('generic').filter((el: HTMLElement) =>
      el.classList.contains('animate-pulse')
    )
    expect(skeletons.length).toBe(3)
  })

  it('renders with custom lines', () => {
    render(<SkeletonText lines={5} />)
    const skeletons = screen.getAllByRole('generic').filter((el: HTMLElement) =>
      el.classList.contains('animate-pulse')
    )
    expect(skeletons.length).toBe(5)
  })

  it('last line has shorter width', () => {
    render(<SkeletonText lines={3} />)
    const skeletons = screen.getAllByRole('generic').filter((el: HTMLElement) =>
      el.classList.contains('animate-pulse')
    )
    expect(skeletons[2]).toHaveClass('w-3/4') // Last line should be shorter
  })
})
