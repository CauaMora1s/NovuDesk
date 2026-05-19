<script lang="ts">
	import { _ } from 'svelte-i18n';
	import { ticketsApi } from '$lib/api/tickets';
	import type { Ticket } from '$lib/api/tickets';

	let openCount = 0;
	let slaBreachedCount = 0;
	let recentTickets: Ticket[] = [];
	let loading = true;

	async function loadDashboard() {
		loading = true;
		try {
			const [open, _breached, recent] = await Promise.all([
				ticketsApi.list({ status: 'open', per_page: 1 }),
				ticketsApi.list({ per_page: 1 }),
				ticketsApi.list({ per_page: 5 })
			]);
			openCount = Array.isArray(open) ? open.length : 0;
			recentTickets = Array.isArray(recent) ? recent : [];
		} catch {
			// handled gracefully
		} finally {
			loading = false;
		}
	}

	// Fires once at component init — no onMount needed
	$: loadDashboard();

	const stats = [
		{ label: 'dashboard.openTickets',   value: () => openCount,        color: 'text-info' },
		{ label: 'dashboard.slaBreached',   value: () => slaBreachedCount, color: 'text-error' },
		{ label: 'dashboard.resolvedToday', value: () => 0,                color: 'text-success' }
	];

	function formatDate(iso: string) {
		return new Intl.DateTimeFormat(undefined, {
			day: '2-digit', month: 'short', hour: '2-digit', minute: '2-digit'
		}).format(new Date(iso));
	}
</script>

<svelte:head><title>Painel — NovuDesk</title></svelte:head>

<div class="p-8 max-w-6xl mx-auto">
	<!-- Page header -->
	<div class="mb-8">
		<h1 class="text-2xl font-bold">{$_('dashboard.title')}</h1>
		<p class="text-base-content/50 text-sm mt-1">Visão geral do seu atendimento</p>
	</div>

	<!-- Stats row -->
	<div class="grid grid-cols-1 sm:grid-cols-3 gap-4 mb-8">
		{#each stats as stat}
			<div class="card bg-base-100 shadow-card">
				<div class="card-body p-5">
					<p class="text-sm text-base-content/60 font-medium">{$_(stat.label)}</p>
					<p class="text-3xl font-bold mt-1 {stat.color}">
						{#if loading}
							<span class="loading loading-ring loading-sm"></span>
						{:else}
							{stat.value()}
						{/if}
					</p>
				</div>
			</div>
		{/each}
	</div>

	<!-- Recent tickets -->
	<div class="card bg-base-100 shadow-card">
		<div class="card-body p-0">
			<div class="flex items-center justify-between px-6 py-4 border-b border-base-200">
				<h2 class="font-semibold text-base">Tickets recentes</h2>
				<a href="/tickets" class="btn btn-ghost btn-xs">Ver todos →</a>
			</div>

			{#if loading}
				<div class="flex items-center justify-center py-12">
					<span class="loading loading-spinner loading-md text-primary"></span>
				</div>
			{:else if recentTickets.length === 0}
				<div class="flex flex-col items-center justify-center py-12 text-base-content/40">
					<svg class="w-10 h-10 mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5"
							d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
					</svg>
					<p class="text-sm">{$_('tickets.noTickets')}</p>
				</div>
			{:else}
				<ul class="divide-y divide-base-200">
					{#each recentTickets as ticket}
						<li>
							<a href="/tickets/{ticket.id}" class="flex items-center gap-4 px-6 py-3.5 hover:bg-base-50 transition-colors">
								<span class="text-xs text-base-content/40 font-mono w-10 shrink-0">#{ticket.number}</span>
								<span class="flex-1 text-sm font-medium truncate">{ticket.title}</span>
								<span class="status-badge status-{ticket.status.replace('_', '-')} shrink-0">
									{$_(`tickets.status.${ticket.status}`)}
								</span>
								<span class="text-xs text-base-content/40 shrink-0 hidden sm:block">
									{formatDate(ticket.updated_at)}
								</span>
							</a>
						</li>
					{/each}
				</ul>
			{/if}
		</div>
	</div>
</div>
