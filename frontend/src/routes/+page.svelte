<script lang="ts">
    import { onMount } from "svelte";
    import { fetchDashboard, triggerSync, type CycleStatus } from "$lib/api";
    import { Card, Button, Spinner } from "flowbite-svelte";

    let cycles: CycleStatus[] = $state([]);
    let loading = $state(true);
    let syncing = $state(false);

    async function load() {
        loading = true;
        try {
            const data = await fetchDashboard();
            cycles = data.cycles || [];
        } catch (e) {
            console.error(e);
        } finally {
            loading = false;
        }
    }

    async function handleSync() {
        syncing = true;
        try {
            await triggerSync();
            await load();
        } catch (e) {
            alert("Sync failed");
        } finally {
            syncing = false;
        }
    }

    onMount(() => {
        load();
    });
</script>

<div class="p-8">
    <div class="flex justify-between items-center mb-8">
        <div>
            <h1 class="text-4xl font-bold text-white mb-2">Dashboard</h1>
            <p class="text-slate-400">
                Monitor your infinite buying strategy cycles
            </p>
        </div>
        <button onclick={handleSync} disabled={syncing} class="btn-primary">
            {#if syncing}
                <Spinner size="4" class="mr-2" /> Syncing...
            {:else}
                <svg
                    class="w-5 h-5 mr-2 inline-block"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                >
                    <path
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        stroke-width="2"
                        d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
                    />
                </svg>
                Sync Now
            {/if}
        </button>
    </div>

    {#if loading}
        <div class="text-center py-12">
            <Spinner size="12" />
            <p class="text-slate-400 mt-4">Loading dashboard...</p>
        </div>
    {:else if cycles.length === 0}
        <div class="stat-card text-center py-12">
            <svg
                class="w-16 h-16 mx-auto text-slate-600 mb-4"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
            >
                <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4"
                />
            </svg>
            <p class="text-slate-300 text-lg mb-2">No active cycles found</p>
            <p class="text-slate-500">
                Check Settings or click Sync to get started
            </p>
        </div>
    {:else}
        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {#each cycles as cycle}
                <div class="stat-card">
                    <div class="flex justify-between items-start mb-4">
                        <h5 class="text-2xl font-bold text-blue-400">
                            {cycle.Symbol}
                        </h5>
                        <span
                            class="px-3 py-1 bg-blue-500/20 text-blue-400 rounded-full text-sm font-medium"
                        >
                            Active
                        </span>
                    </div>

                    <div class="space-y-3 text-slate-300">
                        <div class="flex justify-between items-center">
                            <span class="text-slate-400">Cycle Progress</span>
                            <span class="font-semibold text-white">
                                {cycle.CurrentCycleDay} / 40
                            </span>
                        </div>

                        <!-- Progress Bar -->
                        <div
                            class="w-full bg-slate-800 rounded-full h-2 overflow-hidden"
                        >
                            <div
                                class="bg-gradient-to-r from-blue-600 to-blue-400 h-2 rounded-full transition-all duration-500"
                                style="width: {Math.min(
                                    (cycle.CurrentCycleDay / 40) * 100,
                                    100,
                                )}%"
                            ></div>
                        </div>

                        <div class="pt-2 space-y-2">
                            <div class="flex justify-between">
                                <span class="text-slate-400">Holdings</span>
                                <span class="font-semibold text-white"
                                    >{cycle.TotalBoughtQty} shares</span
                                >
                            </div>
                            <div class="flex justify-between">
                                <span class="text-slate-400">Avg Price</span>
                                <span class="font-semibold text-green-400"
                                    >${cycle.AvgPrice.toFixed(2)}</span
                                >
                            </div>
                            <div
                                class="flex justify-between pt-2 border-t border-slate-700"
                            >
                                <span class="text-slate-400"
                                    >Total Invested</span
                                >
                                <span class="font-bold text-xl text-white"
                                    >${cycle.TotalInvested.toFixed(2)}</span
                                >
                            </div>
                        </div>
                    </div>
                </div>
            {/each}
        </div>
    {/if}
</div>
