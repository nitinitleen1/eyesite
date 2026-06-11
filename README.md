# Eyesite

<div align="center">

**Open-Source Observability for AI Agents**

[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.21+-00ADD8?logo=go)](https://go.dev/)
[![Next.js](https://img.shields.io/badge/next.js-15-black?logo=next.js)](https://nextjs.org/)

Track every AI interaction. Optimize costs. Build better agents.

[Quick Start](#-quick-start) • [Features](#-features) • [Architecture](#%EF%B8%8F-architecture) • [Roadmap](#-roadmap) • [Contributing](#-contributing)

</div>

---

## 📊 Status

**MVP complete** (June 2026). Core platform works end-to-end: register → create API key → instrument your agent with the SDK → see sessions, costs, and analytics in the dashboard.

This project is **vibe-coded** — rapidly built with AI assistance and reviewed/hardened iteratively. It is suitable for self-hosted development and alpha use, not yet battle-tested for production. See [TASKS.md](TASKS.md) for the full task list and [Roadmap](#-roadmap) for what's next.

## ✨ Features

### Working today

- **Multi-provider cost tracking** — OpenAI, Anthropic, Google Gemini pricing built in; cost computed per interaction at ingest time
- **TypeScript SDK** — wrap your OpenAI/Anthropic client in two lines; every call is tracked automatically ([sdk/](sdk/))
- **Sessions** — group interactions per agent run; drill into any session from the dashboard
- **Analytics dashboard** — totals, provider/model breakdown, cost over time, token usage (Next.js + Tailwind)
- **Search & export** — full-text search across prompts/responses; CSV/JSON export
- **Multi-tenant workspaces** — workspace isolation enforced on every query; owner/admin/developer/viewer roles
- **API keys** — scoped per workspace, SHA-256 hashed at rest, revocable, expiry + last-used tracking; enforced on all ingest endpoints
- **OpenTelemetry ingestion** — trace/span endpoints storing into a dedicated telemetry schema
- **JWT auth** — email/password registration and login; all dashboard APIs require a valid token

### Provider support

| Provider | Status | Models priced |
|----------|--------|---------------|
| OpenAI | ✅ Ready | GPT-4, GPT-4 Turbo, GPT-3.5 Turbo |
| Anthropic | ✅ Ready | Claude 3 Opus, Sonnet, Haiku |
| Google | ✅ Ready | Gemini Pro, Gemini Pro Vision |
| Others | 📋 Planned | Azure OpenAI, AWS Bedrock, Ollama |

## 🚀 Quick Start

### Prerequisites

- **Go 1.21+**, **Node.js 20+**, **Docker** + **Docker Compose**

### Run the platform

```bash
git clone https://github.com/nitinitleen1/eyesite.git
cd eyesite

# 1. Start infrastructure (PostgreSQL, Redis, ClickHouse, Adminer)
make docker-up

# 2. Create the database schema
make migrate

# 3. Start the three backend services (separate terminals)
make dev-auth     # auth service    → :8000
make dev-ingest   # ingest service  → :8001
make dev-query    # query service   → :8002

# 4. Start the frontend
cd frontend && npm install && npm run dev   # → :3000
```

Open http://localhost:3000, register an account (a default workspace is created automatically), then create an API key in **Dashboard → API Keys**.

### Instrument your agent

```typescript
import { Eyesite } from '@eyesite/sdk';
import OpenAI from 'openai';

const eyesite = new Eyesite({ apiKey: process.env.EYESITE_API_KEY! });
const openai = new OpenAI({ apiKey: process.env.OPENAI_API_KEY });

eyesite.wrapOpenAI(openai);                    // every call now tracked
await eyesite.createSession('My agent run');   // optional: group calls

const response = await openai.chat.completions.create({
  model: 'gpt-4',
  messages: [{ role: 'user', content: 'Hello!' }],
});
```

See [sdk/README.md](sdk/README.md) for Anthropic wrapping, manual tracking, and configuration. The SDK is not yet published to npm — install it from `sdk/` locally.

## 🏗️ Architecture

Three Go microservices over PostgreSQL, with a Next.js frontend:

```
┌──────────────────────────────────────────────┐
│          Frontend (Next.js)  :3000           │
│        React, TypeScript, Tailwind           │
└──────┬──────────────────────────┬────────────┘
       │ JWT                      │ JWT
┌──────▼───────┐          ┌───────▼────────┐
│ Auth Service │          │ Query Service  │
│    :8000     │          │     :8002      │
│ users, work- │          │ analytics,     │
│ spaces, keys │          │ sessions,      │
└──────┬───────┘          │ search, export │
       │                  └───────┬────────┘
       │   ┌──────────────┐       │
 SDK ──┼──▶│Ingest Service│       │
 X-API │   │    :8001     │       │
 -Key  │   │ interactions,│       │
       │   │ OTel traces  │       │
       │   └──────┬───────┘       │
┌──────▼──────────▼───────────────▼────────────┐
│                PostgreSQL                     │
│   main (users/workspaces/keys)                │
│   spotlight (sessions/interactions)           │
│   telemetry (traces/spans)                    │
└───────────────────────────────────────────────┘
```

**Auth model:** browsers authenticate with JWT (auth + query services); SDKs authenticate with `X-API-Key` (ingest service). The ingest service derives the workspace from the key — a key can only write to its own workspace. Redis and ClickHouse run in Docker but are not yet on the hot path (see Roadmap).

### Project structure

```
eyesite/
├── backend/
│   ├── cmd/                # auth-service, ingest-service, query-service
│   ├── pkg/                # config, database, models, providers
│   └── internal/errors/    # standardized error types
├── frontend/src/
│   ├── app/                # landing, login, register, dashboard
│   ├── contexts/           # auth context
│   └── lib/                # API client
├── sdk/                    # TypeScript SDK (@eyesite/sdk)
├── database/schema.sql     # full PostgreSQL schema
├── docker-compose.yml      # PostgreSQL, Redis, ClickHouse, Adminer
└── Makefile                # dev commands (run `make help`)
```

## 🗺️ Roadmap

- **OAuth** — Google/GitHub login (routes exist, return 501)
- **Refresh tokens** — real rotation (currently mirrors the access token)
- **Tests + CI** — unit/integration tests, GitHub Actions
- **Member invitations** — invite users to workspaces (roles already enforced)
- **API key rotation**
- **Real-time dashboards** — WebSocket updates
- **ClickHouse pipeline** — high-volume analytics
- **More providers** — Azure OpenAI, Bedrock, Ollama; tiktoken-based counting
- **Alerting** — cost/error thresholds, Slack/email/webhooks
- **npm publish** of the SDK

## 🛠️ Development

```bash
make help        # all commands
make build       # build the three services into ./bin
make test        # go test ./...
make lint        # golangci-lint
```

Branching: active development happens on `develop`; `main` holds stable releases. See [BRANCHING.md](BRANCHING.md).

## 🤝 Contributing

We're building in public — contributions welcome! See [CONTRIBUTING.md](CONTRIBUTING.md).

1. Fork the repository
2. Create a feature branch from `develop` (`git checkout -b feature/amazing-feature`)
3. Commit your changes
4. Open a Pull Request against `develop`

## 📜 License

Apache License 2.0 — see [LICENSE](LICENSE).

## 🙏 Acknowledgments

Inspired by [Shinzo](https://github.com/shinzo-project/shinzo). Built with [Go](https://go.dev/), [Next.js](https://nextjs.org/), [OpenTelemetry](https://opentelemetry.io/), and [PostgreSQL](https://www.postgresql.org/).

## 📞 Support

- **Issues**: [GitHub Issues](https://github.com/nitinitleen1/eyesite/issues)
- **Discussions**: [GitHub Discussions](https://github.com/nitinitleen1/eyesite/discussions)

---

**Eyesite** — See everything your AI agents do. 👁️
