<script lang="ts">
	import { _ } from 'svelte-i18n';
	import { goto } from '$app/navigation';
	import { ticketsApi, type Ticket, type TicketStatus } from '$lib/api/tickets';
	import { can } from '$lib/permissions';

	let tickets: Ticket[] = [];
	let loading = true;
	let error = '';
	let searchQuery = '';
	let activeStatus: TicketStatus | '' = '';

	const tabs: Array<{ key: TicketStatus | ''; label: string }> = [
		{ key: '', label: 'tickets.tabs.all' },
		{ key: 'open', label: 'tickets.tabs.open' },
		{ key: 'pending', label: 'tickets.tabs.pending' },
		{ key: 'resolved', label: 'tickets.tabs.resolved' }
	];

	async function loadTickets() {
		loading = true;
		error = '';
		try {
			const result = await ticketsApi.list({
				status: activeStatus || undefined,
				q: searchQuery || undefined
			});
			tickets = Array.isArray(result) ? result : [];
		} catch (e: unknown) {
			error = 'Não foi possível carregar os tickets.';
		} finally {
			loading = false;
		}
	}

	$: activeStatus, searchQuery, loadTickets();

	function statusClass(status: TicketStatus) {
		return `status-badge status-${status.replace('_', '-')}`;
	}

	function priorityClass(priority: string) {
		return `status-badge priority-${priority}`;
	}

	function formatDate(iso: string): string {
		const diffMs = Date.now() - new Date(iso).getTime();
		const diffSec = Math.floor(diffMs / 1000);
		if (diffSec < 60) return `${diffSec} seg`;
		const diffMin = Math.floor(diffSec / 60);
		if (diffMin < 60) return `${diffMin} min`;
		const diffHour = Math.floor(diffMin / 60);
		if (diffHour < 6) return `${diffHour}h`;
		const now = new Date();
		const d = new Date(iso);
		const todayStr = now.toDateString();
		const yesterdayStr = new Date(now.getTime() - 86_400_000).toDateString();
		if (d.toDateString() === todayStr) return 'hoje';
		if (d.toDateString() === yesterdayStr) return 'ontem';
		const diffDay = Math.floor(diffMs / 86_400_000);
		if (diffDay < 30) return `${diffDay} dias`;
		return new Intl.DateTimeFormat('pt-BR', { day: '2-digit', month: 'short' }).format(d);
	}
</script>

<svelte:head><title>Tickets — NovuDesk</title></svelte:head>

<div class="p-8 max-w-6xl mx-auto">
	<!-- Header -->
	<div class="flex items-center justify-between mb-6">
		<div>
			<h1 class="text-2xl font-bold">{$_('tickets.title')}</h1>
			<p class="text-base-content/50 text-sm mt-0.5">
				{tickets.length} ticket{tickets.length !== 1 ? 's' : ''}
			</p>
		</div>
		{#if can('tickets:create')}
			<a href="/tickets/new" class="btn btn-primary btn-sm gap-2">
				<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
				</svg>
				{$_('tickets.new')}
			</a>
		{/if}
	</div>

	<!-- Filters -->
	<div class="flex flex-col sm:flex-row gap-3 mb-5">
		<!-- Search -->
		<div class="relative flex-1">
			<svg class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-base-content/40" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
			</svg>
			<input
				type="search"
				bind:value={searchQuery}
				placeholder={$_('tickets.search')}
				class="input input-bordered w-full pl-9 text-sm h-9"
			/>
		</div>

		<!-- Status tabs -->
		<div class="tabs tabs-boxed bg-base-200 p-1 h-9 items-center">
			{#each tabs as tab}
				<button
					class="tab tab-sm h-7 text-xs"
					class:tab-active={activeStatus === tab.key}
					on:click={() => (activeStatus = tab.key)}
				>
					{$_(tab.label)}
				</button>
			{/each}
		</div>
	</div>

	<!-- Ticket list -->
	<div class="card bg-base-100 shadow-card overflow-hidden">
		{#if loading}
			<div class="flex items-center justify-center py-16">
				<span class="loading loading-spinner loading-md text-primary"></span>
			</div>
		{:else if error}
			<div class="flex items-center justify-center py-16 text-error text-sm">{error}</div>
		{:else if tickets.length === 0}
			<div class="flex flex-col items-center justify-center py-16 text-base-content/40">
				<svg class="w-10 h-10 mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5"
						d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
				</svg>
				<p class="text-sm font-medium">{$_('tickets.noTickets')}</p>
			</div>
		{:else}
			<table class="table table-sm">
				<thead class="bg-base-50">
					<tr class="text-xs text-base-content/50 uppercase tracking-wide">
						<th class="w-12">{$_('tickets.number')}</th>
						<th>{$_('tickets.subject')}</th>
						<th class="w-28">{$_('tickets.priorityLabel')}</th>
						<th class="w-28">{$_('tickets.statusLabel')}</th>
						<th class="w-32 hidden md:table-cell">{$_('tickets.updated')}</th>
					</tr>
				</thead>
				<tbody>
					{#each tickets as ticket}
						<tr
							class="hover:bg-base-50 cursor-pointer transition-colors"
							on:click={() => goto(`/tickets/${ticket.id}`)}
						>
							<td class="font-mono text-xs text-base-content/40">#{ticket.number}</td>
							<td>
								<div class="flex items-center gap-2">
									{#if ticket.sla_breached}
										<span class="tooltip tooltip-right" data-tip="SLA violado">
											<span class="w-1.5 h-1.5 rounded-full bg-error shrink-0 block"></span>
										</span>
									{/if}
									<span class="font-medium text-sm truncate max-w-xs">{ticket.title}</span>
									{#if ticket.tags.length > 0}
										{#each ticket.tags.slice(0, 2) as tag}
											<span class="badge badge-ghost badge-xs">{tag}</span>
										{/each}
									{/if}
								</div>
							</td>
							<td>
								<span class={priorityClass(ticket.priority)}>
									{$_(`tickets.priority.${ticket.priority}`)}
								</span>
							</td>
							<td>
								<span class={statusClass(ticket.status)}>
									{$_(`tickets.status.${ticket.status}`)}
								</span>
							</td>
							<td class="text-xs text-base-content/40 hidden md:table-cell">
								{formatDate(ticket.updated_at)}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		{/if}
	</div>
</div>
