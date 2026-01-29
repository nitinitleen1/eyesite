# @eyesite/sdk

TypeScript/JavaScript SDK for the Eyesite Eyesite.

## Installation

```bash
npm install @eyesite/sdk
# or
yarn add @eyesite/sdk
# or
pnpm add @eyesite/sdk
```

## Quick Start

```typescript
import { Eyesite } from '@eyesite/sdk';

// Initialize the SDK
const eyesite = new Eyesite({
  apiKey: process.env.EYESITE_API_KEY!,
  endpoint: 'http://localhost:8001', // Optional, defaults to localhost
  workspaceUuid: 'your-workspace-uuid',
});

// Create a session
const sessionId = await eyesite.createSession('My Agent Session');

// Record an interaction manually
await eyesite.recordInteraction({
  provider: 'openai',
  model: 'gpt-4',
  prompt: 'Hello, world!',
  response: 'Hi there! How can I help you today?',
  promptTokens: 5,
  responseTokens: 12,
  latencyMs: 324,
  status: 'success',
});
```

## OpenAI Integration

Automatically track all OpenAI API calls:

```typescript
import { Eyesite } from '@eyesite/sdk';
import OpenAI from 'openai';

const eyesite = new Eyesite({
  apiKey: process.env.EYESITE_API_KEY!,
  workspaceUuid: 'your-workspace-uuid',
});

const openai = new OpenAI({
  apiKey: process.env.OPENAI_API_KEY,
});

// Wrap the  OpenAI client
eyesite.wrapOpenAI(openai);

// Now all calls are automatically tracked!
const response = await openai.chat.completions.create({
  model: 'gpt-4',
  messages: [{ role: 'user', content: 'Hello!' }],
});
```

## Anthropic Integration

Automatically track all Anthropic API calls:

```typescript
import { Eyesite } from '@eyesite/sdk';
import Anthropic from '@anthropic-ai/sdk';

const eyesite = new Eyesite({
  apiKey: process.env.EYESITE_API_KEY!,
  workspaceUuid: 'your-workspace-uuid',
});

const anthropic = new Anthropic({
  apiKey: process.env.ANTHROPIC_API_KEY,
});

// Wrap the Anthropic client
eyesite.wrapAnthropic(anthropic);

// Now all calls are automatically tracked!
const response = await anthropic.messages.create({
  model: 'claude-3-opus-20240229',
  max_tokens: 1024,
  messages: [{ role: 'user', content: 'Hello!' }],
});
```

## Manual Tracking

Track any function with automatic timing:

```typescript
await eyesite.track(
  'my-operation',
  async () => {
    // Your code here
    return await someApiCall();
  },
  {
    provider: 'custom',
    model: 'my-model',
    metadata: { userId: '123' },
  }
);
```

## Configuration

```typescript
interface EyesiteConfig {
  apiKey: string; // Required: Your API key from Eyesite dashboard
  endpoint?: string; // Optional: API endpoint (default: http://localhost:8001)
  workspaceUuid?: string; // Optional: Workspace UUID
  sessionUuid?: string; // Optional: Session UUID
  autoCapture?: boolean; // Optional: Auto-capture wrapped calls (default: true)
}
```

## Features

- ✅ **Automatic Tracking** - Wrap OpenAI/Anthropic clients for zero-code instrumentation
- ✅ **Manual Recording** - Record interactions with full control
- ✅ **Session Management** - Group interactions into sessions
- ✅ **Cost Tracking** - Automatic cost calculation based on tokens
- ✅ **Error Handling** - Track both successes and failures
- ✅ **TypeScript** - Full type safety
- ✅ **Zero Dependencies** - Lightweight and fast

## API Reference

### `new Eyesite(config)`

Create a new SDK instance.

### `eyesite.createSession(name?): Promise<string>`

Create a new session and return its UUID.

### `eyesite.recordInteraction(data): Promise<void>`

Manually record an LLM interaction.

### `eyesite.track(name, fn, options?): Promise<T>`

Track a function execution with automatic timing.

### `eyesite.wrapOpenAI(openai)`

Wrap an OpenAI client for automatic tracking.

### `eyesite.wrapAnthropic(anthropic)`

Wrap an Anthropic client for automatic tracking.

## License

Apache-2.0

## Support

- GitHub: https://github.com/nitinitleen1/eyesite
- Issues: https://github.com/nitinitleen1/eyesite/issues
