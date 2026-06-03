# zenfl-forwarder platform

Monorepo for a Telegram-to-job-board platform.

## Current Scope

- Telegram personal-account listener (Zenfl messages)
- Message forwarding to configured users
- Parsed job persistence to MongoDB
- REST API with role-based demo auth
- Unseen-first job board and admin user management frontend

## Dev Run

1. Copy `.env.example` to `.env` and fill Telegram values.
2. Start MongoDB locally.
3. Install workspace tooling:

```bash
npm install
```

4. Start backend and frontend together:

```bash
npm run dev
```

This runs:

- Go backend from `apps/backend`
- Vite frontend from `apps/frontend`

Demo API users (auto-created at backend startup):

- `demo@zenfl.local`
- `demo1234`
- `admin@zenfl.local`
- `admin1234`

## API Endpoints

- `GET /api/health`
- `POST /api/auth/login`
- `GET /api/auth/me` (Bearer token)
- `GET /api/jobs` (Bearer token)
- `GET /api/jobs/:id` (marks job seen)
- `POST /api/jobs/:id/seen`
- `GET /api/admin/users` (admin)
- `POST /api/admin/users` (admin)
- `PATCH /api/admin/users/:id` (admin)

## Individual Run

```bash
npm run dev:backend
npm run dev:frontend
```

Optional frontend env:

- `VITE_API_BASE` (default: `http://localhost:8080`)

## Notes

- App does not mark messages as read.
- Inline Telegram buttons from bot messages are not preserved when sending from user accounts.
- Fallback currently appends `Upwork job link: <url>` to outgoing message text.
