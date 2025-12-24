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
	class="px-4 py-2.5 fixed w-full z-20 top-0 left-0 border-b border-gray-200 dark:border-gray-600 bg-white dark:bg-gray-900"
>
	<NavBrand href="/">
		<span
			class="self-center whitespace-nowrap text-xl font-semibold dark:text-white"
			>Infinite Trader</span
		>
	</NavBrand>
	<NavHamburger />
	<NavUl>
		<NavLi href="/" active={$page.url.pathname === "/"}>Dashboard</NavLi>
		<NavLi href="/logs" active={$page.url.pathname.startsWith("/logs")}
			>Logs</NavLi
		>
		<NavLi
			href="/settings"
			active={$page.url.pathname.startsWith("/settings")}>Settings</NavLi
		>
	</NavUl>
</Navbar>

<div class="mt-16 container mx-auto p-4">
	{@render children()}
</div>
