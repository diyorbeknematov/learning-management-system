import { courseGrid } from '@/components/layout/grids'
import { useQuery } from '@tanstack/react-query'
import { Search, SlidersHorizontal, X } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useSearchParams } from 'react-router'
import { api, call } from '@/api/client'
import { useCategories } from '@/api/queries'
import { CourseCard } from '@/components/CourseCard'
import { NativeSelect } from '@/components/NativeSelect'
import { PageHeader } from '@/components/PageHeader'
import { Pagination } from '@/components/Pagination'
import { Empty, ErrorBlock } from '@/components/States'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { plural } from '@/lib/format'

// a rounded, compact select for the filters
const pill = 'h-9 w-auto min-w-36 rounded-full bg-background pr-8'

const LIMIT = 20

const sorts = [
  { value: 'newest', label: 'Newest' },
  { value: 'popular', label: 'Most popular' },
  { value: 'rating', label: 'Best rated' },
  { value: 'price', label: 'Price: low to high' },
]

type Difficulty = 'beginner' | 'intermediate' | 'advanced'
type PriceType = 'free' | 'paid'

// The catalog. Every filter lives in the address (?q=go&sort=price), so a
// search can be shared and the back button works.
export default function CatalogPage() {
  const [params, setParams] = useSearchParams()
  const categories = useCategories()

  const q = params.get('q') ?? ''
  const categoryId = params.get('category') ?? ''
  const difficulty = (params.get('difficulty') ?? '') as Difficulty | ''
  const priceType = (params.get('price') ?? '') as PriceType | ''
  const minRating = params.get('rating') ?? ''
  const sort = params.get('sort') ?? 'newest'
  const page = Math.max(1, Number(params.get('page')) || 1)

  // the search box is typed into; the address is updated after a short pause
  const [typed, setTyped] = useState(q)

  useEffect(() => setTyped(q), [q])

  useEffect(() => {
    if (typed === q) return

    const timer = setTimeout(() => change({ q: typed }), 350)

    return () => clearTimeout(timer)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [typed])

  function change(next: Record<string, string>) {
    const updated = new URLSearchParams(params)

    for (const [key, value] of Object.entries(next)) {
      if (value) updated.set(key, value)
      else updated.delete(key)
    }

    // a new filter starts from the first page
    if (!('page' in next)) updated.delete('page')

    setParams(updated, { replace: true })
  }

  const courses = useQuery({
    queryKey: ['courses', { q, categoryId, difficulty, priceType, minRating, sort, page }],
    queryFn: () =>
      call(
        api.GET('/courses', {
          params: {
            query: {
              q: q || undefined,
              category_id: categoryId || undefined,
              status: 'published',
              difficulty: difficulty || undefined,
              price_type: priceType || undefined,
              min_rating: minRating ? Number(minRating) : undefined,
              sort: sort as 'newest',
              page,
              limit: LIMIT,
            },
          },
        }),
      ),
    placeholderData: (previous) => previous,
  })

  const filtered = Boolean(q || categoryId || difficulty || priceType || minRating)

  return (
    <div className="space-y-6">
      <PageHeader title="Courses" description="Learn something new from our instructors." />

      <div className="space-y-4">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div className="relative w-full max-w-md">
            <Search className="pointer-events-none absolute left-4 top-3 size-4 text-muted-foreground" />
            <Input
              value={typed}
              onChange={(e) => setTyped(e.target.value)}
              placeholder="Search courses"
              aria-label="Search courses"
              className="h-10 rounded-full pl-10"
            />
          </div>

          <label className="flex items-center gap-2 text-sm text-muted-foreground">
            Sort by
            <NativeSelect aria-label="Sort by" className={pill} value={sort} onChange={(e) => change({ sort: e.target.value })}>
              {sorts.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </NativeSelect>
          </label>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <span className="mr-1 inline-flex items-center gap-1.5 text-sm font-medium">
            <SlidersHorizontal className="size-4" /> Filters
          </span>

          <NativeSelect aria-label="Category" className={pill} value={categoryId} onChange={(e) => change({ category: e.target.value })}>
            <option value="">All categories</option>
            {categories.data?.items?.map((category) => (
              <option key={category.id} value={category.id}>
                {category.name}
              </option>
            ))}
          </NativeSelect>

          <NativeSelect aria-label="Level" className={pill} value={difficulty} onChange={(e) => change({ difficulty: e.target.value })}>
            <option value="">Any level</option>
            <option value="beginner">Beginner</option>
            <option value="intermediate">Intermediate</option>
            <option value="advanced">Advanced</option>
          </NativeSelect>

          <NativeSelect aria-label="Price" className={pill} value={priceType} onChange={(e) => change({ price: e.target.value })}>
            <option value="">Free and paid</option>
            <option value="free">Free</option>
            <option value="paid">Paid</option>
          </NativeSelect>

          <NativeSelect aria-label="Rating" className={pill} value={minRating} onChange={(e) => change({ rating: e.target.value })}>
            <option value="">Any rating</option>
            <option value="4">4 stars and up</option>
            <option value="3">3 stars and up</option>
          </NativeSelect>

          {filtered && (
            <Button variant="ghost" size="sm" className="rounded-full" onClick={() => setParams({}, { replace: true })}>
              <X /> Clear
            </Button>
          )}

          <p className="ml-auto text-sm text-muted-foreground" aria-live="polite">
            {courses.data ? plural(courses.data.total, 'course') : ' '}
          </p>
        </div>
      </div>

      {courses.isPending && (
        <div className={courseGrid}>
          {Array.from({ length: 6 }, (_, i) => (
            <Skeleton key={i} className="h-72" />
          ))}
        </div>
      )}

      {courses.isError && <ErrorBlock error={courses.error} />}

      {courses.data && courses.data.items?.length === 0 && (
        <Empty title={filtered ? 'No course matches your search' : 'No published courses yet'}>{filtered && 'Try other words or clear the filters.'}</Empty>
      )}

      {courses.data && (courses.data.items?.length ?? 0) > 0 && (
        <div className={courseGrid}>
          {courses.data.items?.map((course) => (
            <CourseCard key={course.id} course={course} />
          ))}
        </div>
      )}

      {courses.data && <Pagination page={page} limit={LIMIT} total={courses.data.total ?? 0} onPage={(next) => change({ page: String(next) })} />}
    </div>
  )
}
