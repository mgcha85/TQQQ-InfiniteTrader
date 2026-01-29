<script lang="ts">
    import { onMount } from "svelte";
    import {
        fetchRebalancePreview,
        executeCustomRebalance,
        type RebalancePlan,
        type RebalanceItem,
    } from "$lib/api";
    import {
        Spinner,
        Badge,
        Button,
        Table,
        TableBody,
        TableBodyCell,
        TableBodyRow,
        TableHead,
        TableHeadCell,
    } from "flowbite-svelte";

    let plan: RebalancePlan | null = $state(null);
    let editedPlan: RebalancePlan | null = $state(null);
    let loading = $state(true);
    let executing = $state(false);
    let errorMsg = $state("");
    let lastUpdated = $state("");
    let editMode = $state(false);

    async function loadPreview() {
        loading = true;
        errorMsg = "";
        editMode = false;
        try {
            plan = await fetchRebalancePreview();
            editedPlan = JSON.parse(JSON.stringify(plan)); // Deep copy
            lastUpdated = new Date().toLocaleTimeString();
        } catch (e: any) {
            errorMsg = e.message;
        } finally {
            loading = false;
        }
    }

    function toggleEditMode() {
        editMode = !editMode;
        if (editMode && plan) {
            editedPlan = JSON.parse(JSON.stringify(plan)); // Reset to original
        }
    }

    function updateQuantity(index: number, newQty: number) {
        if (!editedPlan) return;

        const item = editedPlan.items[index];
        item.target_qty = Math.max(0, Math.floor(newQty));
        item.target_val = item.target_qty * item.current_price;

        // Recalculate action
        if (item.target_qty > item.current_qty) {
            item.action = "BUY";
            item.action_qty = item.target_qty - item.current_qty;
        } else if (item.target_qty < item.current_qty) {
            item.action = "SELL";
            item.action_qty = item.current_qty - item.target_qty;
        } else {
            item.action = "HOLD";
            item.action_qty = 0;
        }
    }

    async function handleApproval(dryRun: boolean) {
        if (!editedPlan) return;

        const confirmMsg = dryRun
            ? "Run simulation with this plan?"
            : `⚠️ WARNING: Execute REAL TRADES?\n\n${getSummary()}`;

        if (!confirm(confirmMsg)) return;

        executing = true;
        try {
            await executeCustomRebalance(editedPlan, dryRun);
            alert(
                dryRun
                    ? "✓ Dry Run Completed. Check logs."
                    : "✓ Trades Executed Successfully!",
            );
            await loadPreview();
        } catch (e: any) {
            alert("Execution failed: " + e.message);
        } finally {
            executing = false;
        }
    }

    function getSummary(): string {
        if (!editedPlan) return "";
        const buys = editedPlan.items.filter((i) => i.action === "BUY");
        const sells = editedPlan.items.filter((i) => i.action === "SELL");
        return `BUY: ${buys.map((i) => `${i.symbol}(${i.action_qty})`).join(", ")}\nSELL: ${sells.map((i) => `${i.symbol}(${i.action_qty})`).join(", ")}`;
    }

    onMount(() => {
        loadPreview();
    });
</script>

<div class="p-8 max-w-7xl mx-auto">
    <!-- Header -->
    <div class="flex justify-between items-center mb-8">
        <div>
            <h1 class="text-3xl font-bold text-white mb-2">
                Portfolio Rebalance (V2)
            </h1>
            <p class="text-slate-400">
                Monthly Strategy: TQQQ, PFIX, SCHD, TMF with MA130 Logic
            </p>
        </div>
        <div class="flex gap-3">
            <Button
                color="light"
                onclick={loadPreview}
                disabled={loading || executing}
            >
                {#if loading}
                    <Spinner size="4" class="mr-2" />
                {/if}
                Refresh Preview
            </Button>
            {#if !editMode}
                <Button
                    color="blue"
                    onclick={toggleEditMode}
                    disabled={loading || executing}
                >
                    ✏️ Edit Plan
                </Button>
            {:else}
                <Button
                    color="light"
                    onclick={toggleEditMode}
                    disabled={executing}
                >
                    Cancel
                </Button>
                <Button
                    color="purple"
                    onclick={() => handleApproval(true)}
                    disabled={executing}
                >
                    Simulate (Dry Run)
                </Button>
                <Button
                    color="red"
                    onclick={() => handleApproval(false)}
                    disabled={executing}
                >
                    ✅ Approve & Execute
                </Button>
            {/if}
        </div>
    </div>

    {#if errorMsg}
        <div class="p-4 mb-4 text-red-500 bg-red-100 rounded-lg">
            {errorMsg}
        </div>
    {/if}

    {#if plan}
        <!-- Stats Cards -->
        <div class="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
            <div class="stat-card bg-slate-800 p-6 rounded-lg shadow-lg">
                <div class="text-slate-400 text-sm">Total Equity</div>
                <div class="text-3xl font-bold text-white">
                    ${plan.total_value.toFixed(2)}
                </div>
            </div>
            <div class="stat-card bg-slate-800 p-6 rounded-lg shadow-lg">
                <div class="text-slate-400 text-sm">Available Cash</div>
                <div class="text-3xl font-bold text-green-400">
                    ${plan.cash.toFixed(2)}
                </div>
            </div>
            <div
                class="stat-card bg-slate-800 p-6 rounded-lg shadow-lg border border-red-900/30"
            >
                <div class="text-slate-400 text-sm">Est. Tax Impact</div>
                <div class="text-3xl font-bold text-red-400">
                    ${plan.estimated_tax.toFixed(2)}
                </div>
            </div>
        </div>

        <!-- Main Table -->
        <div class="bg-slate-800 rounded-lg shadow-lg overflow-hidden">
            <Table hoverable={true}>
                <TableHead>
                    <TableHeadCell>Symbol</TableHeadCell>
                    <TableHeadCell>Price / MA130</TableHeadCell>
                    <TableHeadCell>Conditions</TableHeadCell>
                    <TableHeadCell>Weight (Target)</TableHeadCell>
                    <TableHeadCell
                        >Target Qty {editMode ? "(Edit)" : ""}</TableHeadCell
                    >
                    <TableHeadCell>Action</TableHeadCell>
                </TableHead>
                <TableBody>
                    {#each editedPlan?.items || [] as item, index}
                        <TableBodyRow class="border-b border-slate-700">
                            <TableBodyCell
                                class="font-bold text-lg text-blue-400"
                            >
                                {item.symbol}
                                {#if item.kill_switch}
                                    <Badge color="red" class="ml-2">KILL</Badge>
                                {/if}
                            </TableBodyCell>

                            <TableBodyCell>
                                <div class="text-white">
                                    ${item.current_price.toFixed(2)}
                                </div>
                                <div class="text-xs text-slate-400">
                                    MA130: ${item.ma_130.toFixed(2)}
                                </div>
                            </TableBodyCell>

                            <TableBodyCell>
                                <div class="flex gap-2">
                                    {#if item.cond_price_under_ma}
                                        <Badge color="yellow"
                                            >Price &lt; MA</Badge
                                        >
                                    {:else}
                                        <Badge color="green"
                                            >Price &gt; MA</Badge
                                        >
                                    {/if}
                                    {#if item.cond_ma_down}
                                        <Badge color="red">MA &darr;</Badge>
                                    {:else}
                                        <Badge color="green">MA &uarr;</Badge>
                                    {/if}
                                </div>
                            </TableBodyCell>

                            <TableBodyCell>
                                <div class="text-white font-mono">
                                    {(item.target_wt * 100).toFixed(1)}%
                                </div>
                                <div class="text-xs text-slate-500">
                                    Curr: {(item.current_wt * 100).toFixed(1)}%
                                </div>
                            </TableBodyCell>

                            <TableBodyCell>
                                {#if editMode}
                                    <input
                                        type="number"
                                        value={item.target_qty}
                                        oninput={(e) =>
                                            updateQuantity(
                                                index,
                                                parseInt(
                                                    e.currentTarget.value,
                                                ) || 0,
                                            )}
                                        class="w-20 px-2 py-1 bg-slate-700 text-white border border-slate-600 rounded"
                                    />
                                {:else}
                                    <div class="text-white">
                                        {item.target_qty}
                                    </div>
                                {/if}
                                <div class="text-xs text-slate-500">
                                    Curr: {item.current_qty}
                                </div>
                            </TableBodyCell>

                            <TableBodyCell>
                                {#if item.action === "BUY"}
                                    <span class="text-green-400 font-bold"
                                        >BUY {item.action_qty}</span
                                    >
                                {:else if item.action === "SELL"}
                                    <span class="text-red-400 font-bold"
                                        >SELL {item.action_qty}</span
                                    >
                                {:else}
                                    <span class="text-slate-500">HOLD</span>
                                {/if}
                            </TableBodyCell>
                        </TableBodyRow>
                    {/each}
                </TableBody>
            </Table>
        </div>
        <div class="mt-4 text-right text-slate-500 text-sm">
            Last Updated: {lastUpdated}
        </div>
    {:else if !loading}
        <div class="text-center text-slate-400 mt-12">
            No plan data available.
        </div>
    {/if}

    {#if loading}
        <div class="text-center mt-12">
            <Spinner size="8" />
            <p class="mt-4 text-slate-400">Calculating Rebalance Plan...</p>
        </div>
    {/if}
</div>
