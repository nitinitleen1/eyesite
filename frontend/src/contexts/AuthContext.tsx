'use client';

import { createContext, useContext, useEffect, useState, ReactNode } from 'react';
import { apiClient, AuthResponse } from '@/lib/api-client';

interface User {
    uuid: string;
    email: string;
    full_name: string;
    is_verified: boolean;
    created_at: string;
}

interface AuthContextType {
    user: User | null;
    loading: boolean;
    login: (email: string, password: string) => Promise<void>;
    register: (email: string, password: string, fullName: string) => Promise<void>;
    logout: () => void;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
    const [user, setUser] = useState<User | null>(null);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        // Check if user is already logged in
        const checkAuth = async () => {
            try {
                const userData = await apiClient.getCurrentUser();
                setUser(userData as User);
            } catch (error) {
                // Not logged in or token expired
                apiClient.clearToken();
            } finally {
                setLoading(false);
            }
        };

        checkAuth();
    }, []);

    const login = async (email: string, password: string) => {
        const response: AuthResponse = await apiClient.login(email, password);
        apiClient.setToken(response.token);
        setUser(response.user);
    };

    const register = async (email: string, password: string, fullName: string) => {
        const response: AuthResponse = await apiClient.register(email, password, fullName);
        apiClient.setToken(response.token);
        setUser(response.user);
    };

    const logout = () => {
        apiClient.clearToken();
        setUser(null);
    };

    return (
        <AuthContext.Provider value={{ user, loading, login, register, logout }}>
            {children}
        </AuthContext.Provider>
    );
}

export function useAuth() {
    const context = useContext(AuthContext);
    if (context === undefined) {
        throw new Error('useAuth must be used within an AuthProvider');
    }
    return context;
}
