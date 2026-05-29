<script lang="ts">
	import { _ } from 'svelte-i18n';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { ticketsApi, type Ticket, type TicketStatus, type TicketPriority, type TicketSort } from '$lib/api/tickets';
	import { can } from '$lib/permissions';
	import { pollingInterval } from '$lib/stores/polling';
	import PollingControl from '$lib/components/PollingControl.svelte';

	let tickets: Ticket[] = [];
	let total = 0;
	let loading = true;
	let error = '';

	let searchQuery = '';
	let filterStatus: TicketStatus[] = [];
	let filterPriority: TicketPriority[] = [];
	let filterSlaBreached = false;
	let sortBy: TicketSort = 'created_at';

	let showFilterMenu = false;

	const allStatuses: TicketStatus[] = ['open', 'pending', 'on_hold', 'resolved', 'closed'];
	const allPriorities: TicketPriority[] = ['low', 'normal', 'high', 'urgent'];

	function toggleStatus(s: TicketStatus) {
		filterStatus = filterStatus.includes(s)
			? filterStatus.filter((x) => x !== s)
			: [...filterStatus, s];
	}

	function togglePriority(p: TicketPriority) {
		filterPriority = filterPriority.includes(p)
			? filterPriority.filter((x) => x !== p)
			: [...filterPriority, p];
	}

	function clearAll() {
		filterStatus = [];
		filterPriority = [];
		filterSlaBreached = false;
		sortBy = 'created_at';
		searchQuery = '';
		showFilterMenu = false;
	}

	$: hasActiveFilters =
		filterStatus.length > 0 || filterPriority.length > 0 || filterSlaBreached || sortBy !== 'created_at';

	async function loadTickets(silent = false) {
		if (!silent) { loading = true; error = ''; }
		try {
			const q = searchQuery.trim();
			const params: Parameters<typeof ticketsApi.list>[0] = { sort: sortBy };

			// Smart search: detect #number or UUID, else full-text
			if (/^\d+$/.test(q)) {
				params.number = parseInt(q, 10);
			} else if (/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(q)) {
				// UUID search — pass as q (backend will handle ID lookup via full-text fallback)
				params.q = q;
			} else if (q) {
				params.q = q;
			}

			if (filterStatus.length > 0) params.status = filterStatus;
			if (filterPriority.length > 0) params.priority = filterPriority;
			if (filterSlaBreached) params.sla_breached = true;

			const result = await ticketsApi.list(params);
			tickets = result.data;
			total = result.total;
		} catch {
			if (!silent) error = 'Não foi possível carregar os tickets.';
		} finally {
			loading = false;
		}
	}

	$: searchQuery, filterStatus, filterPriority, filterSlaBreached, sortBy, loadTickets();

	let timer: ReturnType<typeof setInterval> | null = null;
	$: {
		if (timer) clearInterval(timer);
		timer = $pollingInterval > 0 ? setInterval(() => loadTickets(true), $pollingInterval) : null;
	}

	onMount(() => () => { if (timer) clearInterval(timer); });

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

	function formatSLADue(iso: string | null): string {
		if (!iso) return '';
		const d = new Date(iso);
		const now = new Date();
		const diffMs = d.getTime() - now.getTime();
		if (diffMs < 0) return 'Vencido';
		const diffHour = Math.floor(diffMs / 3600000);
		if (diffHour < 1) return `< 1h`;
		if (diffHour < 24) return `${diffHour}h`;
		return `${Math.floor(diffHour / 24)}d`;
	}
</script>

<svelte:head><title>Tickets — NovuDesk</title></svelte:head>

<!-- svelte-ignore a11y-no-static-element-interactions -->
<div class="p-8 max-w-6xl mx-auto" on:click={() => (showFilterMenu = false)}>
	<!-- Header -->
	<div class="flex items-center justify-between mb-6">
		<div>
			<h1 class="text-2xl font-bold">{$_('tickets.title')}</h1>
			<p class="text-base-content/50 text-sm mt-0.5">
				{total} ticket{total !== 1 ? 's' : ''}
			</p>
		</div>
		<div class="flex items-center gap-3">
			<PollingControl />
			{#if can('tickets:create')}
				<a href="/tickets/new" class="btn btn-primary btn-sm gap-2">
					<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
					</svg>
					{$_('tickets.new')}
				</a>
			{/if}
		</div>
	</div>

	<!-- Search + Filter bar -->
	<div class="flex flex-col sm:flex-row gap-3 mb-3">
		<!-- Search -->
		<div class="relative flex-1">
			<svg class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-base-content/40" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
			</svg>
			<input
				type="search"
				bind:value={searchQuery}
				placeholder={$_('tickets.filters.searchPlaceholder')}
				class="input input-bordered w-full pl-9 text-sm h-9"
			/>
		</div>

		<!-- Filter dropdown trigger -->
		<!-- svelte-ignore a11y-no-static-element-interactions -->
		<div class="relative" on:click|stopPropagation>
			<button
				class="btn btn-sm h-9 gap-2"
				class:btn-primary={hasActiveFilters}
				class:btn-ghost={!hasActiveFilters}
				class:btn-outline={!hasActiveFilters}
				on:click={() => (showFilterMenu = !showFilterMenu)}
			>
				<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 4a1 1 0 011-1h16a1 1 0 011 1v2a1 1 0 01-.293.707L13 13.414V19a1 1 0 01-.553.894l-4 2A1 1 0 017 21v-7.586L3.293 6.707A1 1 0 013 6V4z" />
				</svg>
				{$_('tickets.filters.add')}
				{#if hasActiveFilters}
					<span class="badge badge-sm badge-warning">
						{filterStatus.length + filterPriority.length + (filterSlaBreached ? 1 : 0)}
					</span>
				{/if}
			</button>

			{#if showFilterMenu}
				<div class="absolute right-0 top-full mt-1 z-50 bg-base-100 border border-base-200 rounded-xl shadow-xl w-72 p-4 space-y-4">
					<!-- Status -->
					<div>
						<p class="text-xs font-semibold text-base-content/60 uppercase tracking-wide mb-2">{$_('tickets.filters.filterStatus')}</p>
						<div class="flex flex-wrap gap-1.5">
							{#each allStatuses as s}
								<button
									class="btn btn-xs"
									class:btn-primary={filterStatus.includes(s)}
									class:btn-ghost={!filterStatus.includes(s)}
									on:click={() => toggleStatus(s)}
								>
									{$_(`tickets.status.${s}`)}
								</button>
							{/each}
						</div>
					</div>

					<!-- Priority -->
					<div>
						<p class="text-xs font-semibold text-base-content/60 uppercase tracking-wide mb-2">{$_('tickets.filters.filterPriority')}</p>
						<div class="flex flex-wrap gap-1.5">
							{#each allPriorities as p}
								<button
									class="btn btn-xs"
									class:btn-primary={filterPriority.includes(p)}
									class:btn-ghost={!filterPriority.includes(p)}
									on:click={() => togglePriority(p)}
								>
									{$_(`tickets.priority.${p}`)}
								</button>
							{/each}
						</div>
					</div>

					<!-- SLA -->
					<div>
						<p class="text-xs font-semibold text-base-content/60 uppercase tracking-wide mb-2">SLA</p>
						<button
							class="btn btn-xs w-full"
							class:btn-error={filterSlaBreached}
							class:btn-ghost={!filterSlaBreached}
							on:click={() => (filterSlaBreached = !filterSlaBreached)}
						>
							{#if filterSlaBreached}
								<span class="w-1.5 h-1.5 rounded-full bg-white shrink-0"></span>
							{/if}
							{$_('tickets.filters.slaBreached')}
						</button>
					</div>

					<!-- Sort -->
					<div>
						<p class="text-xs font-semibold text-base-content/60 uppercase tracking-wide mb-2">{$_('tickets.filters.sort')}</p>
						<div class="flex flex-col gap-1">
							{#each [
								{ value: 'created_at', label: 'tickets.filters.sortNewest' },
								{ value: 'updated_at', label: 'tickets.filters.sortUpdated' },
								{ value: 'sla_due',    label: 'tickets.filters.sortSlaDue' }
							] as opt}
								<button
									class="btn btn-xs justify-start"
									class:btn-primary={sortBy === opt.value}
									class:btn-ghost={sortBy !== opt.value}
									on:click={() => { sortBy = opt.value as TicketSort; }}
								>
									{$_(opt.label)}
								</button>
							{/each}
						</div>
					</div>

					<div class="border-t border-base-200 pt-3">
						<button class="btn btn-ghost btn-xs w-full" on:click={clearAll}>
							{$_('tickets.filters.clearAll')}
						</button>
					</div>
				</div>
			{/if}
		</div>
	</div>

	<!-- Active filter chips -->
	{#if hasActiveFilters}
		<div class="flex flex-wrap gap-1.5 mb-4">
			{#each filterStatus as s}
				<button
					class="badge badge-primary badge-sm gap-1 cursor-pointer"
					on:click={() => toggleStatus(s)}
				>
					{$_(`tickets.status.${s}`)}
					<span>×</span>
				</button>
			{/each}
			{#each filterPriority as p}
				<button
					class="badge badge-secondary badge-sm gap-1 cursor-pointer"
					on:click={() => togglePriority(p)}
				>
					{$_(`tickets.priority.${p}`)}
					<span>×</span>
				</button>
			{/each}
			{#if filterSlaBreached}
				<button
					class="badge badge-error badge-sm gap-1 cursor-pointer"
					on:click={() => (filterSlaBreached = false)}
				>
					{$_('tickets.filters.slaBreached')} ×
				</button>
			{/if}
			{#if sortBy !== 'created_at'}
				<span class="badge badge-ghost badge-sm">
					↕ {$_(`tickets.filters.sort${sortBy === 'updated_at' ? 'Updated' : 'SlaDue'}`)}
				</span>
			{/if}
		</div>
	{/if}

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
						{#if sortBy === 'sla_due'}
							<th class="w-28 hidden md:table-cell">Prazo SLA</th>
						{:else}
							<th class="w-32 hidden md:table-cell">{$_('tickets.updated')}</th>
						{/if}
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
										<span class="tooltip tooltip-right" data-tip={$_('tickets.slaBreached')}>
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
								{#if sortBy === 'sla_due'}
									{#if ticket.sla_resolution_due_at}
										<span class:text-error={ticket.sla_breached}>
											{formatSLADue(ticket.sla_resolution_due_at)}
										</span>
									{:else}
										—
									{/if}
								{:else}
									{formatDate(ticket.updated_at)}
								{/if}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		{/if}
	</div>
</div>
