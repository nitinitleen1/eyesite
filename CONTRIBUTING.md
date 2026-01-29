# Contributing to Agent Observability Platform

Thank you for your interest in contributing! This document provides guidelines and instructions for contributing to the project.

## 🎯 Ways to Contribute

- **Report Bugs**: File detailed bug reports with reproduction steps
- **Suggest Features**: Propose new features or improvements
- **Write Code**: Submit pull requests for bug fixes or features
- **Improve Documentation**: Help make our docs better
- **Answer Questions**: Help other users in discussions

## 🚀 Getting Started

### Prerequisites

- Go 1.21 or higher
- Node.js 20 or higher
- Docker and Docker Compose
- Git

### Development Setup

1. **Fork and clone the repository**

```bash
git clone https://github.com/YOUR_USERNAME/agent-observability.git
cd agent-observability
```

2. **Start development environment**

```bash
# Start databases
docker-compose up -d

# Backend setup
cd backend
go mod download
make dev

# Frontend setup (in another terminal)
cd frontend
npm install
npm run dev
```

3. **Create a feature branch**

```bash
git checkout -b feature/your-feature-name
```

## 📝 Code Guidelines

### Go Code Style

- Follow standard Go conventions and idioms
- Use `gofmt` for formatting
- Run `golangci-lint` before committing
- Write tests for new functionality
- Add comments for exported functions and complex logic

**Example:**

```go
// CalculateCost computes the total cost for an LLM interaction
// based on the provider's pricing model and token usage.
func CalculateCost(provider string, model string, tokens TokenCount) (float64, error) {
    pricing, err := getPricing(provider, model)
    if err != nil {
        return 0, fmt.Errorf("failed to get pricing: %w", err)
    }

    inputCost := float64(tokens.Input) * pricing.InputPricePerToken
    outputCost := float64(tokens.Output) * pricing.OutputPricePerToken
    
    return inputCost + outputCost, nil
}
```

### TypeScript/React Code Style

- Use TypeScript strict mode
- Follow Airbnb React/TypeScript style guide
- Use functional components and hooks
- Write type-safe code
- Add JSDoc comments for complex functions

**Example:**

```typescript
/**
 * Fetches trace data from the API with automatic retry logic.
 * 
 * @param traceId - The unique identifier for the trace
 * @param options - Fetch options including retry configuration
 * @returns Promise resolving to the trace data
 * @throws {APIError} When the API returns an error after all retries
 */
export async function fetchTrace(
  traceId: string,
  options?: FetchOptions
): Promise<Trace> {
  const response = await apiClient.get<Trace>(`/traces/${traceId}`, {
    retry: options?.retry ?? 3,
  });
  
  return response.data;
}
```

### Error Handling

**Go:**
```go
// Always return errors, don't panic in library code
func ProcessData(data []byte) (*Result, error) {
    if len(data) == 0 {
        return nil, errors.New("data cannot be empty")
    }
    
    result, err := parseData(data)
    if err != nil {
        return nil, fmt.Errorf("parsing failed: %w", err)
    }
    
    return result, nil
}
```

**TypeScript:**
```typescript
// Use try-catch for async operations
async function processData(data: unknown): Promise<Result> {
  try {
    const validated = schema.parse(data);
    return await transformData(validated);
  } catch (error) {
    if (error instanceof ZodError) {
      throw new ValidationError('Invalid data format', error);
    }
    throw new ProcessingError('Failed to process data', { cause: error });
  }
}
```

### Testing

- Write unit tests for all new functions
- Aim for 80%+ code coverage
- Use table-driven tests in Go
- Use Jest and React Testing Library for frontend

**Go Test Example:**
```go
func TestCalculateCost(t *testing.T) {
    tests := []struct {
        name     string
        provider string
        model    string
        tokens   TokenCount
        want     float64
        wantErr  bool
    }{
        {
            name:     "OpenAI GPT-4",
            provider: "openai",
            model:    "gpt-4",
            tokens:   TokenCount{Input: 100, Output: 50},
            want:     0.0075, // $0.03/1K input + $0.06/1K output
            wantErr:  false,
        },
        // Add more test cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := CalculateCost(tt.provider, tt.model, tt.tokens)
            if (err != nil) != tt.wantErr {
                t.Errorf("CalculateCost() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if math.Abs(got-tt.want) > 0.0001 {
                t.Errorf("CalculateCost() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

## 🔄 Pull Request Process

1. **Update your fork**
   ```bash
   git remote add upstream https://github.com/original/agent-observability.git
   git fetch upstream
   git rebase upstream/main
   ```

2. **Make your changes**
   - Write clean, well-documented code
   - Follow the style guides above
   - Add tests for new functionality
   - Update documentation as needed

3. **Test thoroughly**
   ```bash
   # Backend tests
   cd backend && make test
   
   # Frontend tests
   cd frontend && npm test
   
   # Linting
   make lint
   ```

4. **Commit with clear messages**
   ```bash
   git commit -m "feat: add cost forecasting for multi-agent workflows
   
   - Implement ML-based cost prediction
   - Add API endpoint for forecast data
   - Include tests for edge cases"
   ```

   Follow [Conventional Commits](https://www.conventionalcommits.org/):
   - `feat:` New feature
   - `fix:` Bug fix
   - `docs:` Documentation changes
   - `style:` Code style changes
   - `refactor:` Code refactoring
   - `test:` Adding tests
   - `chore:` Maintenance tasks

5. **Push and create PR**
   ```bash
   git push origin feature/your-feature-name
   ```
   
   Then create a PR on GitHub with:
   - Clear title and description
   - Link to related issues
   - Screenshots (for UI changes)
   - Breaking changes highlighted

6. **Address review feedback**
   - Respond to comments
   - Make requested changes
   - Push updates to your branch

## 🐛 Bug Reports

When filing a bug report, include:

1. **Description**: What happened vs. what you expected
2. **Steps to Reproduce**: Detailed steps to recreate the issue
3. **Environment**: OS, Go version, Node version, etc.
4. **Logs**: Relevant error messages or stack traces
5. **Screenshots**: If applicable

## 💡 Feature Requests

When proposing a feature:

1. **Use Case**: Describe the problem it solves
2. **Proposed Solution**: How you envision it working
3. **Alternatives**: Other approaches considered
4. **Examples**: Similar features in other tools

## 📜 Code of Conduct

- Be respectful and inclusive
- Welcome newcomers
- Focus on constructive feedback
- Assume good intentions

## 📄 License

By contributing, you agree that your contributions will be licensed under the Apache License 2.0.

## 🙏 Recognition

Contributors will be recognized in:
- README.md contributors section
- Release notes
- Project documentation

Thank you for contributing to Agent Observability Platform! 🎉
