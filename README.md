# Edura

An online learning platform (a Learning Management System): instructors build courses, students learn and earn certificates, and a SuperAdmin runs the platform and its finances.

| Role | What they do |
|---|---|
| **Student** | Browses the catalog, enrolls (free or paid), learns lesson by lesson, takes quizzes, earns a certificate with a QR code, reviews courses |
| **Instructor** | Creates courses with modules, lessons, materials (text, video, files) and quizzes; sees the progress and quiz results of every student |
| **SuperAdmin** | Manages users, categories and payments; sees finance charts and reports |

## Stack

| Part | With |
|---|---|
| [`backend/`](backend) | Go, Gin, PostgreSQL, Redis, MinIO, Casbin (roles), JWT with rotating refresh tokens, Swagger |
| [`frontend/`](frontend) | React 19, TypeScript, Vite, Tailwind CSS 4, shadcn/ui, TanStack Query, Playwright |

Each part has its own README with the details: [backend](backend/README.md), [frontend](frontend/README.md).

## Quick start

You need Docker with the Compose plugin.

```bash
cd backend
make up      # API, frontend, PostgreSQL, Redis, MinIO and Mailpit
make seed    # optional: demo courses, teachers and students
```

`make seed` needs `ADMIN_PASSWORD`, for example `ADMIN_PASSWORD='Admin-Passw0rd!2024' make seed`. `make seed-spread` then spreads the demo payments over six months, so the finance charts have something to show.

| What | Address |
|---|---|
| App | http://localhost:3000 |
| API and Swagger UI | http://localhost:8080/swagger/index.html |
| Mailpit (emails such as password reset) | http://localhost:8025 |
| MinIO console | http://localhost:9001 (`minioadmin` / `minioadmin`) |

Demo logins (development only):

| Who | Username | Password |
|---|---|---|
| SuperAdmin | `admin` | `Admin-Passw0rd!2024` |
| Instructors (after `make seed`) | `ali.teacher`, `sara.teacher` | `Demo-Passw0rd!2024` |
| Students (after `make seed`) | `student1` … `student5` | `Demo-Passw0rd!2024` |

## Development

Run the frontend with hot reload while the API runs in Docker:

```bash
cd frontend
npm install
npm run dev
```

Other commands: `make test`, `make check` (fmt, vet, test), `make logs`, `make down` (data stays), `make reset` (deletes all data) in `backend/`; `npm run lint`, `npm run build`, `npm run test:e2e` in `frontend/`. `make help` lists everything.

## Repository layout

```text
backend/    Go API, migrations, Docker Compose, Swagger and Postman docs
frontend/   React app and Playwright end-to-end tests
```
