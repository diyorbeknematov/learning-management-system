import { courseGrid } from '@/components/layout/grids'
import { useQuery } from '@tanstack/react-query'
import { Plus } from 'lucide-react'
import { useState } from 'react'
import { Link } from 'react-router'
import { api, call } from '@/api/client'
import { useAuth } from '@/auth/context'
import { CourseCard } from '@/components/CourseCard'
import { PageHeader } from '@/components/PageHeader'
import { plural } from '@/lib/format'
import { Pagination } from '@/components/Pagination'
import { Empty, ErrorBlock, LoadingBlock } from '@/components/States'
import { buttonVariants } from '@/components/ui/button'

const LIMIT = 20

// An instructor sees their own courses, drafts too; a SuperAdmin sees all courses.
export default function TeachCoursesPage() {
  const { user } = useAuth()
  const [page, setPage] = useState(1)
  const isAdmin = user?.role_name === 'SuperAdmin'

  const courses = useQuery({
    queryKey: ['courses', 'teach', user?.id, page],
    queryFn: () => call(api.GET('/courses', { params: { query: { instructor_id: isAdmin ? undefined : user?.id, sort: 'newest', page, limit: LIMIT } } })),
    placeholderData: (previous) => previous,
  })

  return (
    <div className="space-y-6">
      <PageHeader
        title={isAdmin ? 'All courses' : 'Courses you teach'}
        description={courses.data ? `${plural(courses.data.total, 'course')}. Drafts are visible only to you.` : undefined}
        actions={
          <Link to="/teach/courses/new" className={buttonVariants()}>
            <Plus /> New course
          </Link>
        }
      />

      {courses.isPending && <LoadingBlock />}
      {courses.isError && <ErrorBlock error={courses.error} />}
      {courses.data?.items?.length === 0 && <Empty title="No courses yet">Create the first one with the button above.</Empty>}

      <div className={courseGrid}>
        {courses.data?.items?.map((course) => (
          <div key={course.id} className="relative">
            <CourseCard course={course} />
            <Link to={`/teach/courses/${course.id}`} className={buttonVariants({ size: 'sm', variant: 'secondary' }) + ' absolute right-3 top-3'}>
              Edit
            </Link>
          </div>
        ))}
      </div>

      <Pagination page={page} limit={LIMIT} total={courses.data?.total ?? 0} onPage={setPage} />
    </div>
  )
}
