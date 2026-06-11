'use client';

import ProtectedRoute from '@/components/ProtectedRoute';
import { useAuth } from '@/contexts/AuthContext';
import { useState, useEffect, useCallback } from 'react';
import {
    apiClient,
    Workspace,
    ApiKey,
    CreatedApiKey,
    OverviewStats,
    SessionSummary,
    Interaction,
    ProviderStat,
} from '@/lib/api-client';

const TABS = ['overview', 'sessions', 'analytics', 'api-keys', 'settings'] as const;
type Tab = typeof TABS[number];

const TAB_LABELS: Record<Tab, string> = {
    overview: 'Overview',
    sessions: 'Sessions',
    analytics: 'Analytics',
    'api-keys': 'API Keys',
    settings: 'Settings',
};

export default function DashboardPage() {
    const { user, logout } = useAuth();
    const [activeTab, setActiveTab] = useState<Tab>('overview');
    const [workspaces, setWorkspaces] = useState<Workspace[]>([]);
    const [workspaceUuid, setWorkspaceUuid] = useState<string | null>(null);
    const [wsLoading, setWsLoading] = useState(true);

    const loadWorkspaces = useCallback(async () => {
        try {
            const data = await apiClient.getWorkspaces();
            setWorkspaces(data);
            setWorkspaceUuid(prev =>
                prev && data.some(w => w.uuid === prev) ? prev : (data[0]?.uuid ?? null)
            );
        } catch (error) {
            console.error('Failed to load workspaces:', error);
        } finally {
            setWsLoading(false);
        }
    }, []);

    useEffect(() => {
        loadWorkspaces();
    }, [loadWorkspaces]);

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
                                    {TABS.map(tab => (
                                        <button
                                            key={tab}
                                            onClick={() => setActiveTab(tab)}
                                            className={`${activeTab === tab
                                                ? 'border-blue-500 text-slate-900'
                                                : 'border-transparent text-slate-500 hover:border-slate-300 hover:text-slate-700'
                                                } inline-flex items-center px-1 pt-1 border-b-2 text-sm font-medium transition`}
                                        >
                                            {TAB_LABELS[tab]}
                                        </button>
                                    ))}
                                </div>
                            </div>
                            <div className="flex items-center space-x-4">
                                {workspaces.length > 0 && (
                                    <select
                                        value={workspaceUuid ?? ''}
                                        onChange={e => setWorkspaceUuid(e.target.value)}
                                        className="text-sm border border-slate-300 rounded-lg px-3 py-1.5 bg-white text-slate-700"
                                    >
                                        {workspaces.map(ws => (
                                            <option key={ws.uuid} value={ws.uuid}>{ws.name}</option>
                                        ))}
                                    </select>
                                )}
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
                    {wsLoading ? (
                        <Spinner />
                    ) : !workspaceUuid ? (
                        <EmptyState icon="🏢" title="No workspace found" subtitle="Create a workspace in Settings to get started" />
                    ) : (
                        <>
                            {activeTab === 'overview' && <OverviewTab workspaceUuid={workspaceUuid} />}
                            {activeTab === 'sessions' && <SessionsTab workspaceUuid={workspaceUuid} />}
                            {activeTab === 'analytics' && <AnalyticsTab workspaceUuid={workspaceUuid} />}
                            {activeTab === 'api-keys' && <ApiKeysTab workspaceUuid={workspaceUuid} />}
                            {activeTab === 'settings' && (
                                <SettingsTab
                                    workspaces={workspaces}
                                    currentUserUuid={user?.uuid}
                                    onChanged={loadWorkspaces}
                                />
                            )}
                        </>
                    )}
                </main>
            </div>
        </ProtectedRoute>
    );
}

function Spinner() {
    return (
        <div className="flex items-center justify-center h-64">
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
        </div>
    );
}

function EmptyState({ icon, title, subtitle }: { icon: string; title: string; subtitle: string }) {
    return (
        <div className="bg-white rounded-lg shadow-sm border border-slate-200 p-8 text-center">
            <div className="w-16 h-16 mx-auto bg-slate-100 rounded-full flex items-center justify-center text-2xl mb-4">
                {icon}
            </div>
            <h3 className="text-lg font-medium text-slate-900 mb-2">{title}</h3>
            <p className="text-slate-600">{subtitle}</p>
        </div>
    );
}

function OverviewTab({ workspaceUuid }: { workspaceUuid: string }) {
    const [stats, setStats] = useState<OverviewStats | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        let cancelled = false;
        setLoading(true);
        setError(null);
        apiClient.getOverview(workspaceUuid)
            .then(data => { if (!cancelled) setStats(data); })
            .catch(err => {
                console.error('Failed to load overview:', err);
                if (!cancelled) setError('Could not reach the analytics service');
            })
            .finally(() => { if (!cancelled) setLoading(false); });
        return () => { cancelled = true; };
    }, [workspaceUuid]);

    if (loading) return <Spinner />;

    return (
        <div className="space-y-6">
            <div>
                <h1 className="text-3xl font-bold text-slate-900">Dashboard Overview</h1>
                <p className="mt-2 text-slate-600">Monitor your AI agent performance and costs</p>
            </div>

            {error && (
                <div className="bg-amber-50 border border-amber-200 text-amber-800 rounded-lg px-4 py-3 text-sm">
                    {error}
                </div>
            )}

            {/* Stats Grid */}
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
                <StatCard title="Total Interactions" value={(stats?.total_interactions ?? 0).toLocaleString()} icon="📊" />
                <StatCard title="Total Cost" value={`$${(stats?.total_cost ?? 0).toFixed(4)}`} icon="💰" />
                <StatCard title="Active Sessions" value={String(stats?.active_sessions ?? 0)} icon="🔄" />
                <StatCard title="Total Tokens" value={(stats?.total_tokens ?? 0).toLocaleString()} icon="🔢" />
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

function SessionsTab({ workspaceUuid }: { workspaceUuid: string }) {
    const [sessions, setSessions] = useState<SessionSummary[]>([]);
    const [selected, setSelected] = useState<SessionSummary | null>(null);
    const [interactions, setInteractions] = useState<Interaction[]>([]);
    const [loading, setLoading] = useState(true);
    const [detailLoading, setDetailLoading] = useState(false);

    useEffect(() => {
        let cancelled = false;
        setLoading(true);
        setSelected(null);
        apiClient.listSessions(workspaceUuid)
            .then(data => { if (!cancelled) setSessions(data); })
            .catch(err => console.error('Failed to load sessions:', err))
            .finally(() => { if (!cancelled) setLoading(false); });
        return () => { cancelled = true; };
    }, [workspaceUuid]);

    const openSession = async (session: SessionSummary) => {
        setSelected(session);
        setDetailLoading(true);
        try {
            setInteractions(await apiClient.getSessionInteractions(session.uuid));
        } catch (err) {
            console.error('Failed to load session interactions:', err);
            setInteractions([]);
        } finally {
            setDetailLoading(false);
        }
    };

    if (loading) return <Spinner />;

    if (selected) {
        return (
            <div className="space-y-6">
                <div>
                    <button onClick={() => setSelected(null)} className="text-sm text-blue-600 hover:text-blue-800 mb-2">
                        ← Back to sessions
                    </button>
                    <h1 className="text-3xl font-bold text-slate-900">{selected.name}</h1>
                    {selected.description && <p className="mt-2 text-slate-600">{selected.description}</p>}
                </div>

                <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                    <StatCard title="Interactions" value={selected.interaction_count.toLocaleString()} icon="💬" />
                    <StatCard title="Tokens" value={selected.total_tokens.toLocaleString()} icon="🔢" />
                    <StatCard title="Cost" value={`$${selected.total_cost.toFixed(4)}`} icon="💰" />
                </div>

                {detailLoading ? <Spinner /> : interactions.length === 0 ? (
                    <EmptyState icon="💬" title="No interactions" subtitle="This session has no recorded interactions yet" />
                ) : (
                    <InteractionsTable interactions={interactions} />
                )}
            </div>
        );
    }

    return (
        <div className="space-y-6">
            <div>
                <h1 className="text-3xl font-bold text-slate-900">Sessions</h1>
                <p className="mt-2 text-slate-600">View all your agent sessions</p>
            </div>

            {sessions.length === 0 ? (
                <EmptyState icon="📂" title="No sessions yet" subtitle="Create a session with the SDK's createSession() to group your interactions" />
            ) : (
                <div className="bg-white rounded-lg shadow-sm border border-slate-200 overflow-hidden">
                    <table className="min-w-full divide-y divide-slate-200">
                        <thead className="bg-slate-50">
                            <tr>
                                <Th>Name</Th>
                                <Th>Interactions</Th>
                                <Th>Tokens</Th>
                                <Th>Cost</Th>
                                <Th>Last Activity</Th>
                            </tr>
                        </thead>
                        <tbody className="bg-white divide-y divide-slate-200">
                            {sessions.map(s => (
                                <tr key={s.uuid} onClick={() => openSession(s)} className="hover:bg-slate-50 transition cursor-pointer">
                                    <Td className="font-medium text-slate-900">{s.name}</Td>
                                    <Td>{s.interaction_count}</Td>
                                    <Td>{s.total_tokens.toLocaleString()}</Td>
                                    <Td className="text-slate-900 font-medium">${s.total_cost.toFixed(4)}</Td>
                                    <Td>{s.last_interaction ? new Date(s.last_interaction).toLocaleString() : '—'}</Td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </div>
            )}
        </div>
    );
}

function InteractionsTable({ interactions }: { interactions: Interaction[] }) {
    return (
        <div className="bg-white rounded-lg shadow-sm border border-slate-200 overflow-hidden">
            <div className="overflow-x-auto">
                <table className="min-w-full divide-y divide-slate-200">
                    <thead className="bg-slate-50">
                        <tr>
                            <Th>Provider</Th>
                            <Th>Model</Th>
                            <Th>Tokens</Th>
                            <Th>Cost</Th>
                            <Th>Status</Th>
                            <Th>Time</Th>
                        </tr>
                    </thead>
                    <tbody className="bg-white divide-y divide-slate-200">
                        {interactions.map(i => (
                            <tr key={i.uuid} className="hover:bg-slate-50 transition">
                                <Td className="font-medium text-slate-900">{i.provider}</Td>
                                <Td>{i.model}</Td>
                                <Td>{(i.prompt_tokens + i.response_tokens).toLocaleString()}</Td>
                                <Td className="text-slate-900 font-medium">${i.cost.toFixed(4)}</Td>
                                <Td>
                                    <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${i.status === 'success' ? 'bg-green-100 text-green-700' :
                                        i.status === 'error' ? 'bg-red-100 text-red-700' : 'bg-slate-100 text-slate-600'
                                        }`}>
                                        {i.status}
                                    </span>
                                </Td>
                                <Td>{new Date(i.created_at).toLocaleString()}</Td>
                            </tr>
                        ))}
                    </tbody>
                </table>
            </div>
        </div>
    );
}

function AnalyticsTab({ workspaceUuid }: { workspaceUuid: string }) {
    const [providerStats, setProviderStats] = useState<ProviderStat[]>([]);
    const [loading, setLoading] = useState(true);
    const [exporting, setExporting] = useState(false);

    const exportData = async (format: 'csv' | 'json') => {
        setExporting(true);
        try {
            const blob = await apiClient.exportInteractions(workspaceUuid, format);
            const url = URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = `interactions.${format}`;
            a.click();
            URL.revokeObjectURL(url);
        } catch (err) {
            console.error('Failed to export interactions:', err);
        } finally {
            setExporting(false);
        }
    };

    useEffect(() => {
        let cancelled = false;
        setLoading(true);
        apiClient.getProviderStats(workspaceUuid)
            .then(data => { if (!cancelled) setProviderStats(data || []); })
            .catch(err => {
                console.error('Failed to load provider stats:', err);
                if (!cancelled) setProviderStats([]);
            })
            .finally(() => { if (!cancelled) setLoading(false); });
        return () => { cancelled = true; };
    }, [workspaceUuid]);

    if (loading) return <Spinner />;

    return (
        <div className="space-y-6">
            <div className="flex justify-between items-center">
                <div>
                    <h1 className="text-3xl font-bold text-slate-900">Analytics</h1>
                    <p className="mt-2 text-slate-600">Deep dive into your agent performance</p>
                </div>
                <div className="flex space-x-2">
                    <button
                        onClick={() => exportData('csv')}
                        disabled={exporting}
                        className="px-4 py-2 text-sm border border-slate-300 text-slate-700 rounded-lg hover:bg-slate-100 transition disabled:opacity-50"
                    >
                        Export CSV
                    </button>
                    <button
                        onClick={() => exportData('json')}
                        disabled={exporting}
                        className="px-4 py-2 text-sm border border-slate-300 text-slate-700 rounded-lg hover:bg-slate-100 transition disabled:opacity-50"
                    >
                        Export JSON
                    </button>
                </div>
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
                                    <Th>Provider</Th>
                                    <Th>Model</Th>
                                    <Th>Calls</Th>
                                    <Th>Tokens</Th>
                                    <Th>Cost</Th>
                                </tr>
                            </thead>
                            <tbody className="bg-white divide-y divide-slate-200">
                                {providerStats.map((stat, idx) => (
                                    <tr key={idx} className="hover:bg-slate-50 transition">
                                        <Td className="font-medium text-slate-900">{stat.provider}</Td>
                                        <Td>{stat.model}</Td>
                                        <Td>{stat.count}</Td>
                                        <Td>{stat.total_tokens?.toLocaleString()}</Td>
                                        <Td className="text-slate-900 font-medium">${stat.total_cost?.toFixed(4)}</Td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    </div>
                </div>
            ) : (
                <EmptyState icon="📈" title="No analytics yet" subtitle="Start tracking interactions to see analytics here" />
            )}
        </div>
    );
}

function ApiKeysTab({ workspaceUuid }: { workspaceUuid: string }) {
    const [keys, setKeys] = useState<ApiKey[]>([]);
    const [loading, setLoading] = useState(true);
    const [newKeyName, setNewKeyName] = useState('');
    const [creating, setCreating] = useState(false);
    const [createdKey, setCreatedKey] = useState<CreatedApiKey | null>(null);
    const [error, setError] = useState<string | null>(null);

    const loadKeys = useCallback(async () => {
        try {
            setKeys(await apiClient.getApiKeys(workspaceUuid));
        } catch (err) {
            console.error('Failed to load API keys:', err);
        } finally {
            setLoading(false);
        }
    }, [workspaceUuid]);

    useEffect(() => {
        setLoading(true);
        setCreatedKey(null);
        loadKeys();
    }, [loadKeys]);

    const createKey = async () => {
        if (!newKeyName.trim()) return;
        setCreating(true);
        setError(null);
        try {
            const created = await apiClient.createApiKey(workspaceUuid, newKeyName.trim());
            setCreatedKey(created);
            setNewKeyName('');
            await loadKeys();
        } catch (err: any) {
            setError(err?.message || 'Failed to create API key');
        } finally {
            setCreating(false);
        }
    };

    const revokeKey = async (uuid: string) => {
        if (!confirm('Revoke this API key? Clients using it will stop working.')) return;
        try {
            await apiClient.revokeApiKey(uuid);
            await loadKeys();
        } catch (err: any) {
            setError(err?.message || 'Failed to revoke API key');
        }
    };

    if (loading) return <Spinner />;

    return (
        <div className="space-y-6">
            <div>
                <h1 className="text-3xl font-bold text-slate-900">API Keys</h1>
                <p className="mt-2 text-slate-600">Keys used by the SDK to send telemetry to this workspace</p>
            </div>

            {error && (
                <div className="bg-red-50 border border-red-200 text-red-800 rounded-lg px-4 py-3 text-sm">
                    {error}
                </div>
            )}

            {createdKey && (
                <div className="bg-green-50 border border-green-200 rounded-lg px-4 py-3">
                    <p className="text-sm font-medium text-green-800 mb-2">
                        Key created — copy it now, it will not be shown again:
                    </p>
                    <div className="flex items-center space-x-2">
                        <code className="flex-1 bg-white border border-green-200 rounded px-3 py-2 text-sm font-mono break-all">
                            {createdKey.key}
                        </code>
                        <button
                            onClick={() => navigator.clipboard.writeText(createdKey.key)}
                            className="px-3 py-2 text-sm bg-green-600 text-white rounded-lg hover:bg-green-700 transition"
                        >
                            Copy
                        </button>
                    </div>
                </div>
            )}

            <div className="bg-white rounded-lg shadow-sm border border-slate-200 p-4 flex space-x-3">
                <input
                    value={newKeyName}
                    onChange={e => setNewKeyName(e.target.value)}
                    onKeyDown={e => e.key === 'Enter' && createKey()}
                    placeholder="Key name (e.g. production)"
                    className="flex-1 border border-slate-300 rounded-lg px-3 py-2 text-sm"
                />
                <button
                    onClick={createKey}
                    disabled={creating || !newKeyName.trim()}
                    className="px-4 py-2 bg-blue-600 text-white text-sm rounded-lg hover:bg-blue-700 transition disabled:opacity-50"
                >
                    {creating ? 'Creating…' : 'Create Key'}
                </button>
            </div>

            {keys.length === 0 ? (
                <EmptyState icon="🔑" title="No API keys" subtitle="Create a key to start sending data from the SDK" />
            ) : (
                <div className="bg-white rounded-lg shadow-sm border border-slate-200 overflow-hidden">
                    <table className="min-w-full divide-y divide-slate-200">
                        <thead className="bg-slate-50">
                            <tr>
                                <Th>Name</Th>
                                <Th>Key</Th>
                                <Th>Created</Th>
                                <Th>Status</Th>
                                <Th>{''}</Th>
                            </tr>
                        </thead>
                        <tbody className="bg-white divide-y divide-slate-200">
                            {keys.map(k => (
                                <tr key={k.uuid} className="hover:bg-slate-50 transition">
                                    <Td className="font-medium text-slate-900">{k.name}</Td>
                                    <Td><code className="text-xs">{k.key_preview}…</code></Td>
                                    <Td>{new Date(k.created_at).toLocaleDateString()}</Td>
                                    <Td>
                                        <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${k.revoked_at ? 'bg-red-100 text-red-700' : 'bg-green-100 text-green-700'
                                            }`}>
                                            {k.revoked_at ? 'revoked' : 'active'}
                                        </span>
                                    </Td>
                                    <Td>
                                        {!k.revoked_at && (
                                            <button
                                                onClick={() => revokeKey(k.uuid)}
                                                className="text-sm text-red-600 hover:text-red-800"
                                            >
                                                Revoke
                                            </button>
                                        )}
                                    </Td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </div>
            )}
        </div>
    );
}

function SettingsTab({ workspaces, currentUserUuid, onChanged }: {
    workspaces: Workspace[];
    currentUserUuid?: string;
    onChanged: () => Promise<void>;
}) {
    const [newName, setNewName] = useState('');
    const [busy, setBusy] = useState(false);
    const [editing, setEditing] = useState<string | null>(null);
    const [editName, setEditName] = useState('');
    const [error, setError] = useState<string | null>(null);

    const run = async (fn: () => Promise<unknown>) => {
        setBusy(true);
        setError(null);
        try {
            await fn();
            await onChanged();
        } catch (err: any) {
            setError(err?.message || 'Operation failed');
        } finally {
            setBusy(false);
        }
    };

    const create = () => {
        if (!newName.trim()) return;
        run(() => apiClient.createWorkspace(newName.trim())).then(() => setNewName(''));
    };

    const rename = (uuid: string) => {
        if (!editName.trim()) return;
        run(() => apiClient.updateWorkspace(uuid, editName.trim())).then(() => setEditing(null));
    };

    const remove = (ws: Workspace) => {
        if (!confirm(`Delete workspace "${ws.name}"? All its sessions, interactions, and API keys will be permanently deleted.`)) return;
        run(() => apiClient.deleteWorkspace(ws.uuid));
    };

    return (
        <div className="space-y-6">
            <div>
                <h1 className="text-3xl font-bold text-slate-900">Workspace Settings</h1>
                <p className="mt-2 text-slate-600">Manage your workspaces</p>
            </div>

            {error && (
                <div className="bg-red-50 border border-red-200 text-red-800 rounded-lg px-4 py-3 text-sm">
                    {error}
                </div>
            )}

            <div className="bg-white rounded-lg shadow-sm border border-slate-200 p-4 flex space-x-3">
                <input
                    value={newName}
                    onChange={e => setNewName(e.target.value)}
                    onKeyDown={e => e.key === 'Enter' && create()}
                    placeholder="New workspace name"
                    className="flex-1 border border-slate-300 rounded-lg px-3 py-2 text-sm"
                />
                <button
                    onClick={create}
                    disabled={busy || !newName.trim()}
                    className="px-4 py-2 bg-blue-600 text-white text-sm rounded-lg hover:bg-blue-700 transition disabled:opacity-50"
                >
                    Create Workspace
                </button>
            </div>

            <div className="bg-white rounded-lg shadow-sm border border-slate-200 overflow-hidden">
                <table className="min-w-full divide-y divide-slate-200">
                    <thead className="bg-slate-50">
                        <tr>
                            <Th>Name</Th>
                            <Th>Slug</Th>
                            <Th>Created</Th>
                            <Th>{''}</Th>
                        </tr>
                    </thead>
                    <tbody className="bg-white divide-y divide-slate-200">
                        {workspaces.map(ws => (
                            <tr key={ws.uuid} className="hover:bg-slate-50 transition">
                                <Td className="font-medium text-slate-900">
                                    {editing === ws.uuid ? (
                                        <input
                                            value={editName}
                                            onChange={e => setEditName(e.target.value)}
                                            onKeyDown={e => e.key === 'Enter' && rename(ws.uuid)}
                                            className="border border-slate-300 rounded px-2 py-1 text-sm"
                                            autoFocus
                                        />
                                    ) : ws.name}
                                </Td>
                                <Td><code className="text-xs">{ws.slug}</code></Td>
                                <Td>{new Date(ws.created_at).toLocaleDateString()}</Td>
                                <Td>
                                    <div className="flex space-x-3 text-sm">
                                        {editing === ws.uuid ? (
                                            <>
                                                <button onClick={() => rename(ws.uuid)} className="text-blue-600 hover:text-blue-800">Save</button>
                                                <button onClick={() => setEditing(null)} className="text-slate-500 hover:text-slate-700">Cancel</button>
                                            </>
                                        ) : (
                                            <>
                                                <button
                                                    onClick={() => { setEditing(ws.uuid); setEditName(ws.name); }}
                                                    className="text-blue-600 hover:text-blue-800"
                                                >
                                                    Rename
                                                </button>
                                                {ws.owner_uuid === currentUserUuid && workspaces.length > 1 && (
                                                    <button onClick={() => remove(ws)} className="text-red-600 hover:text-red-800">
                                                        Delete
                                                    </button>
                                                )}
                                            </>
                                        )}
                                    </div>
                                </Td>
                            </tr>
                        ))}
                    </tbody>
                </table>
            </div>
        </div>
    );
}

function Th({ children }: { children: React.ReactNode }) {
    return (
        <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">
            {children}
        </th>
    );
}

function Td({ children, className = '' }: { children: React.ReactNode; className?: string }) {
    return (
        <td className={`px-6 py-4 whitespace-nowrap text-sm text-slate-600 ${className}`}>
            {children}
        </td>
    );
}

function StatCard({ title, value, icon }: { title: string; value: string; icon: string }) {
    return (
        <div className="bg-white rounded-lg shadow-sm border border-slate-200 p-6 hover:shadow-md transition">
            <div className="flex items-center justify-between">
                <div>
                    <p className="text-sm text-slate-600 font-medium">{title}</p>
                    <p className="mt-2 text-3xl font-bold text-slate-900">{value}</p>
                </div>
                <div className="text-4xl">{icon}</div>
            </div>
        </div>
    );
}
