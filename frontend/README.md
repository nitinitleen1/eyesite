# Eyesite Frontend

Next.js + TypeScript + Tailwind frontend for Eyesite.

## Pages

- `/` — landing page
- `/login`, `/register` — email/password auth (JWT)
- `/dashboard` — overview stats, sessions (with drill-down), analytics + export, API key management, workspace settings

## Development

```bash
npm install
npm run dev    # http://localhost:3000
```

Expects the backend services running locally (see root [README.md](../README.md)):
auth `:8000`, ingest `:8001`, query `:8002`.

Override service URLs via env:

```bash
NEXT_PUBLIC_API_URL=http://localhost:8000     # auth service
NEXT_PUBLIC_QUERY_URL=http://localhost:8002   # query service
NEXT_PUBLIC_INGEST_URL=http://localhost:8001  # ingest service
```

## Build

```bash
npm run build
npm start
```
