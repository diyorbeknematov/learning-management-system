import { BookOpen } from 'lucide-react'
import { Link } from 'react-router'
import type { components } from '@/api/schema'
import { Rating } from '@/components/Rating'
import { Badge } from '@/components/ui/badge'
import { Card } from '@/components/ui/card'
import { capitalize, money } from '@/lib/format'

type CourseItem = components['schemas']['models.CourseListItem']

export function CourseCover({ url, title, className }: { url?: string; title?: string; className?: string }) {
  if (url) return <img src={url} alt="" className={className} loading="lazy" />

  return (
    <div
      className={`flex items-center justify-center bg-gradient-to-br from-primary/20 to-primary/5 text-primary/60 ${className ?? ''}`}
      role="img"
      aria-label={title}
    >
      <BookOpen className="size-10" />
    </div>
  )
}

export function CourseCard({ course }: { course: CourseItem }) {
  return (
    <Link to={`/courses/${course.id}`} className="group block h-full rounded-xl outline-none focus-visible:ring-3 focus-visible:ring-ring/50">
      <Card className="h-full overflow-hidden transition group-hover:shadow-md">
        <CourseCover url={course.cover_url} title={course.title} className="h-36 w-full object-cover" />
        <div className="flex flex-1 flex-col gap-2 px-4 pb-4">
          <h3 className="line-clamp-2 font-medium leading-snug">{course.title}</h3>
          <p className="text-sm text-muted-foreground">{course.instructor_name}</p>
          <Rating value={course.avg_rating} count={course.review_count} />
          <div className="mt-auto flex flex-wrap items-center gap-2 pt-2">
            <span className="font-semibold">{money(course.price)}</span>
            {course.difficulty && <Badge variant="secondary">{capitalize(course.difficulty)}</Badge>}
            {course.status === 'draft' && <Badge variant="outline">Draft</Badge>}
            <span className="ml-auto text-xs text-muted-foreground">{course.lesson_count ?? 0} lessons</span>
          </div>
        </div>
      </Card>
    </Link>
  )
}
