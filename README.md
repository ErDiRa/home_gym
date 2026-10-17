<p align="center">
  <img src="frontend/src/assets/logo.svg" alt="HomeGym logo" width="96" />
</p>

# HomeGym

![Status: in development](https://img.shields.io/badge/status-in%20development-orange)

> [!WARNING]
> HomeGym is in early development. Features, the API and the database schema can change at any time, and it is not ready for production use.

A workout planner for coaches. Admins manage an exercise library and build trainings for their clients: training days on fixed weekdays or dates, made of blocks (single, superset, complex). Each exercise in a block has its own sets, reps, load and tempo.

## Tech stack

- **Backend:** Go, [chi](https://github.com/go-chi/chi), PostgreSQL 18, [sqlc](https://sqlc.dev) for type-safe queries
- **Frontend:** Vue 3, TypeScript, Vite, Tailwind CSS
- **Dev setup:** Docker Compose (Postgres + API with hot reload via [air](https://github.com/air-verse/air)), [Task](https://taskfile.dev) as the task runner

## Getting started

Requirements: Docker, Node.js 20+, [Task](https://taskfile.dev/installation/) (optional, but the commands below use it).

```sh
# Create the database from the schema and seed data, then start the API on :8080
task backend:fresh

# Start the frontend on http://localhost:5173
task frontend:dev
```

`task dev` starts both at once, keeping the existing database. Vite proxies `/api` to the backend.

Seed logins (password `test` for all):

| Account             | Role                         |
| ------------------- | ---------------------------- |
| `admin@example.com` | Admin                        |
| `anna@example.com`  | Client with sample trainings |
| `ben@example.com`   | Client with a home training  |
| `clara@example.com` | Client without trainings     |

### Without Task

```sh
cd backend && docker compose up --build    # API + Postgres
cd frontend && npm install && npm run dev  # frontend
```

## Configuration

The backend works out of the box with dev defaults. To override them, copy `backend/.env.example` to `backend/.env`:

| Variable                  | Purpose                                                      |
| ------------------------- | ------------------------------------------------------------ |
| `POSTGRES_*`              | Database user, password and name                             |
| `JWT_SECRET`              | Secret for signing session tokens. Change it outside of dev. |
| `COOKIE_SECURE`           | Set to `true` when served over HTTPS                         |
| `CORS_ALLOWED_ORIGINS`    | Allowed frontend origins                                     |
| `LOG_LEVEL`, `LOG_FORMAT` | `debug`/`info`/`warn`/`error`, `text`/`json`                 |

## Project layout

```text
backend/
  cmd/api/            entrypoint
  internal/handlers/  HTTP handlers and routes (thin adapters over the services)
  internal/service/   business logic, one package per domain (training, client, ...):
                      one file per usecase, its SQL in queries.sql and the
                      sqlc-generated code in repos/ (don't edit by hand)
  db/migrations/      schema
  db/seed/            dev seed data
frontend/
  src/views/          pages (exercises, trainings, clients, login)
  src/components/     shared UI components
```

## Development

| Command                                         | What it does                                                  |
| ----------------------------------------------- | ------------------------------------------------------------- |
| `task --list`                                   | Show all tasks                                                |
| `task backend:fresh`                            | Wipe the database and recreate it from schema and seeds       |
| `cd backend && go test ./...`                   | Run the Go tests                                              |
| `docker exec -it workout_backend sqlc generate` | Regenerate database code after changing queries or the schema |
| `cd frontend && npm run build`                  | Type-check and build the frontend                             |

There are no migrations yet: during v1 development the schema in `backend/db/migrations/000001_init_schema.up.sql` is edited directly, so run `task backend:fresh` after pulling schema changes.

Users and admins are created directly in the database; there is no sign-up.

Private seed data goes into `backend/db/seed/local/` (git-ignored): every `*.sql` file there is loaded after the exercise seeds when the database is created.
