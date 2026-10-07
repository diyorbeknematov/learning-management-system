import { BookOpen, Clock } from 'lucide-react'
import { Link } from 'react-router'
import type { components } from '@/api/schema'
import { Rating } from '@/components/Rating'
import { Badge } from '@/components/ui/badge'
import { Card } from '@/components/ui/card'
import { capitalize, duration, money, plural } from '@/lib/format'

type CourseItem = components['schemas']['models.CourseListItem']

// the picture is decoration: the title stands beside it
export function CourseCover({ url, className }: { url?: string; title?: string; className?: string }) {
  if (url) return <img src={url} alt="" className={className} loading="lazy" />

  return (
    <div
      className={`flex items-center justify-center bg-gradient-to-br from-primary/25 via-primary/10 to-primary/5 text-primary/60 ${className ?? ''}`}
      aria-hidden
    >
      <BookOpen className="size-12" />
    </div>
  )
}

export function CourseCard({ course, label }: { course: CourseItem; label?: string }) {
  return (
    <Link to={`/courses/${course.id}`} className="group block h-full rounded-2xl outline-none focus-visible:ring-3 focus-visible:ring-ring/50">
      <Card className="h-full gap-0 overflow-hidden rounded-2xl py-0 transition duration-200 group-hover:-translate-y-0.5 group-hover:shadow-lg">
        <div className="relative">
          <CourseCover url={course.cover_url} title={course.title} className="aspect-video w-full object-cover" />
          {course.status === 'draft' && <Badge className="absolute left-3 top-3">Draft</Badge>}
          {label && course.status !== 'draft' && <Badge className="absolute left-3 top-3 border-transparent bg-amber-400 text-amber-950 shadow">{label}</Badge>}
        </div>

        <div className="flex flex-1 flex-col gap-3 p-5">
          <div className="space-y-1">
            {course.category_name && <p className="text-xs font-medium uppercase tracking-wide text-primary">{course.category_name}</p>}
            <h3 className="line-clamp-2 text-lg font-semibold leading-snug group-hover:text-primary">{course.title}</h3>
            <p className="text-sm text-muted-foreground">{course.instructor_name}</p>
          </div>

          <Rating value={course.avg_rating} count={course.review_count} />

          <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground">
            <span>{plural(course.lesson_count, 'lesson')}</span>
            {course.total_duration ? (
              <span className="inline-flex items-center gap-1">
                <Clock className="size-3.5" /> {duration(course.total_duration)}
              </span>
            ) : null}
            {course.difficulty && <span>{capitalize(course.difficulty)}</span>}
          </div>

          <div className="mt-auto flex items-center justify-between border-t pt-4">
            <span className="text-xl font-bold">{money(course.price)}</span>
            <span className="text-sm font-medium text-primary opacity-0 transition group-hover:opacity-100">View course →</span>
          </div>
        </div>
      </Card>
    </Link>
  )
}
