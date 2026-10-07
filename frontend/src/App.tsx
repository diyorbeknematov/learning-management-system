import { useEffect, useState } from 'react'
import { api, ApiError, call } from './api/client'
import './index.css'

type Course = { id?: string; title?: string; instructor_name?: string; price?: number }

// A first page that proves the frontend talks to the API with the generated
// types: it lists the published courses of the catalog.
export default function App() {
  const [courses, setCourses] = useState<Course[] | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    call(api.GET('/courses', { params: { query: { limit: 10 } } }))
      .then((page) => setCourses(page.items ?? []))
      .catch((e: unknown) => setError(e instanceof ApiError ? e.message : 'The API is not reachable'))
  }, [])

  return (
    <main>
      <h1>Learning Management System</h1>

      {error && <p role="alert">{error}</p>}
      {!error && courses === null && <p>Loading…</p>}
      {courses?.length === 0 && <p>No published courses yet.</p>}

      <ul>
        {courses?.map((course) => (
          <li key={course.id}>
            {course.title} <small>{course.instructor_name}</small>
          </li>
        ))}
      </ul>
    </main>
  )
}
