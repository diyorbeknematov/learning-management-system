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

## Pages

| Who | Address | What |
|---|---|---|
| everybody | `/` | catalog: search, category, level, price, rating, sort, pages (all in the address) |
| everybody | `/courses/:id` | course page: what you learn, syllabus, instructor, reviews, enroll |
| everybody | `/verify/:number` | check a certificate |
| guests | `/login`, `/register`, `/forgot-password`, `/reset-password` | accounts |
| logged in | `/profile`, `/quizzes/:id` | profile and password; quiz |
| student | `/my-courses`, `/learn/:course/lessons/:lesson`, `/certificates` | studying, progress, certificates (PDF) |
| instructor, SuperAdmin | `/teach/courses`, `/teach/courses/new`, `/teach/courses/:id` | course list; editor with Details, Content (modules, lessons, materials, file upload), Quizzes (questions), Students |
| SuperAdmin | `/admin/users`, `/admin/categories`, `/admin/payments`, `/admin/finance`, `/admin/reports` | users (create, edit, block, delete), categories, payments, finance, reports with CSV |

Files (covers, materials) are uploaded from the browser straight to MinIO, so the backend must know the address the browser uses: `MINIO_PUBLIC_ENDPOINT` (the compose file sets it to `localhost:9000`).

Ideas for later: avatar upload on the profile, a dashboard on the home page of each role, a Playwright test suite in the repository.
