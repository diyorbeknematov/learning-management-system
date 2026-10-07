# LMS Frontend

React 19 + TypeScript + Vite. It talks to the backend in [`../backend`](../backend).

| What | With |
|---|---|
| Pages | React Router (`src/router.tsx`) |
| Server data | TanStack Query (`src/lib/query.ts`) |
| Forms | React Hook Form + Zod (`src/lib/validation.ts` has the rules of the backend) |
| UI | Tailwind CSS 4 + shadcn/ui (`src/components/ui`); add a component with `npx shadcn@latest add <name>` |
| API | `openapi-fetch` over types generated from Swagger (`src/api`) |

```text
src/
├── api/          client, tokens, generated schema.d.ts
├── auth/         AuthProvider, useAuth, route guards (RequireAuth, GuestOnly)
├── components/   layout, shared form parts, ui/ (shadcn)
├── lib/          query client, validation helpers
├── pages/        one file per page
└── router.tsx
```

## End-to-end tests

`e2e/lms.spec.ts` plays one course from start to end in a real browser, as four people: a SuperAdmin makes a category and an instructor; the instructor builds and publishes a course (cover, module, lesson, material, quiz); a visitor finds it; a student enrolls, learns, passes the quiz, downloads and shares the certificate, reviews the course; the SuperAdmin checks the money and blocks the student, who is signed out at once.

```bash
cd backend && RATE_LIMIT_ENABLED=false make up   # the rate limits would stop the many sign-ups
cd ../frontend && npm run test:e2e
```

It uses the Chrome that is installed. Settings: `E2E_BASE_URL` (default `http://localhost:3000`), `E2E_ADMIN_PASSWORD` (default is the one of the compose file), `E2E_CHANNEL` (`chrome`; use `chromium` after `npx playwright install chromium`). Every run creates its own users and course, so it can be repeated on the same data. After a failure, look at `test-results/` (screenshots and a trace).

## Pages

| Who | Address | What |
|---|---|---|
| everybody | `/` | catalog: search, category, level, price, rating, sort, pages (all in the address) |
| everybody | `/courses/:id` | course page: what you learn, syllabus, instructor, reviews, enroll |
| everybody | `/verify/:number` | check a certificate |
| guests | `/login`, `/register`, `/forgot-password`, `/reset-password` | accounts |
| logged in | `/dashboard`, `/profile`, `/quizzes/:id` | what matters for the role (the page after login); profile with photo and password; quiz |
| student | `/my-courses`, `/learn/:course/lessons/:lesson`, `/certificates` | studying, progress, certificates (PDF) |
| instructor, SuperAdmin | `/teach/courses`, `/teach/courses/new`, `/teach/courses/:id` | course list; editor with Details, Content (modules, lessons, materials, file upload), Quizzes (questions), Students |
| SuperAdmin | `/admin/users`, `/admin/categories`, `/admin/payments`, `/admin/finance`, `/admin/reports` | users (create, edit, block, delete), categories, payments, finance, reports with CSV |

Files (covers, materials) are uploaded from the browser straight to MinIO, so the backend must know the address the browser uses: `MINIO_PUBLIC_ENDPOINT` (the compose file sets it to `localhost:9000`).

Ideas for later: charts in the reports, choosing the instructor when a SuperAdmin creates a course.
