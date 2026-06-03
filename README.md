# zenfl-forwarder platform

Monorepo for a Telegram-to-job-board platform.

## Current Scope

- Telegram personal-account listener (Zenfl messages)
- Message forwarding to configured users
- Message persistence to MongoDB
- REST API with demo auth
- React + Vite demo job-board frontend

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

Demo API auth user (auto-created at backend startup):

- `demo@zenfl.local`
- `demo1234`

## API Endpoints

- `GET /api/health`
- `POST /api/auth/login`
- `GET /api/auth/me` (Bearer token)
- `GET /api/jobs` (Bearer token)

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
