<script lang="ts">
	import { page } from '$app/stores';
	import { _ } from 'svelte-i18n';
	import { can } from '$lib/permissions';
	import { authStore } from '$lib/stores/auth';
	import { goto } from '$app/navigation';
	import { api } from '$lib/api/client';

	$: currentPath = $page.url.pathname;

	$: canViewTeams = can('teams:view') && (
		$authStore.role === 'owner' ||
		$authStore.role === 'admin' ||
		($authStore.teamIds?.length ?? 0) > 0
	);

	function isActive(path: string) {
		return currentPath.startsWith(path);
	}

	async function logout() {
		await api.post('/api/v1/auth/logout', {}).catch(() => {});
		authStore.clear();
		goto('/login');
	}

	const navItems = [
		{
			href: '/dashboard',
			label: 'nav.dashboard',
			permission: null,
			icon: `<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.75"
				d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6" />`
		},
		{
			href: '/tickets',
			label: 'nav.tickets',
			permission: 'tickets:view',
			icon: `<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.75"
				d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4" />`
		},
		{
			href: '/teams',
			label: 'nav.teams',
			permission: 'teams:view',
			icon: `<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.75"
				d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0z" />`
		},
		{
			href: '/settings',
			label: 'nav.settings',
			permission: 'organization:view_settings',
			icon: `<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.75"
				d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" /><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.75" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />`
		}
	] as const;
</script>

<aside class="flex flex-col w-[var(--sidebar-width)] h-screen bg-base-100 border-r border-base-200 shrink-0">
	<!-- Logo -->
	<div class="flex items-center gap-2.5 px-5 py-5 border-b border-base-200">
		<div class="w-7 h-7 bg-primary rounded-lg flex items-center justify-center shrink-0">
			<svg class="w-4 h-4 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
					d="M18 8h1a4 4 0 010 8h-1M2 8h16v9a4 4 0 01-4 4H6a4 4 0 01-4-4V8z" />
			</svg>
		</div>
		<span class="font-bold text-base tracking-tight">NovuDesk</span>
	</div>

	<!-- Navigation -->
	<nav class="flex-1 overflow-y-auto py-4 px-3 space-y-0.5">
		{#each navItems as item}
			{#if item.permission === null || (item.href === '/teams' ? canViewTeams : can(item.permission))}
				<a
					href={item.href}
					class="sidebar-link"
					class:active={isActive(item.href)}
					aria-current={isActive(item.href) ? 'page' : undefined}
				>
					<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						{@html item.icon}
					</svg>
					<span>{$_(item.label)}</span>
				</a>
			{/if}
		{/each}
	</nav>

	<!-- User footer -->
	<div class="border-t border-base-200 p-3">
		<div class="flex items-center gap-3 px-2 py-2">
			<div class="avatar placeholder shrink-0">
				<div class="bg-primary/10 text-primary rounded-full w-8">
					<span class="text-sm font-semibold">
						{($authStore.user?.fullName ?? 'U').charAt(0).toUpperCase()}
					</span>
				</div>
			</div>
			<div class="flex-1 min-w-0">
				<p class="text-sm font-medium truncate">{$authStore.user?.fullName ?? ''}</p>
				<p class="text-xs text-base-content/50 truncate">{$authStore.role ?? ''}</p>
			</div>
			<button
				on:click={logout}
				class="btn btn-ghost btn-xs btn-square"
				title={$_('auth.logout')}
			>
				<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
						d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
				</svg>
			</button>
		</div>
	</div>
</aside>
