import { Link } from 'react-router'

export default function NotFoundPage() {
  return (
    <div className="space-y-2 py-16 text-center">
      <h1 className="text-2xl font-semibold">Page not found</h1>
      <Link to="/" className="underline">
        Back to the courses
      </Link>
    </div>
  )
}
