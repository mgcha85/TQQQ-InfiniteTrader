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
            await updateSettings(settings); // Svelte 5 proxy state automatically unwraps or is compatible? Usually yes, or use $state.snapshot if deep.
            // But JS fetch JSON.stringify handles it.
            alert("Settings saved");
        } catch (e) {
            alert("Failed to save");
        } finally {
            saving = false;
        }
    }

    onMount(() => {
        load();
    });
</script>

<div class="max-w-2xl mx-auto">
    <Heading tag="h2" class="mb-6">Settings</Heading>

    <Card>
        <form
            class="flex flex-col space-y-6"
            onsubmit={(e) => {
                e.preventDefault();
                save();
            }}
        >
            <Label class="space-y-2">
                <span>Principal Amount ($)</span>
                <Input type="number" bind:value={settings.Principal} required />
            </Label>

            <Label class="space-y-2">
                <span>Split Count (Default 40)</span>
                <Input
                    type="number"
                    bind:value={settings.SplitCount}
                    required
                />
            </Label>

            <Label class="space-y-2">
                <span>Target Profit Rate (0.10 = 10%)</span>
                <Input
                    type="number"
                    step="0.01"
                    bind:value={settings.TargetRate}
                    required
                />
            </Label>

            <Label class="space-y-2">
                <span>Trading Symbols (Comma separated, e.g. TQQQ,SOXL)</span>
                <Input
                    type="text"
                    bind:value={settings.Symbols}
                    placeholder="TQQQ"
                    required
                />
            </Label>

            <Toggle bind:checked={settings.IsActive} color="green">
                Activate Strategy
            </Toggle>

            <Button type="submit" disabled={saving || loading}>
                {saving ? "Saving..." : "Save Settings"}
            </Button>
        </form>
    </Card>
</div>
