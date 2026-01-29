# Agent Observability Platform

<div align="center">

**The Universal Observability Platform for AI Agents**

[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.21+-00ADD8?logo=go)](https://go.dev/)
[![Next.js](https://img.shields.io/badge/next.js-14+-black?logo=next.js)](https://nextjs.org/)

Track every AI interaction. Optimize costs. Build better agents.

[Documentation](#documentation) • [Quick Start](#quick-start) • [Features](#features) • [Contributing](#contributing)

</div>

---

## 🎯 Overview

Agent Observability Platform is an **open-source, production-ready observability platform** specifically designed for AI agents and LLM applications. It provides comprehensive monitoring, analytics, and optimization tools for modern AI systems.

### Why This Platform?

- 🌐 **Universal Provider Support** - Works with OpenAI, Anthropic, Google, Azure, AWS Bedrock, and more
- 🤖 **Multi-Agent Tracking** - Visualize complex agent workflows and hierarchies
- 💰 **Intelligent Cost Management** - Track, forecast, and optimize AI spending
- 📊 **Deep Analytics** - A/B testing, custom metrics, and business insights
- 🔒 **Enterprise Security** - PII detection, compliance tools, audit logs
- 🔌 **Extensible** - Plugin system for custom integrations
- 🚀 **Production Ready** - Built with Go for performance and reliability

## ✨ Features

### Core Capabilities

- **OpenTelemetry Compliant** - Standards-based telemetry ingestion
- **Multi-Tenant Architecture** - Secure workspace isolation
- **Real-Time Dashboards** - Live metrics and beautiful visualizations
- **Cost Attribution** - Track spending by project, agent, or user
- **Advanced Alerting** - Multi-channel notifications (Slack, email, webhooks)
- **Full-Text Search** - Search across all prompts and responses
- **Team Collaboration** - RBAC, shared dashboards, comments

### Supported Providers

| Provider | Status | Models |
|----------|--------|--------|
| OpenAI | ✅ Ready | GPT-4, GPT-3.5, Embeddings |
| Anthropic | ✅ Ready | Claude 3 (Opus, Sonnet, Haiku) |
| Google | ✅ Ready | Gemini Pro, Gemini Ultra |
| Azure OpenAI | 🚧 Coming Soon | All Azure OpenAI models |
| AWS Bedrock | 🚧 Coming Soon | Claude, Llama, Titan |
| Local Models | 🚧 Coming Soon | Ollama, LM Studio |

### Framework Integrations

- **LangChain** - Native instrumentation
- **LlamaIndex** - Query engine tracking
- **CrewAI** - Multi-agent workflows
- **AutoGPT** - Goal-based agents
- **Custom** - OpenTelemetry SDK

## 🚀 Quick Start

### Prerequisites

- **Go 1.21+** for backend services
- **Node.js 20+** for frontend
- **Docker** and **Docker Compose** for local development
- **PostgreSQL 16+** (or use Docker)

### Installation

```bash
# Clone the repository
git clone https://github.com/yourusername/agent-observability.git
cd agent-observability

# Start infrastructure (PostgreSQL, Redis, ClickHouse)
docker-compose up -d

# Start backend services
cd backend
go mod download
make run-services

# Start frontend (in another terminal)
cd frontend
npm install
npm run dev
```

The platform will be available at:
- **Frontend**: http://localhost:3000
- **API**: http://localhost:8000
- **Telemetry Ingestion**: http://localhost:8000/v1/traces

### Using the SDK

**JavaScript/TypeScript:**

```typescript
import { AgentObs } from '@agent-obs/sdk-js';

const obs = new AgentObs({
  apiKey: process.env.AGENT_OBS_API_KEY,
  endpoint: 'http://localhost:8000',
});

// Track an LLM call
await obs.track('chat-completion', async () => {
  const response = await openai.chat.completions.create({
    model: 'gpt-4',
    messages: [{ role: 'user', content: 'Hello!' }],
  });
  return response;
}, {
  provider: 'openai',
  model: 'gpt-4',
  user: 'user-123',
});
```

**Python:**

```python
from agent_obs import AgentObs

obs = AgentObs(
    api_key=os.getenv("AGENT_OBS_API_KEY"),
    endpoint="http://localhost:8000"
)

# Track an LLM call
with obs.track("chat-completion", provider="openai", model="gpt-4"):
    response = openai.ChatCompletion.create(
        model="gpt-4",
        messages=[{"role": "user", "content": "Hello!"}]
    )
```

## 📖 Documentation

- [Architecture Overview](docs/architecture.md)
- [API Reference](docs/api-reference.md)
- [SDK Documentation](docs/sdk-guide.md)
- [Self-Hosting Guide](docs/self-hosting.md)
- [Contributing Guidelines](CONTRIBUTING.md)

## 🏗️ Architecture

```
┌─────────────────────────────────────────┐
│         Frontend (Next.js)              │
│    React, TypeScript, Tailwind          │
└─────────────────┬───────────────────────┘
                  │
┌─────────────────▼───────────────────────┐
│         API Gateway (Go)                │
│    Rate Limiting, Auth, Routing         │
└─────────────────┬───────────────────────┘
                  │
      ┌───────────┼───────────┐
      │           │           │
┌─────▼──────┐┌──▼─────┐┌───▼────────┐
│Auth Service││Ingest  ││Query       │
│   (Go)     ││Service ││Service     │
│            ││  (Go)  ││  (Go)      │
└─────┬──────┘└──┬─────┘└───┬────────┘
      │          │           │
      └──────────┼───────────┘
                 │
┌────────────────▼────────────────────────┐
│        Data Layer                       │
│  ┌──────────┐  ┌────────┐  ┌────────┐  │
│  │PostgreSQL│  │ Click- │  │ Redis  │  │
│  │          │  │ House  │  │        │  │
│  └──────────┘  └────────┘  └────────┘  │
└─────────────────────────────────────────┘
```

## 🛠️ Development

### Project Structure

```
agent-observability/
├── backend/                 # Go backend services
│   ├── cmd/                # Service entry points
│   ├── pkg/                # Shared packages
│   └── internal/           # Internal utilities
├── frontend/               # Next.js frontend
│   ├── app/               # App Router pages
│   ├── components/        # React components
│   └── lib/               # Utilities
├── database/              # Database schemas
│   ├── migrations/        # SQL migrations
│   └── seeds/            # Seed data
├── docker/               # Docker configurations
├── docs/                 # Documentation
└── scripts/              # Build and deployment scripts
```

### Running Tests

```bash
# Backend tests
cd backend
make test

# Frontend tests
cd frontend
npm test

# E2E tests
npm run test:e2e
```

### Building for Production

```bash
# Build all services
make build

# Build Docker images
make docker-build

# Deploy to Kubernetes
kubectl apply -f k8s/
```

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guidelines](CONTRIBUTING.md) for details.

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📜 License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

Inspired by [Shinzo](https://github.com/shinzo-project/shinzo) - An excellent OpenTelemetry-based observability platform.

Built with:
- [OpenTelemetry](https://opentelemetry.io/) - Observability framework
- [Go](https://go.dev/) - Backend services
- [Next.js](https://nextjs.org/) - Frontend framework
- [ClickHouse](https://clickhouse.com/) - Analytics database
- [Radix UI](https://www.radix-ui.com/) - UI components

## 📞 Support

- **Documentation**: [docs/](docs/)
- **Issues**: [GitHub Issues](https://github.com/yourusername/agent-observability/issues)
- **Discussions**: [GitHub Discussions](https://github.com/yourusername/agent-observability/discussions)
- **Discord**: [Join our community](https://discord.gg/agent-obs)

---

Made with ❤️ for the AI developer community
