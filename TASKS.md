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

- [ ] **Query Service**
  - [ ] Session analytics endpoints
  - [ ] Interaction search/filter
  - [ ] Cost aggregation queries
  - [ ] Export functionality (CSV, JSON)

### 🟡 Medium Priority - Enhancement

- [x] **TypeScript SDK**
  - [x] Create npm package structure
  - [x] Implement OpenTelemetry wrapper
  - [x] Add provider proxying (OpenAI, Anthropic wrappers)
  - [x] Create usage examples
  - [x] Write SDK documentation
  - [ ] Publish to npm (when ready)

- [ ] **API Key Management (Complete)**
  - [ ] Implement ListAPIKeys handler
  - [ ] Implement CreateAPIKey handler
  - [ ] Implement RevokeAPIKey handler
  - [ ] Add API key rotation
  - [ ] API key usage tracking

- [ ] **Workspace Management (Complete)**
  - [ ] ListWorkspaces handler
  - [ ] CreateWorkspace handler
  - [ ] UpdateWorkspace handler
  - [ ] DeleteWorkspace handler
  - [ ] Member invitation system
  - [ ] Role-based permissions

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
- [ ] Query service (Next)
- [ ] Complete workspace/API key management (Next)

**Completion: 75%**

---

## 🎯 Next 3 Tasks (Immediate)

1. **Create Telemetry Ingestion Service** - Start accepting OpenTelemetry data
2. **Initialize Next.js Frontend** - Set up authentication and layout
3. **Build Session Dashboard** - First visual interface for users

---

*Updated: 2026-01-29 21:30*
