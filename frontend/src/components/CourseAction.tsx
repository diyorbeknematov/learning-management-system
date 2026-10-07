import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate } from 'react-router'
import { toast } from 'sonner'
import { ApiError, api, call } from '@/api/client'
import { useMyEnrollments } from '@/api/queries'
import { useAuth } from '@/auth/context'
import { Button, buttonVariants } from '@/components/ui/button'
import { money } from '@/lib/format'
import { errorMessage } from '@/lib/query'

/** What a student, a visitor or the owner does with a course: log in, enroll, continue. */
export function useEnrollment(courseId: string) {
  const enrollments = useMyEnrollments()

  return enrollments.data?.items?.find((item) => item.course_id === courseId && item.status !== 'dropped')
}

export function CourseAction({ courseId, price, from }: { courseId: string; price?: number; from: string }) {
  const { user } = useAuth()
  const navigate = useNavigate()
  const client = useQueryClient()
  const enrollment = useEnrollment(courseId)

  const enroll = useMutation({
    mutationFn: () => call(api.POST('/courses/{courseId}/enrollments', { params: { path: { courseId } } })),
    onSuccess: () => {
      toast.success('You are enrolled')
      client.invalidateQueries({ queryKey: ['enrollments'] })
      client.invalidateQueries({ queryKey: ['course', courseId] })
      navigate(`/learn/${courseId}`)
    },
    onError: (error) => {
      // already enrolled on another device or tab
      if (error instanceof ApiError && error.status === 409) client.invalidateQueries({ queryKey: ['enrollments'] })

      toast.error(errorMessage(error))
    },
  })

  if (!user) {
    return (
      <Link to="/login" state={{ from }} className={buttonVariants({ size: 'lg' }) + ' w-full'}>
        Log in to enroll
      </Link>
    )
  }

  if (enrollment) {
    return (
      <Link to={`/learn/${courseId}`} className={buttonVariants({ size: 'lg' }) + ' w-full'}>
        {enrollment.status === 'completed' ? 'Review the course' : 'Continue learning'}
      </Link>
    )
  }

  if (user.role_name === 'Student') {
    return (
      <Button size="lg" className="w-full" disabled={enroll.isPending} onClick={() => enroll.mutate()}>
        {enroll.isPending ? 'Enrolling…' : price ? `Enroll for ${money(price)}` : 'Enroll for free'}
      </Button>
    )
  }

  return null
}
