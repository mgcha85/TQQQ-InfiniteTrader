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
    <div class="flex justify-between items-center mb-6">
        <h1 class="text-3xl font-bold text-gray-800 dark:text-white">
            Dashboard
        </h1>
        <Button onclick={handleSync} disabled={syncing}>
            {#if syncing}
                <Spinner size="4" class="mr-2" /> Syncing...
            {:else}
                Sync Now
            {/if}
        </Button>
    </div>

    {#if loading}
        <div class="text-center">Loading...</div>
    {:else if cycles.length === 0}
        <div class="text-gray-500">
            No active cycles found. Check Settings or Sync.
        </div>
    {:else}
        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {#each cycles as cycle}
                <Card class="w-full max-w-sm">
                    <h5
                        class="mb-2 text-2xl font-bold tracking-tight text-gray-900 dark:text-white"
                    >
                        {cycle.Symbol}
                    </h5>
                    <div
                        class="font-normal text-gray-700 dark:text-gray-400 space-y-2"
                    >
                        <div class="flex justify-between">
                            <span>Cycle Day:</span>
                            <span class="font-semibold"
                                >{cycle.CurrentCycleDay} / 40</span
                            >
                        </div>
                        <div class="flex justify-between">
                            <span>Held Qty:</span>
                            <span class="font-semibold"
                                >{cycle.TotalBoughtQty}</span
                            >
                        </div>
                        <div class="flex justify-between">
                            <span>Avg Price:</span>
                            <span class="font-semibold"
                                >${cycle.AvgPrice.toFixed(2)}</span
                            >
                        </div>
                        <div class="flex justify-between">
                            <span>Invested:</span>
                            <span class="font-semibold"
                                >${cycle.TotalInvested.toFixed(2)}</span
                            >
                        </div>

                        <!-- Progress Bar Logic (simple) -->
                        <div
                            class="w-full bg-gray-200 rounded-full h-2.5 dark:bg-gray-700 mt-4"
                        >
                            <div
                                class="bg-blue-600 h-2.5 rounded-full"
                                style="width: {Math.min(
                                    (cycle.CurrentCycleDay / 40) * 100,
                                    100,
                                )}%"
                            ></div>
                        </div>
                    </div>
                </Card>
            {/each}
        </div>
    {/if}
</div>
