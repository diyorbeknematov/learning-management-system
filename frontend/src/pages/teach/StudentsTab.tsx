import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { api, call } from '@/api/client'
import { Pagination } from '@/components/Pagination'
import { Empty, ErrorBlock, LoadingBlock } from '@/components/States'
import { Progress } from '@/components/ui/progress'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { formatDate } from '@/lib/format'

const LIMIT = 20

/** The students of a course with how far each has come. */
export function StudentsTab({ courseId }: { courseId: string }) {
  const [page, setPage] = useState(1)

  const students = useQuery({
    queryKey: ['students', courseId, page],
    queryFn: () => call(api.GET('/courses/{courseId}/students', { params: { path: { courseId }, query: { page, limit: LIMIT } } })),
    placeholderData: (previous) => previous,
  })

  if (students.isPending) return <LoadingBlock />
  if (students.isError) return <ErrorBlock error={students.error} />
  if (students.data.items?.length === 0) return <Empty title="No students yet" />

  return (
    <div className="space-y-4">
      <p className="text-sm text-muted-foreground">{students.data.total} students</p>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Student</TableHead>
            <TableHead className="w-56">Progress</TableHead>
            <TableHead>Last activity</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {students.data.items?.map((student) => (
            <TableRow key={student.student_id}>
              <TableCell>
                <div className="font-medium">{student.full_name}</div>
                <div className="text-xs text-muted-foreground">{student.email}</div>
              </TableCell>
              <TableCell>
                <Progress value={student.progress_percent ?? 0} aria-label={`Progress of ${student.full_name}`} />
                <span className="text-xs text-muted-foreground">
                  {student.completed_lessons ?? 0} of {student.total_lessons ?? 0} lessons
                </span>
              </TableCell>
              <TableCell>{formatDate(student.last_activity_at)}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
      <Pagination page={page} limit={LIMIT} total={students.data.total ?? 0} onPage={setPage} />
    </div>
  )
}
