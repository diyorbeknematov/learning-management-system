import { Link } from 'react-router'
import { buttonVariants } from '@/components/ui/button'

export default function NotFoundPage() {
  return (
    <div className="mx-auto flex max-w-md flex-col items-center gap-4 py-24 text-center">
      <p className="text-7xl font-bold text-primary">404</p>
      <h1 className="text-2xl font-semibold">This page was not found</h1>
      <p className="text-muted-foreground">The address may be wrong, or the page was moved.</p>
      <Link to="/courses" className={buttonVariants({ size: 'lg' })}>
        Back to the courses
      </Link>
    </div>
  )
}
