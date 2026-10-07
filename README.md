# Learning Management System (LMS)

A Learning Management System for managing courses, students, instructors, quizzes, certificates, enrollments, and platform finances.

## Project Overview

This project is designed to provide an online learning platform where:

* **SuperAdmins** can manage users, courses, roles, and platform finances.
* **Instructors** can create and manage their own courses, modules, lessons, and quizzes.
* **Students** can enroll in courses, complete lessons, take quizzes, receive certificates, and leave reviews.

## Quick Start (Docker)

You need Docker with the Compose plugin.

```bash
make up            # or: docker compose up --build -d
```

This starts the API, PostgreSQL, Redis, MinIO and Mailpit, applies the migrations and creates the first SuperAdmin.

| What | Address |
|---|---|
| API | http://localhost:8080 |
| Swagger UI (try the API in the browser) | http://localhost:8080/swagger/index.html |
| Mailpit (the emails the API sends, e.g. password reset) | http://localhost:8025 |
| MinIO console | http://localhost:9001 (`minioadmin` / `minioadmin`) |

First SuperAdmin (development only): username `admin`, password `Admin-Passw0rd!2024`. Change it after the first login (`PUT /api/v1/users/me/password`).

Useful commands: `make logs` (API logs), `make down` (stop, data stays), `make reset` (stop and delete all data). If a port is taken, set `POSTGRES_PORT`, `REDIS_PORT`, `HTTP_PORT`, ... in a `.env` file or the environment.

## Run Locally (without Docker for the API)

1. Start PostgreSQL, Redis and MinIO (`docker compose up -d postgres redis minio mailpit`).
2. `cp .env.example .env` and set the values (`DB_*`, `REDIS_*`, `MINIO_*`, `TOKEN_SECRET`, `ADMIN_PASSWORD`).
3. `make mig-up` (needs the [migrate](https://github.com/golang-migrate/migrate) CLI).
4. `make run`.

`make help` lists every command: tests (`make test`, they need PostgreSQL and Redis and are skipped without them), migrations, formatting and so on.

### The token secret

`TOKEN_SECRET` signs the access tokens (JWT). In production the API refuses to start unless it is a random string of at least 32 characters. Make one with OpenSSL and paste the result as `TOKEN_SECRET` in `.env`:

```bash
openssl rand -hex 32
```

Keep it secret and do not commit it. Changing it logs everybody out (all issued access tokens stop working).

### The first SuperAdmin

Registration creates Students only. At start-up the API creates a SuperAdmin from `ADMIN_USERNAME`, `ADMIN_EMAIL` and `ADMIN_PASSWORD` when `ADMIN_PASSWORD` is set and the system has no SuperAdmin yet. It does nothing on the next starts. Then the SuperAdmin creates Instructors with `POST /api/v1/users`.

## API Documentation

* [`docs/api-endpoints.md`](docs/api-endpoints.md): every route, who may call it, limits and error codes.
* **Swagger:** generated from the annotations of the handlers into [`docs/swagger`](docs/swagger); `make swagger` regenerates it. The UI is served at `/swagger/index.html` (not in production).
* **Postman:** import [`docs/postman/lms.postman_collection.json`](docs/postman/lms.postman_collection.json) and [`lms.postman_environment.json`](docs/postman/lms.postman_environment.json), select the environment, and run *Auth → Log in*: the tokens and the ids of created resources are saved automatically. `make postman` regenerates both files.

After changing a handler or a request/response model, run `make postman` (it runs `make swagger` first) and commit the result.

## Database

The schema is in [`migrations/`](migrations). The diagram and the description of every table are in [`docs/erd.md`](docs/erd.md).

## System Architecture

The backend follows a layered approach:

```text
Client
  ↓
Go REST API
  ↓
JWT Authentication
  ↓
Casbin Authorization
  ↓
Handler
  ↓
Service
  ↓
Repository
  ↓
PostgreSQL
```

Additional infrastructure:

* **Redis** — rate limits, reset tokens and ended sessions
* **MinIO** — object storage for videos, files, images, and certificates

Detailed architecture documentation is available in [`docs/architecture.md`](docs/architecture.md).

## Tech Stack

* **Backend:** Go
* **Database:** PostgreSQL
* **Cache:** Redis
* **Object Storage:** MinIO
* **Authentication:** JWT + bcrypt
* **Authorization:** Casbin
