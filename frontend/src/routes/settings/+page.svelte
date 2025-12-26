<script lang="ts">
    import { onMount } from "svelte";
    import { fetchSettings, updateSettings, type UserSettings } from "$lib/api";
    import {
        Card,
        Label,
        Input,
        Button,
        Toggle,
        Heading,
    } from "flowbite-svelte";

    let settings: UserSettings = $state({
        Principal: 10000,
        SplitCount: 40,
        TargetRate: 0.1,
        Symbols: "TQQQ",
        IsActive: false,
    });
    let loading = $state(true);
    let saving = $state(false);

    async function load() {
        loading = true;
        try {
            settings = await fetchSettings();
        } catch (e) {
            console.error(e);
        } finally {
            loading = false;
        }
    }

    async function save() {
        saving = true;
        try {
            await updateSettings(settings);
            alert("Settings saved successfully");
        } catch (e) {
            alert("Failed to save settings");
        } finally {
            saving = false;
        }
    }

    onMount(() => {
        load();
    });
</script>

<div class="max-w-3xl mx-auto p-8">
    <div class="mb-8">
        <h1 class="text-4xl font-bold text-white mb-2">Strategy Settings</h1>
        <p class="text-slate-400">
            Configure your infinite buying strategy parameters
        </p>
    </div>

    <div class="stat-card">
        <form
            class="flex flex-col space-y-6"
            onsubmit={(e) => {
                e.preventDefault();
                save();
            }}
        >
            <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div class="space-y-2">
                    <label class="text-sm font-medium text-slate-300"
                        >Principal Amount ($)</label
                    >
                    <input
                        type="number"
                        bind:value={settings.Principal}
                        required
                        class="input-field w-full"
                        placeholder="10000"
                    />
                    <p class="text-xs text-slate-500">
                        Total capital for the strategy
                    </p>
                </div>

                <div class="space-y-2">
                    <label class="text-sm font-medium text-slate-300"
                        >Split Count</label
                    >
                    <input
                        type="number"
                        bind:value={settings.SplitCount}
                        required
                        class="input-field w-full"
                        placeholder="40"
                    />
                    <p class="text-xs text-slate-500">
                        Number of parts to divide principal
                    </p>
                </div>
            </div>

            <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div class="space-y-2">
                    <label class="text-sm font-medium text-slate-300"
                        >Target Profit Rate</label
                    >
                    <input
                        type="number"
                        step="0.01"
                        bind:value={settings.TargetRate}
                        required
                        class="input-field w-full"
                        placeholder="0.10"
                    />
                    <p class="text-xs text-slate-500">
                        Decimal format (0.10 = 10%)
                    </p>
                </div>

                <div class="space-y-2">
                    <label class="text-sm font-medium text-slate-300"
                        >Trading Symbols</label
                    >
                    <input
                        type="text"
                        bind:value={settings.Symbols}
                        required
                        class="input-field w-full"
                        placeholder="TQQQ,SOXL"
                    />
                    <p class="text-xs text-slate-500">
                        Comma-separated ticker symbols
                    </p>
                </div>
            </div>

            <div
                class="flex items-center justify-between p-4 bg-slate-800/30 rounded-lg border border-slate-700"
            >
                <div>
                    <p class="text-white font-medium">Strategy Activation</p>
                    <p class="text-sm text-slate-400">
                        Enable automated trading execution
                    </p>
                </div>
                <Toggle
                    bind:checked={settings.IsActive}
                    color="blue"
                    class="ml-4"
                >
                    {settings.IsActive ? "Active" : "Inactive"}
                </Toggle>
            </div>

            <div class="flex gap-4 pt-4">
                <button
                    type="submit"
                    disabled={saving || loading}
                    class="btn-primary flex-1"
                >
                    {#if saving}
                        <svg
                            class="animate-spin h-5 w-5 mr-2 inline-block"
                            xmlns="http://www.w3.org/2000/svg"
                            fill="none"
                            viewBox="0 0 24 24"
                        >
                            <circle
                                class="opacity-25"
                                cx="12"
                                cy="12"
                                r="10"
                                stroke="currentColor"
                                stroke-width="4"
                            ></circle>
                            <path
                                class="opacity-75"
                                fill="currentColor"
                                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                            ></path>
                        </svg>
                        Saving...
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
                                d="M5 13l4 4L19 7"
                            />
                        </svg>
                        Save Settings
                    {/if}
                </button>
            </div>
        </form>
    </div>

    <div class="mt-6 p-4 bg-blue-500/10 border border-blue-500/30 rounded-lg">
        <div class="flex">
            <svg
                class="w-5 h-5 text-blue-400 mr-2 flex-shrink-0 mt-0.5"
                fill="currentColor"
                viewBox="0 0 20 20"
            >
                <path
                    fill-rule="evenodd"
                    d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z"
                    clip-rule="evenodd"
                />
            </svg>
            <p class="text-sm text-blue-300">
                <strong>Tip:</strong> Changes will take effect on the next scheduled
                execution. Make sure to configure your KIS API credentials in the
                environment variables.
            </p>
        </div>
    </div>
</div>
