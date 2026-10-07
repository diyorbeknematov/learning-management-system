import { CourseDetailsForm } from './CourseDetailsForm'

export default function NewCoursePage() {
  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-semibold">New course</h1>
      <p className="text-muted-foreground">Fill in the basics now. You add the lessons and quizzes after the course is created.</p>
      <CourseDetailsForm />
    </div>
  )
}
