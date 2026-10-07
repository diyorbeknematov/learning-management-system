import { useQuery } from '@tanstack/react-query'
import { Award, BookOpen, CheckCircle2, CircleDollarSign, Coins, FileEdit, GraduationCap, Star, TrendingUp, Users, Wallet } from 'lucide-react'
import { useState } from 'react'
import { Link } from 'react-router'
import { api, call } from '@/api/client'
import { useMyEnrollments } from '@/api/queries'
import { useAuth } from '@/auth/context'
import { CourseCard, CourseCover } from '@/components/CourseCard'
import { Carousel } from '@/components/Carousel'
import { DateRange, RangePresets } from '@/components/DateRange'
import { TableCard } from '@/components/Panels'
import { StatCard } from '@/components/StatCard'
import { StatusBadge } from '@/components/StatusBadge'
import { ErrorBlock, LoadingBlock } from '@/components/States'
import { buttonVariants } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Progress } from '@/components/ui/progress'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { formatDate, fullName, moneyExact, plural } from '@/lib/format'

function Section({ title, link, children }: { title: string; link?: { to: string; label: string }; children: React.ReactNode }) {
  return (
    <section className="space-y-4" aria-label={title}>
      <div className="flex items-end justify-between gap-3">
        <h2 className="text-xl font-semibold">{title}</h2>
        {link && (
          <Link to={link.to} className="text-sm font-medium text-primary hover:underline">
            {link.label}
          </Link>
        )}
      </div>
      {children}
    </section>
  )
}

function Popular({ skip, title = 'Popular courses' }: { skip?: Set<string>; title?: string }) {
  const courses = useQuery({
    queryKey: ['courses', 'dashboard', 'popular'],
    queryFn: () => call(api.GET('/courses', { params: { query: { status: 'published', sort: 'popular', limit: 14 } } })),
  })

  const items = (courses.data?.items ?? []).filter((course) => !skip?.has(course.id!)).slice(0, 12)

  if (courses.isPending || items.length === 0) return null

  return (
    <Section title={title} link={{ to: '/courses', label: 'All courses' }}>
      <Carousel label={title}>
        {items.map((course) => (
          <CourseCard key={course.id} course={course} />
        ))}
      </Carousel>
    </Section>
  )
}

function StudentDashboard() {
  const enrollments = useMyEnrollments()
  const certificates = useQuery({ queryKey: ['certificates', 'me'], queryFn: () => call(api.GET('/certificates/me')) })

  if (enrollments.isPending) return <LoadingBlock rows={2} />
  if (enrollments.isError) return <ErrorBlock error={enrollments.error} />

  const items = enrollments.data.items?.filter((item) => item.status !== 'dropped') ?? []
  const learning = items.filter((item) => item.status === 'active')
  const lessonsDone = items.reduce((sum, item) => sum + (item.completed_lessons ?? 0), 0)

  return (
    <>
      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <StatCard label="Courses you study" value={learning.length} icon={BookOpen} to="/my-courses" />
        <StatCard label="Finished courses" value={items.length - learning.length} icon={CheckCircle2} tone="green" to="/my-courses" />
        <StatCard label="Lessons done" value={lessonsDone} icon={GraduationCap} tone="sky" />
        <StatCard label="Certificates" value={certificates.data?.length ?? '…'} icon={Award} tone="amber" to="/certificates" />
      </div>

      <Section title="Continue learning" link={items.length > 0 ? { to: '/my-courses', label: 'All my courses' } : undefined}>
        {learning.length === 0 ? (
          <Card className="items-center gap-3 border-dashed p-10 text-center">
            <p className="font-medium">You are not studying anything now</p>
            <Link to="/courses" className={buttonVariants()}>
              Find a course
            </Link>
          </Card>
        ) : (
          <div className="grid gap-5 md:grid-cols-2 2xl:grid-cols-3">
            {learning.slice(0, 6).map((item) => (
              <Card key={item.id} className="flex-row gap-0 overflow-hidden py-0">
                <CourseCover url={item.course_cover_url} title={item.course_title} className="w-32 shrink-0 self-stretch object-cover sm:w-40" />
                <div className="flex min-w-0 flex-1 flex-col gap-3 p-4">
                  <h3 className="line-clamp-2 font-medium leading-snug">{item.course_title}</h3>
                  <div className="space-y-1">
                    <Progress value={item.progress_percent ?? 0} aria-label="Progress" />
                    <p className="text-xs text-muted-foreground">
                      {item.completed_lessons ?? 0} of {plural(item.total_lessons, 'lesson')} · {Math.round(item.progress_percent ?? 0)}%
                    </p>
                  </div>
                  <Link to={`/learn/${item.course_id}`} className={buttonVariants({ size: 'sm' }) + ' mt-auto w-fit'}>
                    Continue
                  </Link>
                </div>
              </Card>
            ))}
          </div>
        )}
      </Section>

      <Popular title="Recommended for you" skip={new Set(items.map((item) => item.course_id!))} />
    </>
  )
}

function InstructorDashboard() {
  const { user } = useAuth()

  const courses = useQuery({
    queryKey: ['courses', 'teach', user?.id, 'dashboard'],
    queryFn: () => call(api.GET('/courses', { params: { query: { instructor_id: user?.id, limit: 80 } } })),
  })

  if (courses.isPending) return <LoadingBlock rows={2} />
  if (courses.isError) return <ErrorBlock error={courses.error} />

  const items = courses.data.items ?? []
  const published = items.filter((course) => course.status === 'published')
  const students = items.reduce((sum, course) => sum + (course.enrollment_count ?? 0), 0)
  const rated = published.filter((course) => (course.review_count ?? 0) > 0)
  const rating = rated.length ? rated.reduce((sum, course) => sum + (course.avg_rating ?? 0), 0) / rated.length : 0

  return (
    <>
      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <StatCard label="Published courses" value={published.length} icon={BookOpen} to="/teach/courses" />
        <StatCard label="Drafts" value={items.length - published.length} icon={FileEdit} tone="amber" to="/teach/courses" />
        <StatCard label="Students in total" value={students} icon={Users} tone="green" />
        <StatCard label="Average rating" value={rating ? rating.toFixed(1) : 'No reviews yet'} icon={Star} tone="rose" />
      </div>

      <Section title="Your published courses" link={{ to: '/teach/courses', label: 'Manage all' }}>
        {published.length === 0 ? (
          <Card className="items-center gap-3 border-dashed p-10 text-center">
            <p className="font-medium">You have no published courses yet</p>
            <Link to="/teach/courses/new" className={buttonVariants()}>
              Create a course
            </Link>
          </Card>
        ) : (
          <Carousel label="Your published courses">
            {published.slice(0, 20).map((course) => (
              <div key={course.id} className="relative">
                <CourseCard course={course} />
                <Link to={`/teach/courses/${course.id}`} className={buttonVariants({ size: 'sm', variant: 'secondary' }) + ' absolute right-3 top-3'}>
                  Edit
                </Link>
              </div>
            ))}
          </Carousel>
        )}
      </Section>
    </>
  )
}

function AdminDashboard() {
  const users = useQuery({ queryKey: ['users', 'count'], queryFn: () => call(api.GET('/users', { params: { query: { limit: 1 } } })) })
  const courses = useQuery({
    queryKey: ['courses', 'count'],
    queryFn: () => call(api.GET('/courses', { params: { query: { status: 'published', limit: 1 } } })),
  })
  const [range, setRange] = useState({ from: '', to: '' })
  const payments = useQuery({
    queryKey: ['payments', 'recent', range],
    queryFn: () => call(api.GET('/payments', { params: { query: { from: range.from || undefined, to: range.to || undefined, limit: 6 } } })),
    placeholderData: (previous) => previous,
  })
  const finance = useQuery({ queryKey: ['finance', 'all'], queryFn: () => call(api.GET('/finance', { params: { query: { group_by: 'month' } } })) })

  const failed = [users, courses, payments, finance].find((query) => query.isError)

  if (failed) return <ErrorBlock error={failed.error} />

  return (
    <>
      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-6">
        <StatCard label="Users" value={users.data?.total ?? '…'} icon={Users} to="/admin/users" />
        <StatCard label="Courses" value={courses.data?.total ?? '…'} icon={BookOpen} tone="sky" to="/teach/courses" />
        <StatCard label="Payments" value={payments.data?.total ?? '…'} icon={Wallet} tone="amber" to="/admin/payments" />
        <StatCard label="Revenue" value={finance.data ? moneyExact(finance.data.revenue) : '…'} icon={CircleDollarSign} tone="green" to="/admin/finance" />
        <StatCard label="Paid to instructors" value={finance.data ? moneyExact(finance.data.expenses) : '…'} icon={Coins} tone="rose" to="/admin/finance" />
        <StatCard label="Net profit" value={finance.data ? moneyExact(finance.data.net_profit) : '…'} icon={TrendingUp} to="/admin/finance" />
      </div>

      <section className="space-y-4" aria-label="Latest payments">
        <div className="flex flex-wrap items-end justify-between gap-4">
          <h2 className="text-xl font-semibold">Latest payments</h2>
          <div className="flex flex-wrap items-end gap-3">
            <div className="grid w-80 grid-cols-2 gap-3">
              <DateRange from={range.from} to={range.to} onChange={setRange} />
            </div>
            <Link to="/admin/payments" className="pb-2 text-sm font-medium text-primary hover:underline">
              All payments
            </Link>
          </div>
        </div>
        <RangePresets from={range.from} to={range.to} onChange={setRange} />

        {payments.data?.items?.length === 0 ? (
          <p className="rounded-xl border border-dashed p-8 text-center text-sm text-muted-foreground">No payments in this period.</p>
        ) : (
          <TableCard>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Paid</TableHead>
                  <TableHead>Enrollment</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead className="text-right">Amount</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {payments.data?.items?.map((payment) => (
                  <TableRow key={payment.id}>
                    <TableCell>{formatDate(payment.paid_at)}</TableCell>
                    <TableCell className="font-mono text-xs text-muted-foreground">{payment.enrollment_id?.slice(0, 8)}</TableCell>
                    <TableCell>
                      <StatusBadge value={payment.status} />
                    </TableCell>
                    <TableCell className="text-right font-semibold">{moneyExact(payment.amount)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableCard>
        )}
      </section>

      <Popular />
    </>
  )
}

// The page a user lands on after logging in: what matters for their role.
export default function DashboardPage() {
  const { user } = useAuth()

  return (
    <div className="space-y-10">
      <div className="space-y-1">
        <h1 className="text-3xl font-bold tracking-tight">Hello, {fullName(user)}</h1>
        <p className="text-muted-foreground">
          {user?.role_name === 'Student' && 'Here is where you stopped.'}
          {user?.role_name === 'Instructor' && 'Here is how your courses do.'}
          {user?.role_name === 'SuperAdmin' && 'Here is the platform at a glance.'}
        </p>
      </div>

      {user?.role_name === 'Student' && <StudentDashboard />}
      {user?.role_name === 'Instructor' && <InstructorDashboard />}
      {user?.role_name === 'SuperAdmin' && <AdminDashboard />}
    </div>
  )
}
