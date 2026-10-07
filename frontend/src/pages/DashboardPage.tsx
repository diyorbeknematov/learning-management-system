import { useQuery } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { Link } from 'react-router'
import { api, call } from '@/api/client'
import { useMyEnrollments } from '@/api/queries'
import { useAuth } from '@/auth/context'
import { CourseCover } from '@/components/CourseCard'
import { ErrorBlock, LoadingBlock } from '@/components/States'
import { buttonVariants } from '@/components/ui/button'
import { Card, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Progress } from '@/components/ui/progress'
import { fullName, moneyExact } from '@/lib/format'

function Figure({ label, value, to }: { label: string; value: ReactNode; to?: string }) {
  const body = (
    <Card className={to ? 'transition hover:shadow-md' : undefined}>
      <CardHeader>
        <CardDescription>{label}</CardDescription>
        <CardTitle className="text-2xl">{value}</CardTitle>
      </CardHeader>
    </Card>
  )

  return to ? (
    <Link to={to} className="rounded-xl outline-none focus-visible:ring-3 focus-visible:ring-ring/50">
      {body}
    </Link>
  ) : (
    body
  )
}

function StudentDashboard() {
  const enrollments = useMyEnrollments()
  const certificates = useQuery({ queryKey: ['certificates', 'me'], queryFn: () => call(api.GET('/certificates/me')) })

  if (enrollments.isPending) return <LoadingBlock rows={2} />
  if (enrollments.isError) return <ErrorBlock error={enrollments.error} />

  const items = enrollments.data.items?.filter((item) => item.status !== 'dropped') ?? []
  const learning = items.filter((item) => item.status === 'active')

  return (
    <>
      <div className="grid gap-4 sm:grid-cols-3">
        <Figure label="Courses you study" value={learning.length} to="/my-courses" />
        <Figure label="Finished courses" value={items.length - learning.length} to="/my-courses" />
        <Figure label="Certificates" value={certificates.data?.length ?? '…'} to="/certificates" />
      </div>

      <section className="space-y-3">
        <h2 className="text-lg font-semibold">Continue learning</h2>

        {learning.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            You are not studying anything now.{' '}
            <Link to="/" className="underline">
              Find a course
            </Link>
          </p>
        ) : (
          <div className="grid gap-4 sm:grid-cols-2">
            {learning.slice(0, 4).map((item) => (
              <Card key={item.id} className="overflow-hidden sm:flex-row">
                <CourseCover url={item.course_cover_url} title={item.course_title} className="h-28 w-full object-cover sm:h-auto sm:w-32" />
                <div className="flex flex-1 flex-col gap-2 px-4 pb-4 sm:pt-4">
                  <h3 className="font-medium leading-snug">{item.course_title}</h3>
                  <Progress value={item.progress_percent ?? 0} aria-label="Progress" />
                  <Link to={`/learn/${item.course_id}`} className={buttonVariants({ size: 'sm', variant: 'outline' }) + ' mt-auto w-fit'}>
                    Continue ({Math.round(item.progress_percent ?? 0)}%)
                  </Link>
                </div>
              </Card>
            ))}
          </div>
        )}
      </section>
    </>
  )
}

function InstructorDashboard() {
  const { user } = useAuth()

  const courses = useQuery({
    queryKey: ['courses', 'teach', user?.id, 'dashboard'],
    queryFn: () => call(api.GET('/courses', { params: { query: { instructor_id: user?.id, limit: 100 } } })),
  })

  if (courses.isPending) return <LoadingBlock rows={2} />
  if (courses.isError) return <ErrorBlock error={courses.error} />

  const items = courses.data.items ?? []
  const published = items.filter((course) => course.status === 'published')
  const students = items.reduce((sum, course) => sum + (course.enrollment_count ?? 0), 0)

  return (
    <>
      <div className="grid gap-4 sm:grid-cols-3">
        <Figure label="Published courses" value={published.length} to="/teach/courses" />
        <Figure label="Drafts" value={items.length - published.length} to="/teach/courses" />
        <Figure label="Students in total" value={students} />
      </div>

      <section className="space-y-3">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold">Your courses</h2>
          <Link to="/teach/courses/new" className={buttonVariants({ size: 'sm' })}>
            New course
          </Link>
        </div>

        <ul className="divide-y rounded-lg border">
          {items.slice(0, 6).map((course) => (
            <li key={course.id} className="flex items-center gap-3 px-4 py-2 text-sm">
              <Link to={`/teach/courses/${course.id}`} className="flex-1 font-medium hover:underline">
                {course.title}
              </Link>
              <span className="text-muted-foreground">{course.enrollment_count ?? 0} students</span>
              <span className="w-16 text-right text-muted-foreground">{course.status}</span>
            </li>
          ))}
          {items.length === 0 && <li className="px-4 py-3 text-sm text-muted-foreground">You have no courses yet.</li>}
        </ul>
      </section>
    </>
  )
}

function AdminDashboard() {
  const users = useQuery({ queryKey: ['users', 'count'], queryFn: () => call(api.GET('/users', { params: { query: { limit: 1 } } })) })
  const courses = useQuery({ queryKey: ['courses', 'count'], queryFn: () => call(api.GET('/courses', { params: { query: { limit: 1 } } })) })
  const payments = useQuery({ queryKey: ['payments', 'count'], queryFn: () => call(api.GET('/payments', { params: { query: { limit: 1 } } })) })
  const finance = useQuery({ queryKey: ['finance', 'all'], queryFn: () => call(api.GET('/finance', { params: { query: { group_by: 'month' } } })) })

  const failed = [users, courses, payments, finance].find((query) => query.isError)

  if (failed) return <ErrorBlock error={failed.error} />

  return (
    <div className="space-y-4">
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Figure label="Users" value={users.data?.total ?? '…'} to="/admin/users" />
        <Figure label="Courses" value={courses.data?.total ?? '…'} to="/teach/courses" />
        <Figure label="Payments" value={payments.data?.total ?? '…'} to="/admin/payments" />
        <Figure label="Revenue" value={finance.data ? moneyExact(finance.data.revenue) : '…'} to="/admin/finance" />
      </div>
      <div className="grid gap-4 sm:grid-cols-2">
        <Figure label="Paid to instructors" value={finance.data ? moneyExact(finance.data.expenses) : '…'} to="/admin/finance" />
        <Figure label="Net profit" value={finance.data ? moneyExact(finance.data.net_profit) : '…'} to="/admin/finance" />
      </div>
    </div>
  )
}

// The page a user lands on after logging in: what matters for their role.
export default function DashboardPage() {
  const { user } = useAuth()

  return (
    <div className="space-y-8">
      <div className="space-y-1">
        <h1 className="text-2xl font-semibold">Hello, {fullName(user)}</h1>
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
