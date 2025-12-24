export interface CycleStatus {
    ID: number;
    Symbol: string;
    CurrentCycleDay: number;
    TotalBoughtQty: number;
    AvgPrice: number;
    TotalInvested: number;
}

export interface UserSettings {
    Principal: number;
    SplitCount: number;
    TargetRate: number;
    Symbols: string;
    IsActive: boolean;
}

export async function fetchDashboard() {
    const res = await fetch('/api/dashboard');
    if (!res.ok) throw new Error('Failed to fetch dashboard');
    return await res.json();
}

export async function fetchSettings() {
    const res = await fetch('/api/settings');
    if (!res.ok) throw new Error('Failed to fetch settings');
    return await res.json();
}

export async function updateSettings(settings: UserSettings) {
    const res = await fetch('/api/settings', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify(settings)
    });
    if (!res.ok) throw new Error('Failed to update settings');
    return await res.json();
}

export async function triggerSync() {
    const res = await fetch('/api/sync', { method: 'POST' });
    if (!res.ok) throw new Error('Sync failed');
    return await res.json();
}
