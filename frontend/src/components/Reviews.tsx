import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { toast } from 'sonner'
import { api, call } from '@/api/client'
import { useAuth } from '@/auth/context'
import { NativeSelect } from '@/components/NativeSelect'
import { Pagination } from '@/components/Pagination'
import { Rating } from '@/components/Rating'
import { Empty, ErrorBlock, LoadingBlock } from '@/components/States'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { formatDate } from '@/lib/format'
import { errorMessage } from '@/lib/query'

const LIMIT = 10

function ReviewForm({
  courseId,
  existing,
  onDone,
}: {
  courseId: string
  existing?: { id?: string; rating?: number; comment?: string }
  onDone: () => void
}) {
  const client = useQueryClient()
  const [rating, setRating] = useState(String(existing?.rating ?? 5))
  const [comment, setComment] = useState(existing?.comment ?? '')

  const save = useMutation({
    mutationFn: () => {
      const body = { rating: Number(rating), comment: comment.trim() }

      return existing?.id
        ? call(api.PUT('/reviews/{reviewId}', { params: { path: { reviewId: existing.id } }, body }))
        : call(api.POST('/courses/{courseId}/reviews', { params: { path: { courseId } }, body }))
    },
    onSuccess: () => {
      toast.success(existing?.id ? 'Review updated' : 'Thank you for your review')
      client.invalidateQueries({ queryKey: ['reviews', courseId] })
      client.invalidateQueries({ queryKey: ['course', courseId] })
      onDone()
    },
    onError: (error) => toast.error(errorMessage(error)),
  })

  return (
    <form
      className="space-y-3 rounded-lg border p-4"
      onSubmit={(e) => {
        e.preventDefault()
        save.mutate()
      }}
    >
      <div className="max-w-40 space-y-1.5">
        <Label htmlFor="review-rating">Your rating</Label>
        <NativeSelect id="review-rating" value={rating} onChange={(e) => setRating(e.target.value)}>
          {[5, 4, 3, 2, 1].map((n) => (
            <option key={n} value={n}>
              {n} {n === 1 ? 'star' : 'stars'}
            </option>
          ))}
        </NativeSelect>
      </div>
      <div className="space-y-1.5">
        <Label htmlFor="review-comment">Your review (optional)</Label>
        <Textarea id="review-comment" rows={3} value={comment} onChange={(e) => setComment(e.target.value)} />
      </div>
      <div className="flex gap-2">
        <Button type="submit" disabled={save.isPending}>
          {save.isPending ? 'Saving…' : existing?.id ? 'Update the review' : 'Send the review'}
        </Button>
        {existing?.id && (
          <Button type="button" variant="ghost" onClick={onDone}>
            Cancel
          </Button>
        )}
      </div>
    </form>
  )
}

/** The reviews of a course; a student of the course can write or change their own. */
export function Reviews({ courseId, canReview }: { courseId: string; canReview: boolean }) {
  const { user } = useAuth()
  const client = useQueryClient()
  const [page, setPage] = useState(1)
  const [editing, setEditing] = useState(false)

  const reviews = useQuery({
    queryKey: ['reviews', courseId, page],
    queryFn: () => call(api.GET('/courses/{courseId}/reviews', { params: { path: { courseId }, query: { page, limit: LIMIT } } })),
    placeholderData: (previous) => previous,
  })

  const remove = useMutation({
    mutationFn: (id: string) => call(api.DELETE('/reviews/{reviewId}', { params: { path: { reviewId: id } } })),
    onSuccess: () => {
      toast.success('Review deleted')
      client.invalidateQueries({ queryKey: ['reviews', courseId] })
      client.invalidateQueries({ queryKey: ['course', courseId] })
    },
    onError: (error) => toast.error(errorMessage(error)),
  })

  const mine = reviews.data?.items?.find((review) => review.student_id === user?.id)

  return (
    <section className="space-y-4" aria-labelledby="reviews-title">
      <div className="flex flex-wrap items-center gap-3">
        <h2 id="reviews-title" className="text-xl font-semibold">
          Reviews
        </h2>
        {reviews.data && (reviews.data.total ?? 0) > 0 && <Rating value={reviews.data.avg_rating} count={reviews.data.total} />}
      </div>

      {canReview && (!mine || editing) && <ReviewForm courseId={courseId} existing={mine} onDone={() => setEditing(false)} />}

      {reviews.isPending && <LoadingBlock rows={2} />}
      {reviews.isError && <ErrorBlock error={reviews.error} />}
      {reviews.data?.items?.length === 0 && <Empty title="No reviews yet" />}

      <ul className="space-y-4">
        {reviews.data?.items?.map((review) => (
          <li key={review.id} className="space-y-1 border-b pb-4 last:border-0">
            <div className="flex flex-wrap items-center gap-2">
              <span className="font-medium">{review.student_name}</span>
              <Rating value={review.rating} />
              <span className="text-xs text-muted-foreground">{formatDate(review.created_at)}</span>
            </div>
            {review.comment && <p className="text-sm">{review.comment}</p>}
            {review.id === mine?.id && !editing && (
              <div className="flex gap-2 pt-1">
                <Button size="xs" variant="outline" onClick={() => setEditing(true)}>
                  Edit
                </Button>
                <Button size="xs" variant="ghost" disabled={remove.isPending} onClick={() => remove.mutate(review.id!)}>
                  Delete
                </Button>
              </div>
            )}
          </li>
        ))}
      </ul>

      <Pagination page={page} limit={LIMIT} total={reviews.data?.total ?? 0} onPage={setPage} />
    </section>
  )
}
