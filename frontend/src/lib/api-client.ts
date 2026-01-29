// API client configuration
const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8000';

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

        const response = await fetch(`${this.baseUrl}${endpoint}`, {
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
    async getWorkspaces() {
        return this.request('/workspaces', { method: 'GET' });
    }

    // Interactions endpoints
    async recordInteraction(data: any) {
        return this.request('/v1/interactions', {
            method: 'POST',
            body: JSON.stringify(data),
        });
    }

    async getInteractions(workspaceUuid: string, params?: Record<string, string>) {
        const query = params ? '?' + new URLSearchParams(params).toString() : '';
        return this.request(`/workspaces/${workspaceUuid}/interactions${query}`, {
            method: 'GET',
        });
    }
}

export const apiClient = new ApiClient();
export default apiClient;
