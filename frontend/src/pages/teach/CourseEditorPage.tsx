import { useQuery } from '@tanstack/react-query'
import { Eye } from 'lucide-react'
import { Link, useParams, useSearchParams } from 'react-router'
import { toast } from 'sonner'
import { api, call } from '@/api/client'
import { ErrorBlock, LoadingBlock } from '@/components/States'
import { Badge } from '@/components/ui/badge'
import { Button, buttonVariants } from '@/components/ui/button'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useApiMutation } from '@/lib/mutations'
import { ContentTab } from './ContentTab'
import { CourseDetailsForm } from './CourseDetailsForm'
import { QuizzesTab } from './QuizzesTab'
import { StudentsTab } from './StudentsTab'

const tabs = ['details', 'content', 'quizzes', 'students'] as const

export default function CourseEditorPage() {
  const { courseId = '' } = useParams()
  const [params, setParams] = useSearchParams()
  const tab = tabs.find((name) => name === params.get('tab')) ?? 'details'

  const course = useQuery({
    queryKey: ['course', courseId],
    queryFn: () => call(api.GET('/courses/{courseId}', { params: { path: { courseId } } })),
  })

  const status = useApiMutation(
    (next: 'draft' | 'published') => call(api.PATCH('/courses/{courseId}/status', { params: { path: { courseId } }, body: { status: next } })),
    {
      invalidate: [['course', courseId], ['courses']],
      onSuccess: (_, next) => toast.success(next === 'published' ? 'The course is published' : 'The course is a draft again'),
    },
  )

  if (course.isPending) return <LoadingBlock rows={4} />
  if (course.isError) return <ErrorBlock error={course.error} />

  const published = course.data.status === 'published'

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center gap-3">
        <h1 className="min-w-0 flex-1 text-2xl font-semibold">{course.data.title}</h1>
        <Badge variant={published ? 'default' : 'outline'}>{published ? 'Published' : 'Draft'}</Badge>
        <Link to={`/teach/courses/${courseId}/preview`} className={buttonVariants({ variant: 'outline' })}>
          <Eye /> View as a student
        </Link>
        <Button disabled={status.isPending} variant={published ? 'outline' : 'default'} onClick={() => status.mutate(published ? 'draft' : 'published')}>
          {published ? 'Unpublish' : 'Publish'}
        </Button>
      </div>

      <Tabs value={tab} onValueChange={(value) => setParams({ tab: String(value) }, { replace: true })}>
        <TabsList>
          <TabsTrigger value="details">Details</TabsTrigger>
          <TabsTrigger value="content">Content</TabsTrigger>
          <TabsTrigger value="quizzes">Quizzes</TabsTrigger>
          <TabsTrigger value="students">Students</TabsTrigger>
        </TabsList>

        <TabsContent value="details" className="pt-4">
          <CourseDetailsForm course={course.data} />
        </TabsContent>
        <TabsContent value="content" className="pt-4">
          <ContentTab courseId={courseId} />
        </TabsContent>
        <TabsContent value="quizzes" className="pt-4">
          <QuizzesTab courseId={courseId} />
        </TabsContent>
        <TabsContent value="students" className="pt-4">
          <StudentsTab courseId={courseId} />
        </TabsContent>
      </Tabs>
    </div>
  )
}
