<p align="center">
  <img src="frontend/app/src/assets/logo.svg" alt="HomeGym logo" width="96" />
</p>

# HomeGym

![Status: in development](https://img.shields.io/badge/status-in%20development-orange)

> [!WARNING]
> HomeGym is in early development. Features, the API and the database schema can change at any time, and it is not ready for production use.

A personal workout planner and tracker. In the dashboard you keep your exercise library and plan trainings: units (e.g. "Day A") made of blocks (single, superset, complex), each exercise with its sets, reps, load and tempo, and a number of sessions per week instead of fixed days. The mobile-first tracker shows this week's sessions like a todo list, suggests the next unit, and logs every set as you go.

## Features

- **Exercises:** your own library with joints, muscle groups, focuses, video links and ratings.
- **PRs (1RM):** your one-rep max per exercise in kg or lb, with its history and PRs, on its own dashboard page (Log PR: search the exercise, enter the max). Loads planned in % 1RM show as a weight in the tracker, switching between kg and lb converts, and a logged single above your 1RM offers to save it as the new one.
- **Trainings:** sessions made of blocks (single, superset, complex, WOD with for time, AMRAP, EMOM, max reps or calories), each exercise with sets, reps, load and tempo, and an explicit plan for every week. Several trainings can be active side by side.
- **Cycles:** a planned period of weeks with goals (e.g. "Mobility, 8 weeks: better hips"), grouping trainings that may run in parallel. The dashboard shows them on a week-by-week timeline; the tracker shows the cycle's week and goals above the training.
- **Tracker:** this week's sessions as cards, logging sets and WOD scores as you go, and the history.
- **Import:** a training week or a whole cycle from a JSON file (see [Import formats](#import-formats)).

## Tech stack

- **Backend:** Go, [chi](https://github.com/go-chi/chi), PostgreSQL 18, [sqlc](https://sqlc.dev) for type-safe queries. A modular monolith: five modules (identity, catalog, planning, tracking, whoop) in one binary, each with its own Postgres schema and its own layers (api, app, adapters); modules call each other through small Go contracts, never over the network or each other's tables
- **API:** API first: one [OpenAPI](https://www.openapis.org) spec per module, from which [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen) generates the Go server interface, models and a client, and [openapi-typescript](https://openapi-ts.dev) the frontend types (client: openapi-fetch)
- **Frontend:** Vue 3, TypeScript, Vite, Tailwind CSS (one app: dashboard and tracker)
- **Dev setup:** Docker Compose (Postgres + API with hot reload via [air](https://github.com/air-verse/air)), [Task](https://taskfile.dev) as the task runner

## Getting started

Requirements: Docker, Node.js 20+, [Task](https://taskfile.dev/installation/) (optional, but the commands below use it).

```sh
# Create the database (the API migrates it on start), load the seed data; the API runs on :8080
task backend:fresh

# Start the app on http://localhost:5173
task app:dev
```

`task dev` starts everything at once, keeping the existing database. Vite proxies `/api` to the backend. `/` opens the tracker (`/track`) on phones and the dashboard (`/dashboard`) on larger screens.

Seed login: `me@example.com` / `test`, with sample exercises, three trainings (one active, 3 sessions a week) and past workouts.

### Without Task

```sh
cd backend && docker compose up --build                  # API + Postgres
cd frontend/app && npm install && npm run dev            # app
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
| `STATIC_DIR`              | Serve the built frontend from this directory (set in the image) |
| `WHOOP_CLIENT_ID`, `WHOOP_CLIENT_SECRET`, `WHOOP_REDIRECT_URL` | Optional WHOOP app, see below |

### WHOOP

Finished workouts can show the strain, heart rate, energy and heart-rate zones of a WHOOP activity plus that day's recovery. To turn it on, create an app in the [WHOOP Developer Dashboard](https://developer.whoop.com) with the scopes `read:workout`, `read:recovery` and `offline`. Register the redirect URL `http://localhost:5173/api/whoop/callback` (dev) or `https://<DOMAIN>/api/whoop/callback` (production; `deploy/docker-compose.yml` sets it), and set `WHOOP_CLIENT_ID` and `WHOOP_CLIENT_SECRET`. Then connect under Tracker → Settings, and on a finished workout tap **Retrieve WHOOP data** to pick that day's activity.

## Production

The root `Dockerfile` builds one image: the Go API, which also serves the built frontend (`index.html` with `no-cache`, the hashed `assets/` cached for a year, unknown paths fall back to `index.html` for the client routes). `deploy/docker-compose.yml` runs it with Postgres and [Caddy](https://caddyserver.com) (automatic HTTPS); container logs use Docker's `json-file` driver, rotated at 10 MB with 5 compressed files kept.

```sh
# Locally: build for linux/amd64 and push (after `docker login ghcr.io`)
task image:push APP_IMAGE=ghcr.io/<user>/homegym:<tag>

# Copy the compose file, Caddyfile and the schema it mounts from backend/db to /opt/homegym
# on the server (or use a checkout of this repo there)
task prod:copy SERVER=user@host

# On the server, in /opt/homegym
cp deploy/.env.example deploy/.env    # set DOMAIN, APP_IMAGE, passwords, JWT_SECRET
task prod:up                          # or: cd deploy && docker compose pull && docker compose up -d
task prod:logs
```

DNS for `DOMAIN` must point at the server and ports 80/443 must be open. The app creates and migrates the database itself when it starts: each module its own schema, including the lists exercises pick from (joints, muscle groups, focuses); no dev users or exercises. Create a user with:

```sh
cd deploy && docker compose exec db sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB"'
```

```sql
INSERT INTO identity.users (email, first_name, last_name, password_hash)
VALUES ('you@example.com', 'First', 'Last', crypt('password', gen_salt('bf', 12)));
```

Schema changes ship as migrations of the module that owns the tables (`backend/modules/<module>/adapters/db/migrations`), embedded in the image: on start the app applies each module's pending ones in its schema (versions in `<module>.schema_migrations`).

**Upgrading a database from before the modules** (all tables in `public`): the first start of this version moves every table into its module's schema in one transaction (`backend/legacy`), then migrates as usual. Take a backup first:

```sh
cd deploy && docker compose exec db sh -c 'pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" -Fc' > backup-$(date +%F).dump
```

## Project layout

```text
backend/
  cmd/api/            entrypoint: config from the environment, then backend.New(...).Run
  svc.go              wires the modules: migrations, contracts, routes under /api, the built frontend
  modules/            the modules, one directory each:
    identity/         users and sessions (login, JWT cookie)
    catalog/          exercises, ratings and the shared lists (joints, focuses, units, block kinds)
    planning/         cycles, trainings with their week plans, the session library
    tracking/         the week, workouts with sets and WOD scores, 1RMs, stats
    whoop/            the WHOOP connection and the activity linked to a workout
      module.go       the module's wiring (migrations, contract, routes)
      app/            business logic: usecases, validation, the interfaces it needs
      adapters/db/    migrations, sqlc queries and generated code, repositories, read model
      api/http/       openapi.yaml, the generated server and client, the handlers
      api/module/     the contract other modules call (client/ is its interface)
  common/             plumbing without application logic: errors, logging, HTTP, db, the module registry
  legacy/             moves a database from before the modules into the module schemas
  tests/              component tests (the whole service against Postgres)
  db/seed/            dev seed data
frontend/
  app/                the app: dashboard (exercises, sessions, cycles, trainings) and tracker (mobile first)
deploy/               production compose file, Caddyfile and .env.example
Dockerfile            production image (frontend build + Go binary)
```

## Import formats

Trainings → Import (or Cycles → Import) reads two kinds of JSON files. Exercises, units and block kinds are named, not referenced by id; the dialog lets you map unknown exercises to yours or create them (prefilled from the file's optional `new_exercises`), and creates unknown units.

**`training-import/1`: one training of one week.** It opens in the training form as a draft, so you check it before saving. With an optional `"cycle": "<name>"`, the form preselects your cycle of that name.

```jsonc
{
  "format": "training-import/1",
  "cycle": "Mobility",              // optional
  "name": "Strength week",
  "start_date": "2026-11-02",       // a Monday
  "sessions": [                     // 1-7, planned in this order in the week
    { "name": "Mon", "blocks": [
      { "kind": "single", "exercises": [
        { "exercise": "Squat", "sets": 5, "reps": { "value": 3, "unit": "reps" }, "load": { "value": 75, "max": 80, "unit": "% 1RM" } },
        // a load with only a unit: what each set is logged in, e.g. the calories of a 4 min interval
        { "exercise": "Echo Bike", "sets": 3, "reps": { "value": 4, "unit": "min" }, "load": { "unit": "cal" } }
      ] }
    ] }
  ]
}
```

**`cycle-import/1`: a whole cycle.** It creates the cycle, its goals and all its trainings at once, or nothing if any part is invalid. Each training is placed by the cycle week it starts in (`week`, from 1) and runs `weeks` weeks (default 1, its sessions planned in each), so moving the cycle is one date. Several trainings may share a week. Trainings take no `start_date` or `format` of their own; their `sessions` are those of `training-import/1`.

```jsonc
{
  "format": "cycle-import/1",
  "name": "Mobility",
  "start_date": "2026-11-02",       // a Monday: the cycle's first week
  "weeks": 4,                       // 1-52
  "goals": [{ "title": "Better hips", "target": "Deep squat hold 2 min" }],   // up to 20
  "trainings": [                    // up to 60
    { "week": 1, "weeks": 4, "name": "Hips", "sessions": [ /* as in training-import/1 */ ] }
  ]
}
```

Complete examples are `training-week.json` and `training-week-into-cycle.json` in `frontend/app/src/components/training/__fixtures__/`, and `cycle.json` in `frontend/app/src/components/cycle/__fixtures__/`; the parsers and their rules are `trainingImport.ts` and `cycleImport.ts` in those folders.

## Development

| Command                                         | What it does                                                  |
| ----------------------------------------------- | ------------------------------------------------------------- |
| `task --list`                                   | Show all tasks                                                |
| `task backend:fresh`                            | Wipe the database, let the API recreate it, load the seeds    |
| `task test`                                     | Run the Go unit tests (no database)                           |
| `task test:integration`                         | Run the adapters' tests against the test database             |
| `task test:component`                           | Run the component tests: the whole service against the test database |
| `task db:gen`                                   | Regenerate every module's database code after changing queries or migrations |
| `task api:gen`                                  | Regenerate the API code (Go and TypeScript) after changing a module's `openapi.yaml` |
| `cd frontend/app && npm run build`              | Type-check and build the app                                  |
| `cd frontend/app && npm run test`               | Run the frontend tests (vitest)                               |

Schema changes are new migrations of the module that owns the tables (`backend/modules/<module>/adapters/db/migrations`, next number); the API applies them on start.

Users are created directly in the database; there is no sign-up.

Private seed data goes into `backend/db/seed/local/` (git-ignored): every `*.sql` file there is loaded after the exercise seeds by `task backend:fresh`.
