import { PageHeader } from '@/components/PageHeader'
import { CourseDetailsForm } from './CourseDetailsForm'

export default function NewCoursePage() {
  return (
    <div className="space-y-6">
      <PageHeader title="New course" description="Fill in the basics now. You add the lessons and quizzes after the course is created." />
      <CourseDetailsForm />
    </div>
  )
}
