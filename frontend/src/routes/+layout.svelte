<script lang="ts">
	import favicon from "$lib/assets/favicon.svg";
	import "../app.css";
	import {
		Navbar,
		NavBrand,
		NavLi,
		NavUl,
		NavHamburger,
	} from "flowbite-svelte";
	import { page } from "$app/stores";

	let { children } = $props();
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
</svelte:head>

<Navbar
	class="px-6 py-4 fixed w-full z-20 top-0 left-0 border-b border-slate-800/50 glass-card"
>
	<NavBrand href="/">
		<svg
			class="w-8 h-8 mr-2"
			viewBox="0 0 24 24"
			fill="none"
			xmlns="http://www.w3.org/2000/svg"
		>
			<path d="M12 2L2 7L12 12L22 7L12 2Z" fill="#3b82f6" opacity="0.8" />
			<path d="M2 17L12 22L22 17" stroke="#3b82f6" stroke-width="2" />
			<path d="M2 12L12 17L22 12" stroke="#60a5fa" stroke-width="2" />
		</svg>
		<span
			class="self-center whitespace-nowrap text-xl font-bold bg-gradient-to-r from-blue-400 to-blue-600 bg-clip-text text-transparent"
			>TQQQ Trader</span
		>
	</NavBrand>
	<NavHamburger />
	<NavUl class="gap-4">
		<NavLi
			href="/"
			active={$page.url.pathname === "/"}
			class="text-slate-300 hover:text-blue-400 transition-colors font-medium"
			>Dashboard</NavLi
		>
		<NavLi
			href="/logs"
			active={$page.url.pathname.startsWith("/logs")}
			class="text-slate-300 hover:text-blue-400 transition-colors font-medium"
			>Logs</NavLi
		>
		<NavLi
			href="/settings"
			active={$page.url.pathname.startsWith("/settings")}
			class="text-slate-300 hover:text-blue-400 transition-colors font-medium"
			>Settings</NavLi
		>
	</NavUl>
</Navbar>

<div class="mt-20 container mx-auto p-6 min-h-screen">
	{@render children()}
</div>
