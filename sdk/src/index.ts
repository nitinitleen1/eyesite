/**
 * Eyesite SDK - TypeScript/JavaScript SDK for Agent Observability
 */

export interface EyesiteConfig {
    apiKey: string;
    endpoint?: string;
    workspaceUuid?: string;
    sessionUuid?: string;
    autoCapture?: boolean;
}

export interface InteractionData {
    provider: string;
    model: string;
    prompt: string;
    response: string;
    promptTokens?: number;
    responseTokens?: number;
    latencyMs?: number;
    status?: 'success' | 'error';
    errorMessage?: string;
    metadata?: Record<string, any>;
}

export interface TrackingOptions {
    provider?: string;
    model?: string;
    user?: string;
    metadata?: Record<string, any>;
}

export class Eyesite {
    private config: Required<EyesiteConfig>;
    private sessionUuid: string | null = null;

    constructor(config: EyesiteConfig) {
        this.config = {
            endpoint: config.endpoint || 'http://localhost:8001',
            workspaceUuid: config.workspaceUuid || '',
            sessionUuid: config.sessionUuid || '',
            autoCapture: config.autoCapture ?? true,
            apiKey: config.apiKey,
        };
        this.sessionUuid = config.sessionUuid || null;
    }

    /**
     * Create a new session
     */
    async createSession(name?: string): Promise<string> {
        const response = await fetch(`${this.config.endpoint}/v1/sessions`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'X-API-Key': this.config.apiKey,
            },
            body: JSON.stringify({ name }),
        });

        if (!response.ok) {
            throw new Error(`Failed to create session: ${await response.text()}`);
        }

        const session = await response.json();
        this.sessionUuid = session.uuid;
        return session.uuid;
    }

    /**
     * Record an LLM interaction
     */
    async recordInteraction(data: InteractionData): Promise<void> {
        try {
            const response = await fetch(`${this.config.endpoint}/v1/interactions`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-API-Key': this.config.apiKey,
                },
                body: JSON.stringify({
                    workspace_uuid: this.config.workspaceUuid,
                    session_uuid: this.sessionUuid,
                    ...data,
                    prompt_tokens: data.promptTokens,
                    response_tokens: data.responseTokens,
                    latency_ms: data.latencyMs,
                    error_message: data.errorMessage,
                }),
            });

            if (!response.ok) {
                console.error('Failed to record interaction:', await response.text());
            }
        } catch (error) {
            console.error('Error recording interaction:', error);
        }
    }

    /**
     * Track a function call with automatic instrumentation
     */
    async track<T>(
        name: string,
        fn: () => Promise<T>,
        options?: TrackingOptions
    ): Promise<T> {
        const startTime = Date.now();
        let result: T;
        let error: Error | null = null;

        try {
            result = await fn();
            return result;
        } catch (err) {
            error = err as Error;
            throw err;
        } finally {
            const latencyMs = Date.now() - startTime;

            // Auto-capture if enabled and we can infer the data
            if (this.config.autoCapture && options) {
                await this.recordInteraction({
                    provider: options.provider || 'unknown',
                    model: options.model || 'unknown',
                    prompt: '', // Would need to be extracted from fn
                    response: '', // Would need to be extracted from result
                    latencyMs,
                    status: error ? 'error' : 'success',
                    errorMessage: error?.message,
                    metadata: options.metadata,
                });
            }
        }
    }

    /**
     * Wrap OpenAI client for automatic tracking
     */
    wrapOpenAI(openai: any) {
        const self = this;
        const originalCreate = openai.chat.completions.create.bind(
            openai.chat.completions
        );

        openai.chat.completions.create = async function (params: any) {
            const startTime = Date.now();
            try {
                const result = await originalCreate(params);
                const latencyMs = Date.now() - startTime;

                await self.recordInteraction({
                    provider: 'openai',
                    model: params.model,
                    prompt: JSON.stringify(params.messages),
                    response: result.choices[0]?.message?.content || '',
                    promptTokens: result.usage?.prompt_tokens,
                    responseTokens: result.usage?.completion_tokens,
                    latencyMs,
                    status: 'success',
                });

                return result;
            } catch (error: any) {
                const latencyMs = Date.now() - startTime;
                await self.recordInteraction({
                    provider: 'openai',
                    model: params.model,
                    prompt: JSON.stringify(params.messages),
                    response: '',
                    latencyMs,
                    status: 'error',
                    errorMessage: error.message,
                });
                throw error;
            }
        };

        return openai;
    }

    /**
     * Wrap Anthropic client for automatic tracking
     */
    wrapAnthropic(anthropic: any) {
        const self = this;
        const originalCreate = anthropic.messages.create.bind(anthropic.messages);

        anthropic.messages.create = async function (params: any) {
            const startTime = Date.now();
            try {
                const result = await originalCreate(params);
                const latencyMs = Date.now() - startTime;

                await self.recordInteraction({
                    provider: 'anthropic',
                    model: params.model,
                    prompt: JSON.stringify(params.messages),
                    response: result.content[0]?.text || '',
                    promptTokens: result.usage?.input_tokens,
                    responseTokens: result.usage?.output_tokens,
                    latencyMs,
                    status: 'success',
                });

                return result;
            } catch (error: any) {
                const latencyMs = Date.now() - startTime;
                await self.recordInteraction({
                    provider: 'anthropic',
                    model: params.model,
                    prompt: JSON.stringify(params.messages),
                    response: '',
                    latencyMs,
                    status: 'error',
                    errorMessage: error.message,
                });
                throw error;
            }
        };

        return anthropic;
    }
}

// Default export
export default Eyesite;
