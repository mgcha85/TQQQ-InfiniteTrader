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

export interface RebalanceItem {
    symbol: string;
    current_qty: number;
    current_price: number;
    current_val: number;
    current_wt: number;
    target_wt: number;
    target_val: number;
    target_qty: number;
    action: string;
    action_qty: number;
    ma_130: number;
    ma_130_prev: number;
    cond_price_under_ma: boolean;
    cond_ma_down: boolean;
    kill_switch: boolean;
}

export interface RebalancePlan {
    total_value: number;
    cash: number;
    items: RebalanceItem[];
    estimated_tax: number;
    action_summary: string;
}

export async function fetchRebalancePreview() {
    const res = await fetch('/api/rebalance/preview');
    if (!res.ok) {
        const err = await res.json().catch(() => ({ error: 'Failed to fetch rebalance plan' }));
        throw new Error(err.error || 'Failed to fetch rebalance plan');
    }
    return await res.json();
}

export async function executeRebalance(dryRun: boolean = true) {
    const res = await fetch(`/api/rebalance/execute?dry_run=${dryRun}`, { method: 'POST' });
    if (!res.ok) {
        const err = await res.json().catch(() => ({ error: 'Failed to execute rebalance' }));
        throw new Error(err.error || 'Failed to execute rebalance');
    }
    return await res.json();
}
