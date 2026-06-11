# Eyesite - Task List

## 🎯 Current Sprint: Core Services & Frontend

### 🔴 High Priority - Core Functionality

- [x] **Telemetry Ingestion Service**
  - [x] Create OpenTelemetry HTTP endpoint
  - [x] Implement span/trace parsing (basic)
  - [x] Store traces and spans in database
  - [x] Link spans to interactions  
  - [x] Add cost calculation on ingestion
  - [x] Health check endpoint

- [x] **Frontend Foundation (Next.js)**
  - [x] Initialize Next.js 14+ with TypeScript
  - [x] Set up Tailwind CSS
  - [x] Create authentication pages (login, register)
  - [x] Implement API client with fetch
  - [x] Add authentication context/provider
  - [x] Protected route wrapper
  - [x] Layout components

- [x] **Dashboard - Phase 1**
  - [x] Dashboard shell/layout
  - [x] Session list view (empty state)
  - [x] Interaction timeline (empty state)
  - [x] Cost summary widget
  - [x] Token usage charts (empty state)
  - [x] Provider distribution chart (empty state)

- [x] **Query Service**
  - [x] Session analytics endpoints
  - [x] Interaction search/filter
  - [x] Cost aggregation queries
  - [x] Export functionality (CSV, JSON)
  - [x] JWT auth + workspace membership checks on all endpoints

### 🟡 Medium Priority - Enhancement

- [x] **TypeScript SDK**
  - [x] Create npm package structure
  - [x] Implement OpenTelemetry wrapper
  - [x] Add provider proxying (OpenAI, Anthropic wrappers)
  - [x] Create usage examples
  - [x] Write SDK documentation
  - [ ] Publish to npm (when ready)

- [x] **API Key Management (Complete)**
  - [x] Implement ListAPIKeys handler
  - [x] Implement CreateAPIKey handler
  - [x] Implement RevokeAPIKey handler
  - [ ] Add API key rotation
  - [x] API key usage tracking (last_used_at on ingest)
  - [x] API key enforcement on ingest service (X-API-Key, revocation/expiry honored)

- [x] **Workspace Management (Complete)**
  - [x] ListWorkspaces handler
  - [x] CreateWorkspace handler
  - [x] UpdateWorkspace handler
  - [x] DeleteWorkspace handler
  - [ ] Member invitation system
  - [x] Role-based permissions (owner/admin checks on update, delete, key revoke)

- [ ] **Provider Enhancements**
  - [ ] Add more providers (Cohere, Mistral, etc.)
  - [ ] Better token counting (integrate tiktoken)
  - [ ] Streaming support
  - [ ] Rate limiting per provider
  - [ ] Provider health checks

### 🟢 Low Priority - Polish & Growth

- [ ] **GitHub Repository Enhancements**
  - [ ] Add repository topics/tags
  - [ ] Create social preview image
  - [ ] Add README badges (build, coverage, license)
  - [ ] Set up GitHub Actions CI/CD
  - [ ] Add issue templates
  - [ ] Add PR templates
  - [ ] Create CHANGELOG.md

- [ ] **GitHub Actions CI/CD**
  - [ ] Workflow for Go tests
  - [ ] Workflow for linting
  - [ ] Workflow for building Docker images
  - [ ] Workflow for frontend tests
  - [ ] Deployment workflow
  - [ ] Release automation

- [ ] **Documentation Site**
  - [ ] Set up documentation framework (Docusaurus/VitePress)
  - [ ] API documentation
  - [ ] SDK guides
  - [ ] Architecture diagrams
  - [ ] Getting started guide
  - [ ] Deploy to GitHub Pages

- [ ] **OAuth Implementation**
  - [ ] Complete Google OAuth flow
  - [ ] Complete GitHub OAuth flow
  - [ ] Add OAuth user linking
  - [ ] OAuth token refresh

- [ ] **Testing**
  - [ ] Unit tests for all packages
  - [ ] Integration tests for services
  - [ ] E2E tests for critical flows
  - [ ] Load testing
  - [ ] CI test coverage reporting

### 🔵 Future Features - Backlog

- [ ] **Advanced Analytics**
  - [ ] Cost forecasting with ML
  - [ ] Anomaly detection
  - [ ] Performance recommendations
  - [ ] Usage patterns analysis

- [ ] **Alerting System**
  - [ ] Cost threshold alerts
  - [ ] Error rate alerts
  - [ ] Custom alert rules
  - [ ] Notification channels (email, Slack, webhook)

- [ ] **Multi-Agent Tracking**
  - [ ] Agent orchestration visualization
  - [ ] Agent communication graphs
  - [ ] Cross-agent performance metrics

- [ ] **ClickHouse Integration**
  - [ ] Set up ClickHouse schema
  - [ ] Data pipeline from PostgreSQL
  - [ ] High-performance analytics queries
  - [ ] Real-time dashboards

- [ ] **Proxy Service**
  - [ ] Provider request proxying
  - [ ] Automatic telemetry injection
  - [ ] Request/response logging
  - [ ] Cost tracking middleware

- [ ] **Enterprise Features**
  - [ ] SSO support (SAML, OIDC)
  - [ ] Advanced RBAC
  - [ ] Audit logging
  - [ ] Data retention policies
  - [ ] Multi-region support

---

## 📊 Progress Tracking

**Sprint 1 (Current - MVP Foundation)**
- [x] Backend infrastructure ✅
- [x] Authentication service ✅
- [x] Database schema ✅
- [x] Multi-provider support ✅
- [x] Docker setup ✅
- [x] Telemetry ingestion ✅
- [x] Frontend foundation ✅
- [x] Basic dashboard ✅
- [x] TypeScript SDK ✅
- [x] Query service ✅
- [x] Complete workspace/API key management ✅
- [x] Dashboard wired to real workspaces (sessions, API keys, settings UI) ✅
- [x] Sessions end-to-end (SDK createSession → ingest → dashboard) ✅

**MVP complete.** Remaining work is post-MVP (OAuth, refresh tokens, WebSockets, member invitations, API key rotation, tests, CI/CD).

---

## 🎯 Next 3 Tasks (Post-MVP)

1. **OAuth flows** - Google/GitHub login (handlers are routed but return 501)
2. **Refresh tokens** - Real refresh token issuance/rotation (currently mirrors the access token)
3. **Tests + CI** - Unit/integration tests and GitHub Actions

---

*Updated: 2026-06-12*
