'use client';

import ProtectedRoute from '@/components/ProtectedRoute';
import { useAuth } from '@/contexts/AuthContext';
import { useState, useEffect } from 'react';
import { apiClient } from '@/lib/api-client';

export default function DashboardPage() {
    const { user, logout } = useAuth();
    const [activeTab, setActiveTab] = useState('overview');
    const [stats, setStats] = useState<any>(null);
    const [loading, setLoading] = useState(true);

    // Mock workspace UUID - in real app would come from user's workspace
    const workspaceUuid = '00000000-0000-0000-0000-000000000000';

    useEffect(() => {
        if (activeTab === 'overview') {
            loadOverviewData();
        }
    }, [activeTab]);

    const loadOverviewData = async () => {
        try {
            setLoading(true);
            const data = await apiClient.getOverview(workspaceUuid);
            setStats(data);
        } catch (error) {
            console.error('Failed to load overview:', error);
            // Use empty data on error
            setStats({
                total_interactions: 0,
                total_cost: 0,
                total_tokens: 0,
                active_sessions: 0,
            });
        } finally {
            setLoading(false);
        }
    };

    return (
        <ProtectedRoute>
            <div className="min-h-screen bg-slate-50">
                {/* Navigation */}
                <nav className="bg-white border-b border-slate-200 shadow-sm">
                    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
                        <div className="flex justify-between h-16">
                            <div className="flex items-center">
                                <div className="flex-shrink-0 flex items-center">
                                    <div className="w-8 h-8 bg-gradient-to-br from-blue-600 to-purple-600 rounded-lg flex items-center justify-center text-white font-bold">
                                        E
                                    </div>
                                    <span className="ml-3 text-xl font-bold text-slate-900">Eyesite</span>
                                </div>
                                <div className="hidden sm:ml-10 sm:flex sm:space-x-8">
                                    <button
                                        onClick={() => setActiveTab('overview')}
                                        className={`${activeTab === 'overview'
                                                ? 'border-blue-500 text-slate-900'
                                                : 'border-transparent text-slate-500 hover:border-slate-300 hover:text-slate-700'
                                            } inline-flex items-center px-1 pt-1 border-b-2 text-sm font-medium transition`}
                                    >
                                        Overview
                                    </button>
                                    <button
                                        onClick={() => setActiveTab('sessions')}
                                        className={`${activeTab === 'sessions'
                                                ? 'border-blue-500 text-slate-900'
                                                : 'border-transparent text-slate-500 hover:border-slate-300 hover:text-slate-700'
                                            } inline-flex items-center px-1 pt-1 border-b-2 text-sm font-medium transition`}
                                    >
                                        Sessions
                                    </button>
                                    <button
                                        onClick={() => setActiveTab('analytics')}
                                        className={`${activeTab === 'analytics'
                                                ? 'border-blue-500 text-slate-900'
                                                : 'border-transparent text-slate-500 hover:border-slate-300 hover:text-slate-700'
                                            } inline-flex items-center px-1 pt-1 border-b-2 text-sm font-medium transition`}
                                    >
                                        Analytics
                                    </button>
                                </div>
                            </div>
                            <div className="flex items-center space-x-4">
                                <span className="text-sm text-slate-600">{user?.full_name}</span>
                                <button
                                    onClick={logout}
                                    className="px-4 py-2 text-sm text-slate-700 hover:text-slate-900 hover:bg-slate-100 rounded-lg transition"
                                >
                                    Logout
                                </button>
                            </div>
                        </div>
                    </div>
                </nav>

                {/* Content */}
                <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
                    {activeTab === 'overview' && <OverviewTab stats={stats} loading={loading} />}
                    {activeTab === 'sessions' && <SessionsTab />}
                    {activeTab === 'analytics' && <AnalyticsTab workspaceUuid={workspaceUuid} />}
                </main>
            </div>
        </ProtectedRoute>
    );
}

function OverviewTab({ stats, loading }: { stats: any; loading: boolean }) {
    if (loading) {
        return (
            <div className="flex items-center justify-center h-64">
                <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
            </div>
        );
    }

    const totalInteractions = stats?.total_interactions || 0;
    const totalCost = stats?.total_cost || 0;
    const totalTokens = stats?.total_tokens || 0;
    const activeSessions = stats?.active_sessions || 0;

    return (
        <div className="space-y-6">
            <div>
                <h1 className="text-3xl font-bold text-slate-900">Dashboard Overview</h1>
                <p className="mt-2 text-slate-600">Monitor your AI agent performance and costs</p>
            </div>

            {/* Stats Grid */}
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
                <StatCard
                    title="Total Interactions"
                    value={totalInteractions.toLocaleString()}
                    change="+0%"
                    icon="📊"
                    trend="neutral"
                />
                <StatCard
                    title="Total Cost"
                    value={`$${totalCost.toFixed(4)}`}
                    change="+0%"
                    icon="💰"
                    trend="neutral"
                />
                <StatCard
                    title="Active Sessions"
                    value={activeSessions.toString()}
                    change="+0%"
                    icon="🔄"
                    trend="neutral"
                />
                <StatCard
                    title="Total Tokens"
                    value={totalTokens.toLocaleString()}
                    change="+0%"
                    icon="🔢"
                    trend="neutral"
                />
            </div>

            {/* Quick Start */}
            <div className="bg-white rounded-lg shadow-sm border border-slate-200 p-6">
                <h2 className="text-xl font-semibold text-slate-900 mb-4">🚀 Quick Start</h2>
                <div className="space-y-4">
                    <div className="flex items-start space-x-3">
                        <div className="flex-shrink-0 w-8 h-8 bg-blue-100 text-blue-600 rounded-full flex items-center justify-center font-semibold">
                            1
                        </div>
                        <div className="flex-1">
                            <h3 className="font-medium text-slate-900">Install the SDK</h3>
                            <code className="mt-1 block bg-slate-100 px-3 py-2 rounded text-sm font-mono">
                                npm install @eyesite/sdk
                            </code>
                        </div>
                    </div>
                    <div className="flex items-start space-x-3">
                        <div className="flex-shrink-0 w-8 h-8 bg-blue-100 text-blue-600 rounded-full flex items-center justify-center font-semibold">
                            2
                        </div>
                        <div className="flex-1">
                            <h3 className="font-medium text-slate-900">Wrap Your AI Client</h3>
                            <code className="mt-1 block bg-slate-100 px-3 py-2 rounded text-sm font-mono whitespace-pre">
                                {`import { Eyesite } from '@eyesite/sdk';
const eye = new Eyesite({ apiKey: 'your-key' });
eye.wrapOpenAI(openai); // That's it!`}
                            </code>
                        </div>
                    </div>
                    <div className="flex items-start space-x-3">
                        <div className="flex-shrink-0 w-8 h-8 bg-blue-100 text-blue-600 rounded-full flex items-center justify-center font-semibold">
                            3
                        </div>
                        <div>
                            <h3 className="font-medium text-slate-900">See Insights Here!</h3>
                            <p className="text-sm text-slate-600 mt-1">All your interactions will appear automatically</p>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    );
}

function SessionsTab() {
    return (
        <div className="space-y-6">
            <div className="flex justify-between items-center">
                <div>
                    <h1 className="text-3xl font-bold text-slate-900">Sessions</h1>
                    <p className="mt-2 text-slate-600">View all your agent sessions</p>
                </div>
                <button className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition shadow-sm">
                    Create Session
                </button>
            </div>

            <div className="bg-white rounded-lg shadow-sm border border-slate-200">
                <div className="p-8 text-center">
                    <div className="w-16 h-16 mx-auto bg-slate-100 rounded-full flex items-center justify-center text-2xl mb-4">
                        📂
                    </div>
                    <h3 className="text-lg font-medium text-slate-900 mb-2">No sessions yet</h3>
                    <p className="text-slate-600 mb-4">Create your first session to start tracking agent interactions</p>
                    <button className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition">
                        Create Your First Session
                    </button>
                </div>
            </div>
        </div>
    );
}

function AnalyticsTab({ workspaceUuid }: { workspaceUuid: string }) {
    const [providerStats, setProviderStats] = useState<any[]>([]);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        loadProviderStats();
    }, []);

    const loadProviderStats = async () => {
        try {
            const data = await apiClient.getProviderStats(workspaceUuid);
            setProviderStats(data || []);
        } catch (error) {
            console.error('Failed to load provider stats:', error);
            setProviderStats([]);
        } finally {
            setLoading(false);
        }
    };

    if (loading) {
        return (
            <div className="flex items-center justify-center h-64">
                <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
            </div>
        );
    }

    return (
        <div className="space-y-6">
            <div>
                <h1 className="text-3xl font-bold text-slate-900">Analytics</h1>
                <p className="mt-2 text-slate-600">Deep dive into your agent performance</p>
            </div>

            {providerStats.length > 0 ? (
                <div className="bg-white rounded-lg shadow-sm border border-slate-200 overflow-hidden">
                    <div className="px-6 py-4 border-b border-slate-200">
                        <h2 className="text-lg font-semibold text-slate-900">Provider Breakdown</h2>
                    </div>
                    <div className="overflow-x-auto">
                        <table className="min-w-full divide-y divide-slate-200">
                            <thead className="bg-slate-50">
                                <tr>
                                    <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">Provider</th>
                                    <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">Model</th>
                                    <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">Calls</th>
                                    <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">Tokens</th>
                                    <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">Cost</th>
                                </tr>
                            </thead>
                            <tbody className="bg-white divide-y divide-slate-200">
                                {providerStats.map((stat, idx) => (
                                    <tr key={idx} className="hover:bg-slate-50 transition">
                                        <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-slate-900">{stat.provider}</td>
                                        <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-600">{stat.model}</td>
                                        <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-600">{stat.count}</td>
                                        <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-600">{stat.total_tokens?.toLocaleString()}</td>
                                        <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-900 font-medium">${stat.total_cost?.toFixed(4)}</td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    </div>
                </div>
            ) : (
                <div className="bg-white rounded-lg shadow-sm border border-slate-200 p-8 text-center">
                    <div className="w-16 h-16 mx-auto bg-slate-100 rounded-full flex items-center justify-center text-2xl mb-4">
                        📈
                    </div>
                    <h3 className="text-lg font-medium text-slate-900 mb-2">No analytics yet</h3>
                    <p className="text-slate-600">Start tracking interactions to see analytics here</p>
                </div>
            )}
        </div>
    );
}

function StatCard({ title, value, change, icon, trend }: {
    title: string;
    value: string;
    change: string;
    icon: string;
    trend: 'up' | 'down' | 'neutral';
}) {
    const trendColor = trend === 'up' ? 'text-green-600' : trend === 'down' ? 'text-red-600' : 'text-slate-400';

    return (
        <div className="bg-white rounded-lg shadow-sm border border-slate-200 p-6 hover:shadow-md transition">
            <div className="flex items-center justify-between">
                <div>
                    <p className="text-sm text-slate-600 font-medium">{title}</p>
                    <p className="mt-2 text-3xl font-bold text-slate-900">{value}</p>
                    <p className={`mt-1 text-sm ${trendColor}`}>{change} from last week</p>
                </div>
                <div className="text-4xl">{icon}</div>
            </div>
        </div>
    );
}
