# zenfl-forwarder platform

Monorepo for a Telegram-to-job-board platform.

## Current Scope

- Telegram personal-account listener (Zenfl messages)
- Message forwarding to configured users
- Message persistence to MongoDB
- REST API with demo auth
- React + Vite demo job-board frontend

## Backend Run

1. Copy `.env.example` to `.env` and fill Telegram values.
2. Start MongoDB locally.
3. Run:

```bash
go mod tidy
go run ./backend/cmd/server
```

Demo API auth user (auto-created at startup):

- `demo@zenfl.local`
- `demo1234`

## API Endpoints

- `GET /api/health`
- `POST /api/auth/login`
- `GET /api/auth/me` (Bearer token)
- `GET /api/jobs` (Bearer token)

## Frontend Run

```bash
cd frontend
npm install
npm run dev
```

Optional frontend env:

- `VITE_API_BASE` (default: `http://localhost:8080`)

## Notes

- App does not mark messages as read.
- Inline Telegram buttons from bot messages are not preserved when sending from user accounts.
- Fallback currently appends `Upwork job link: <url>` to outgoing message text.
