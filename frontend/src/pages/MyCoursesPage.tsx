import { courseGrid } from '@/components/layout/grids'
import { Link } from 'react-router'
import { useMyEnrollments } from '@/api/queries'
import { CourseCover } from '@/components/CourseCard'
import { PageHeader } from '@/components/PageHeader'
import { Empty, ErrorBlock, LoadingBlock } from '@/components/States'
import { Badge } from '@/components/ui/badge'
import { buttonVariants } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Progress } from '@/components/ui/progress'

export default function MyCoursesPage() {
  const enrollments = useMyEnrollments()

  return (
    <div className="space-y-6">
      <PageHeader
        title="My courses"
        description="Everything you study, with your progress."
        actions={
          <Link to="/courses" className={buttonVariants({ variant: 'outline' })}>
            Find more courses
          </Link>
        }
      />

      {enrollments.isPending && <LoadingBlock />}
      {enrollments.isError && <ErrorBlock error={enrollments.error} />}

      {enrollments.data?.items?.length === 0 && (
        <Empty title="You have not enrolled in any course yet">
          <Link to="/courses" className="underline">
            Find a course
          </Link>
        </Empty>
      )}

      <div className={courseGrid}>
        {enrollments.data?.items
          ?.filter((item) => item.status !== 'dropped')
          .map((item) => (
            <Card key={item.id} className="overflow-hidden pt-0">
              <CourseCover url={item.course_cover_url} title={item.course_title} className="h-32 w-full object-cover" />
              <div className="flex flex-1 flex-col gap-3 px-4 pb-4">
                <div className="flex items-start justify-between gap-2">
                  <h2 className="font-medium leading-snug">{item.course_title}</h2>
                  {item.status === 'completed' && <Badge>Completed</Badge>}
                </div>
                <Progress value={item.progress_percent ?? 0} aria-label="Progress" />
                <p className="text-xs text-muted-foreground">
                  {item.completed_lessons ?? 0} of {item.total_lessons ?? 0} lessons · {Math.round(item.progress_percent ?? 0)}%
                </p>
                <Link to={`/learn/${item.course_id}`} className={buttonVariants({ variant: 'outline' }) + ' mt-auto'}>
                  {item.status === 'completed' ? 'Open' : 'Continue'}
                </Link>
              </div>
            </Card>
          ))}
      </div>
    </div>
  )
}
