import { courseGrid } from '@/components/layout/grids'
import { useQueries, useQuery } from '@tanstack/react-query'
import { ArrowRight, Award, BookOpen, Search, Sparkles, UserPlus } from 'lucide-react'
import { useState } from 'react'
import { Link, useNavigate } from 'react-router'
import { api, call } from '@/api/client'
import { useCategories } from '@/api/queries'
import { useAuth } from '@/auth/context'
import { Carousel } from '@/components/Carousel'
import { CourseCard } from '@/components/CourseCard'
import { ErrorBlock } from '@/components/States'
import { buttonVariants } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { container } from '@/components/layout/container'
import { Skeleton } from '@/components/ui/skeleton'
import { plural } from '@/lib/format'

type Sort = 'popular' | 'rating' | 'newest'

// a course that this many students joined is called a bestseller
const BESTSELLER = 3

function CourseRow({ title, sort, minRating, link, badge }: { title: string; sort: Sort; minRating?: number; link: string; badge?: 'bestseller' | 'new' }) {
  const courses = useQuery({
    queryKey: ['courses', 'home', sort],
    queryFn: () => call(api.GET('/courses', { params: { query: { status: 'published', sort, min_rating: minRating, limit: 12 } } })),
  })

  if (courses.isError) return <ErrorBlock error={courses.error} />

  // nothing to show: the section is left out
  if (courses.data && courses.data.items?.length === 0) return null

  const labelOf = (course: { enrollment_count?: number }) =>
    badge === 'bestseller' ? ((course.enrollment_count ?? 0) >= BESTSELLER ? 'Bestseller' : undefined) : badge === 'new' ? 'New' : undefined

  return (
    <section className={`${container} space-y-4`} aria-label={title}>
      <div className="flex items-end justify-between gap-3">
        <h2 className="text-xl font-semibold">{title}</h2>
        <Link to={link} className="inline-flex items-center gap-1 text-sm font-medium text-primary hover:underline">
          View all <ArrowRight className="size-4" />
        </Link>
      </div>

      {courses.isPending ? (
        <div className={courseGrid}>
          {Array.from({ length: 4 }, (_, i) => (
            <Skeleton key={i} className="h-72" />
          ))}
        </div>
      ) : (
        <Carousel label={title}>
          {courses.data?.items?.map((course) => (
            <CourseCard key={course.id} course={course} label={labelOf(course)} />
          ))}
        </Carousel>
      )}
    </section>
  )
}

function Categories() {
  const categories = useCategories()
  const shown = (categories.data?.items ?? []).slice(0, 8)

  // how many courses each category has: one small request for each
  const counts = useQueries({
    queries: shown.map((category) => ({
      queryKey: ['courses', 'count', category.id],
      queryFn: () => call(api.GET('/courses', { params: { query: { status: 'published', category_id: category.id, limit: 1 } } })),
    })),
  })

  if (shown.length === 0) return null

  return (
    <section className={`${container} space-y-4`} aria-label="Categories">
      <h2 className="text-xl font-semibold">Browse by category</h2>
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {shown.map((category, index) => (
          <Link
            key={category.id}
            to={`/courses?category=${category.id}`}
            className="group flex items-center gap-3 rounded-xl border p-4 outline-none transition hover:border-primary hover:shadow-sm focus-visible:ring-3 focus-visible:ring-ring/50"
          >
            <span className="flex size-10 items-center justify-center rounded-lg bg-primary/10 text-primary">
              <BookOpen className="size-5" />
            </span>
            <span className="min-w-0">
              <span className="block truncate font-medium group-hover:text-primary">{category.name}</span>
              <span className="text-xs text-muted-foreground">{counts[index]?.data ? plural(counts[index].data.total, 'course') : '…'}</span>
            </span>
          </Link>
        ))}
      </div>
    </section>
  )
}

const steps = [
  { icon: Search, title: 'Find a course', text: 'Search the catalog or browse by category. Free previews let you look inside first.' },
  { icon: BookOpen, title: 'Learn at your pace', text: 'Lessons, materials and quizzes. Your progress is saved on every device.' },
  { icon: Award, title: 'Get a certificate', text: 'Finish the lessons and pass the final quiz: a PDF certificate anybody can check.' },
]

export default function HomePage() {
  const { user } = useAuth()
  const navigate = useNavigate()
  const [query, setQuery] = useState('')

  const totals = useQuery({
    queryKey: ['courses', 'count', 'all'],
    queryFn: () => call(api.GET('/courses', { params: { query: { status: 'published', limit: 1 } } })),
  })
  const categories = useCategories()

  return (
    <div className="space-y-16 pb-16">
      <section className="bg-gradient-to-b from-primary/15 via-primary/5 to-background py-16 text-center sm:py-24">
        <div className={container}>
          <p className="mb-4 inline-flex items-center gap-2 rounded-full border bg-card px-3 py-1 text-xs font-medium text-primary">
            <Sparkles className="size-3.5" /> Learn new skills online
          </p>
          <h1 className="mx-auto max-w-2xl text-4xl font-bold leading-tight tracking-tight sm:text-5xl">Learn something new, one lesson at a time</h1>
          <p className="mx-auto mt-4 max-w-xl text-lg text-muted-foreground">
            Courses from instructors you can trust, quizzes that check what you learned, and certificates you can show.
          </p>

          <form
            className="mx-auto mt-8 flex max-w-xl gap-2"
            onSubmit={(e) => {
              e.preventDefault()
              navigate(query.trim() ? `/courses?q=${encodeURIComponent(query.trim())}` : '/courses')
            }}
          >
            <div className="relative flex-1">
              <Search className="pointer-events-none absolute left-3 top-3 size-4 text-muted-foreground" />
              <Input
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                placeholder="What do you want to learn?"
                aria-label="Search courses"
                className="h-10 bg-background pl-9"
              />
            </div>
            <button type="submit" className={buttonVariants({ size: 'lg' }) + ' h-10'}>
              Search
            </button>
          </form>

          <div className="mt-8 flex flex-wrap justify-center gap-x-10 gap-y-2 text-sm text-muted-foreground">
            <span>
              <strong className="text-foreground">{totals.data?.total ?? '…'}</strong> courses
            </span>
            <span>
              <strong className="text-foreground">{categories.data?.total ?? '…'}</strong> categories
            </span>
            <span>
              <strong className="text-foreground">Free</strong> previews
            </span>
          </div>
        </div>
      </section>

      <Categories />

      <CourseRow title="Bestsellers" sort="popular" badge="bestseller" link="/courses?sort=popular" />
      <CourseRow title="Best rated" sort="rating" minRating={4} link="/courses?sort=rating&rating=4" />
      <CourseRow title="Newest courses" sort="newest" badge="new" link="/courses?sort=newest" />

      <section className={`${container} space-y-6`} aria-label="How it works">
        <h2 className="text-center text-xl font-semibold">How it works</h2>
        <div className="grid gap-4 md:grid-cols-3">
          {steps.map(({ icon: Icon, title, text }, index) => (
            <div key={title} className="space-y-2 rounded-xl border p-5">
              <span className="flex size-10 items-center justify-center rounded-lg bg-primary text-primary-foreground">
                <Icon className="size-5" />
              </span>
              <h3 className="font-medium">
                {index + 1}. {title}
              </h3>
              <p className="text-sm text-muted-foreground">{text}</p>
            </div>
          ))}
        </div>
      </section>

      <section className="bg-primary py-16 text-primary-foreground">
        <div className={`${container} flex flex-col items-center gap-4 text-center`}>
          <h2 className="text-2xl font-semibold">{user ? 'Pick up where you stopped' : 'Ready to start?'}</h2>
          <p className="max-w-md opacity-90">
            {user ? 'Your courses and progress are on your dashboard.' : 'Create a free account and enroll in your first course in a minute.'}
          </p>
          {user ? (
            <Link to="/dashboard" className={buttonVariants({ variant: 'secondary', size: 'lg' })}>
              Go to the dashboard <ArrowRight />
            </Link>
          ) : (
            <Link to="/register" className={buttonVariants({ variant: 'secondary', size: 'lg' })}>
              <UserPlus /> Create an account
            </Link>
          )}
        </div>
      </section>
    </div>
  )
}
