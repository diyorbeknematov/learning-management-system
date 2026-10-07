# LMS Frontend

React + TypeScript + Vite. It talks to the backend in [`../backend`](../backend).

## Run

```bash
cd backend && make up        # the API and its services (see backend/README.md)
cd frontend
cp .env.example .env         # optional, see below
npm install
npm run dev                  # http://localhost:3000
```

The dev server runs on port **3000** on purpose: the backend allows this origin for CORS (`CORS_ORIGINS`) and builds the password reset link with it (`RESET_PASSWORD_URL=http://localhost:3000/reset-password`). In development `/api` is sent to `http://localhost:8080` by the Vite proxy, so the browser sees one origin. `VITE_PROXY_TARGET` changes that target; `VITE_API_URL` makes the app call another server directly (then that server must allow this origin).

Or run it in Docker together with the backend: `make up` in `backend/` also starts the `frontend` service on http://localhost:3000 (nginx serves the build and sends `/api/` to the backend).

## Scripts

| Command | What it does |
|---|---|
| `npm run dev` | dev server with hot reload |
| `npm run build` | type check and production build into `dist/` |
| `npm run typecheck` | `tsc -b` |
| `npm run lint` | oxlint |
| `npm run api:types` | regenerates `src/api/schema.d.ts` from `../backend/docs/swagger/swagger.json` |

## Talking to the API

`src/api/client.ts` has a typed client (`openapi-fetch`) over the generated types, so a wrong path, parameter or body is a compile error:

```ts
import { api, call } from './api/client'

const page = await call(api.GET('/courses', { params: { query: { q: 'go', limit: 10 } } }))
const me = await call(api.GET('/users/me'))
```

* `call()` returns the `data` of `{"success": true, "data": ...}` and throws an `ApiError` (`status`, `code`, `message`, `fields`) for `{"success": false, "error": {...}}`. `fields` says which input field is wrong, to show next to the form field.
* The access token is added to every request. When the API answers 401 and there is a refresh token, the client renews the pair once and repeats the request; if that fails the tokens are cleared and the user has to log in again. Save the tokens after login with `tokens.set(access, refresh)` (`src/api/tokens.ts`).
* A `429` answer means a rate limit; the `Retry-After` header says in seconds when to try again.

**After the backend API changes**, run `make postman` in `backend/` (it regenerates Swagger), then `npm run api:types` here, and fix what the compiler points at.

## Things the backend does that the UI should know

* Roles: `SuperAdmin`, `Instructor`, `Student` (`role_name` of the user). Registration creates Students; a SuperAdmin creates Instructors.
* A blocked or deleted user, or one whose password or role changed, gets 401 on the old token at once.
* Files: ask `POST /uploads/presign`, `PUT` the file to the returned `upload_url`, then send the returned `object_key` when saving the profile, course or material. Files come back as temporary links (`avatar_url`, `cover_url`, `file_url`).
* Lists are `{items, total, page, limit}`.
* Certificates have a public check page: `GET /certificates/verify/:uniqueId` (the QR code of a certificate points to `CERTIFICATE_VERIFY_URL`).
