// API client configuration
const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8000';
const QUERY_BASE_URL = process.env.NEXT_PUBLIC_QUERY_URL || 'http://localhost:8002';
const INGEST_BASE_URL = process.env.NEXT_PUBLIC_INGEST_URL || 'http://localhost:8001';

export interface AuthResponse {
    token: string;
    refresh_token: string;
    expires_at: string;
    user: {
        uuid: string;
        email: string;
        full_name: string;
        is_verified: boolean;
        created_at: string;
    };
}

export interface Workspace {
    uuid: string;
    name: string;
    owner_uuid: string;
    slug: string;
    created_at: string;
    updated_at: string;
}

export interface ApiKey {
    uuid: string;
    user_uuid: string;
    workspace_uuid: string;
    name: string;
    key_preview: string;
    scopes: string[];
    last_used_at?: string;
    expires_at?: string;
    created_at: string;
    revoked_at?: string;
}

export interface CreatedApiKey extends ApiKey {
    key: string; // Plaintext key, returned only at creation
}

export interface OverviewStats {
    total_interactions: number;
    total_cost: number;
    total_tokens: number;
    active_sessions: number;
}

export interface SessionSummary {
    uuid: string;
    name: string;
    description: string;
    workspace_uuid: string;
    interaction_count: number;
    total_tokens: number;
    total_cost: number;
    created_at: string;
    first_interaction?: string;
    last_interaction?: string;
}

export interface Interaction {
    uuid: string;
    provider: string;
    model: string;
    prompt_tokens: number;
    response_tokens: number;
    cost: number;
    created_at: string;
    status: string;
}

export interface ProviderStat {
    provider: string;
    model: string;
    count: number;
    total_cost: number;
    total_tokens: number;
}

export interface ApiError {
    code: string;
    message: string;
    details?: Record<string, any>;
}

class ApiClient {
    private baseUrl: string;
    private token: string | null = null;

    constructor(baseUrl: string = API_BASE_URL) {
        this.baseUrl = baseUrl;
        // Load token from localStorage if available
        if (typeof window !== 'undefined') {
            this.token = localStorage.getItem('auth_token');
        }
    }

    setToken(token: string) {
        this.token = token;
        if (typeof window !== 'undefined') {
            localStorage.setItem('auth_token', token);
        }
    }

    clearToken() {
        this.token = null;
        if (typeof window !== 'undefined') {
            localStorage.removeItem('auth_token');
        }
    }

    private async request<T>(
        endpoint: string,
        options: RequestInit = {}
    ): Promise<T> {
        const headers = new Headers(options.headers);
        headers.set('Content-Type', 'application/json');

        if (this.token) {
            headers.set('Authorization', `Bearer ${this.token}`);
        }

        // Absolute URLs (other services) pass through; paths hit the auth service
        const url = endpoint.startsWith('http') ? endpoint : `${this.baseUrl}${endpoint}`;

        const response = await fetch(url, {
            ...options,
            headers,
        });

        const data = await response.json();

        if (!response.ok) {
            throw data as ApiError;
        }

        return data as T;
    }

    // Auth endpoints
    async register(email: string, password: string, fullName: string): Promise<AuthResponse> {
        return this.request<AuthResponse>('/auth/register', {
            method: 'POST',
            body: JSON.stringify({ email, password, full_name: fullName }),
        });
    }

    async login(email: string, password: string): Promise<AuthResponse> {
        return this.request<AuthResponse>('/auth/login', {
            method: 'POST',
            body: JSON.stringify({ email, password }),
        });
    }

    async getCurrentUser() {
        return this.request('/users/me', { method: 'GET' });
    }

    // Workspace endpoints
    async getWorkspaces(): Promise<Workspace[]> {
        return this.request<Workspace[]>('/workspaces', { method: 'GET' });
    }

    async createWorkspace(name: string): Promise<Workspace> {
        return this.request<Workspace>('/workspaces', {
            method: 'POST',
            body: JSON.stringify({ name }),
        });
    }

    async updateWorkspace(uuid: string, name: string): Promise<Workspace> {
        return this.request<Workspace>(`/workspaces/${uuid}`, {
            method: 'PATCH',
            body: JSON.stringify({ name }),
        });
    }

    async deleteWorkspace(uuid: string): Promise<void> {
        return this.request<void>(`/workspaces/${uuid}`, { method: 'DELETE' });
    }

    // API key endpoints
    async getApiKeys(workspaceUuid: string): Promise<ApiKey[]> {
        return this.request<ApiKey[]>(`/api-keys?workspace_uuid=${workspaceUuid}`, {
            method: 'GET',
        });
    }

    async createApiKey(workspaceUuid: string, name: string): Promise<CreatedApiKey> {
        return this.request<CreatedApiKey>('/api-keys', {
            method: 'POST',
            body: JSON.stringify({ workspace_uuid: workspaceUuid, name }),
        });
    }

    async revokeApiKey(uuid: string): Promise<void> {
        return this.request<void>(`/api-keys/${uuid}`, { method: 'DELETE' });
    }

    // Analytics endpoints (Query Service)
    async getOverview(workspaceUuid: string): Promise<OverviewStats> {
        return this.request<OverviewStats>(`${QUERY_BASE_URL}/v1/analytics/overview?workspace_uuid=${workspaceUuid}`, {
            method: 'GET',
        });
    }

    async getCosts(workspaceUuid: string) {
        return this.request(`${QUERY_BASE_URL}/v1/analytics/costs?workspace_uuid=${workspaceUuid}`, {
            method: 'GET',
        });
    }

    async getUsage(workspaceUuid: string) {
        return this.request(`${QUERY_BASE_URL}/v1/analytics/usage?workspace_uuid=${workspaceUuid}`, {
            method: 'GET',
        });
    }

    async getProviderStats(workspaceUuid: string): Promise<ProviderStat[]> {
        return this.request<ProviderStat[]>(`${QUERY_BASE_URL}/v1/analytics/providers?workspace_uuid=${workspaceUuid}`, {
            method: 'GET',
        });
    }

    // Session endpoints (Query Service)
    async listSessions(workspaceUuid: string): Promise<SessionSummary[]> {
        return this.request<SessionSummary[]>(`${QUERY_BASE_URL}/v1/sessions?workspace_uuid=${workspaceUuid}`, {
            method: 'GET',
        });
    }

    async getSessionInteractions(sessionUuid: string): Promise<Interaction[]> {
        return this.request<Interaction[]>(`${QUERY_BASE_URL}/v1/sessions/${sessionUuid}/interactions`, {
            method: 'GET',
        });
    }

    // Interaction endpoints (Query Service)
    async listInteractions(workspaceUuid: string): Promise<Interaction[]> {
        return this.request<Interaction[]>(`${QUERY_BASE_URL}/v1/interactions?workspace_uuid=${workspaceUuid}`, {
            method: 'GET',
        });
    }

    async searchInteractions(workspaceUuid: string, query: string): Promise<Interaction[]> {
        return this.request<Interaction[]>(`${QUERY_BASE_URL}/v1/interactions/search?workspace_uuid=${workspaceUuid}&q=${encodeURIComponent(query)}`, {
            method: 'GET',
        });
    }

    async exportInteractions(workspaceUuid: string, format: 'csv' | 'json'): Promise<Blob> {
        const headers = new Headers();
        if (this.token) {
            headers.set('Authorization', `Bearer ${this.token}`);
        }

        const response = await fetch(
            `${QUERY_BASE_URL}/v1/export/interactions?workspace_uuid=${workspaceUuid}&format=${format}`,
            { headers }
        );

        if (!response.ok) {
            throw (await response.json()) as ApiError;
        }

        return response.blob();
    }

    // Interactions endpoints (Ingest Service)
    async recordInteraction(data: any) {
        return this.request(`${INGEST_BASE_URL}/v1/interactions`, {
            method: 'POST',
            body: JSON.stringify(data),
        });
    }

}

export const apiClient = new ApiClient();
export default apiClient;
