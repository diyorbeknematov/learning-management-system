import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { api, call } from '@/api/client'
import { Pagination } from '@/components/Pagination'
import { TableCard } from '@/components/Panels'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Empty, ErrorBlock, LoadingBlock } from '@/components/States'
import { Progress } from '@/components/ui/progress'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { formatDate, plural } from '@/lib/format'

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
      <p className="text-sm text-muted-foreground">{plural(students.data.total, 'student')}</p>
      <TableCard>
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
                  <div className="flex items-center gap-3">
                    <Avatar>
                      <AvatarFallback>{student.full_name?.slice(0, 2).toUpperCase()}</AvatarFallback>
                    </Avatar>
                    <div>
                      <div className="font-medium">{student.full_name}</div>
                      <div className="text-xs text-muted-foreground">{student.email}</div>
                    </div>
                  </div>
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
      </TableCard>
      <Pagination page={page} limit={LIMIT} total={students.data.total ?? 0} onPage={setPage} />
    </div>
  )
}
